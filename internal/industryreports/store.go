package industryreports

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages industry research reports in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates an IndustryReportsStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate industryreports: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS industry_reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SaveReport saves a new industry report and returns it with the generated ID and timestamp.
func (s *Store) SaveReport(ctx context.Context, title, category, content, source string) (*IndustryReport, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO industry_reports (title, category, content, source, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, title, category, content, source, now)
	if err != nil {
		return nil, fmt.Errorf("save report: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("save report last insert id: %w", err)
	}
	return &IndustryReport{
		ID:        int(id),
		Title:     title,
		Category:  category,
		Content:   content,
		Source:    source,
		CreatedAt: now,
	}, nil
}

// ListReports returns all industry reports, optionally filtered by category, ordered by created_at DESC.
func (s *Store) ListReports(ctx context.Context, category string) ([]IndustryReport, error) {
	query := `SELECT id, title, category, content, source, created_at FROM industry_reports`
	args := []any{}
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []IndustryReport
	for rows.Next() {
		var r IndustryReport
		if err := rows.Scan(&r.ID, &r.Title, &r.Category, &r.Content, &r.Source, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if reports == nil {
		reports = []IndustryReport{}
	}
	return reports, nil
}

// GetReport retrieves a single industry report by ID.
func (s *Store) GetReport(ctx context.Context, id int) (*IndustryReport, error) {
	var r IndustryReport
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, category, content, source, created_at
		FROM industry_reports WHERE id = ?
	`, id).Scan(&r.ID, &r.Title, &r.Category, &r.Content, &r.Source, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("report not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}
	return &r, nil
}
