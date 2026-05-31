package vendorreviews

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate vendorreviews: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vendor_reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		rating INTEGER NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) AddReview(ctx context.Context, vendor string, rating int, comment string) (*VendorReview, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_reviews (vendor, rating, comment, created_at)
		VALUES (?, ?, ?, ?)
	`, vendor, rating, comment, now)
	if err != nil {
		return nil, fmt.Errorf("add review: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("add review last insert id: %w", err)
	}
	return &VendorReview{
		ID:        int(id),
		Vendor:    vendor,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: now,
	}, nil
}

func (s *Store) GetVendorReviews(ctx context.Context, vendor string) ([]VendorReview, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, rating, comment, created_at
		FROM vendor_reviews
		WHERE vendor = ?
		ORDER BY created_at DESC
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("get vendor reviews: %w", err)
	}
	defer rows.Close()

	var reviews []VendorReview
	for rows.Next() {
		var r VendorReview
		if err := rows.Scan(&r.ID, &r.Vendor, &r.Rating, &r.Comment, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []VendorReview{}
	}
	return reviews, nil
}

func (s *Store) GetTopVendors(ctx context.Context, category string) ([]map[string]any, error) {
	query := `SELECT vendor, AVG(rating) as avg_rating, COUNT(*) as review_count
		FROM vendor_reviews`
	args := []any{}
	if category != "" {
		query += ` WHERE vendor IN (SELECT name FROM vendors WHERE category = ?)`
		args = append(args, category)
	}
	query += ` GROUP BY vendor ORDER BY avg_rating DESC LIMIT 10`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get top vendors: %w", err)
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var vendor string
		var avgRating float64
		var reviewCount int
		if err := rows.Scan(&vendor, &avgRating, &reviewCount); err != nil {
			return nil, fmt.Errorf("scan top vendor: %w", err)
		}
		results = append(results, map[string]any{
			"vendor":       vendor,
			"avg_rating":   avgRating,
			"review_count": reviewCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}
