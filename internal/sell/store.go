package sell

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides listing and bid data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate sell: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS listings (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		title TEXT,
		description TEXT,
		desired_price REAL,
		min_price REAL,
		strategy TEXT DEFAULT 'fixed',
		status TEXT DEFAULT 'active',
		created_at TEXT,
		expires_at TEXT
	);

	CREATE TABLE IF NOT EXISTS bids (
		id TEXT PRIMARY KEY,
		listing_id TEXT REFERENCES listings(id),
		bidder_id TEXT,
		amount REAL,
		message TEXT,
		created_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_bids_listing ON bids(listing_id);
	CREATE INDEX IF NOT EXISTS idx_listings_status ON listings(status);
	CREATE INDEX IF NOT EXISTS idx_listings_user ON listings(user_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateListing inserts a new listing.
func (s *Store) CreateListing(ctx context.Context, l *Listing) error {
	l.ID = uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO listings (id, user_id, title, description, desired_price, min_price, strategy, status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.ID, l.UserID, l.Title, l.Description, l.DesiredPrice, l.MinPrice, l.Strategy, l.Status,
		l.CreatedAt.Format(time.RFC3339), l.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}
	return nil
}

// GetListing retrieves a listing by ID.
func (s *Store) GetListing(ctx context.Context, id string) (*Listing, error) {
	var l Listing
	var createdAt, expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, description, desired_price, min_price, strategy, status, created_at, expires_at
		FROM listings WHERE id = ?
	`, id).Scan(&l.ID, &l.UserID, &l.Title, &l.Description, &l.DesiredPrice, &l.MinPrice,
		&l.Strategy, &l.Status, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("listing not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get listing: %w", err)
	}
	l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	return &l, nil
}

// ListListings returns active listings, optionally filtered by strategy.
// If strategy is empty, all active listings are returned.
func (s *Store) ListListings(ctx context.Context, strategy string) ([]Listing, error) {
	query := `SELECT id, user_id, title, description, desired_price, min_price, strategy, status, created_at, expires_at
		FROM listings WHERE status = 'active'`
	args := []any{}
	if strategy != "" {
		query += " AND strategy = ?"
		args = append(args, strategy)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		var createdAt, expiresAt string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Title, &l.Description, &l.DesiredPrice, &l.MinPrice,
			&l.Strategy, &l.Status, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

// AddBid inserts a new bid for a listing.
func (s *Store) AddBid(ctx context.Context, b *Bid) error {
	b.ID = uuid.New().String()
	b.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bids (id, listing_id, bidder_id, amount, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, b.ID, b.ListingID, b.BidderID, b.Amount, b.Message, b.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("add bid: %w", err)
	}
	return nil
}

// GetBids retrieves all bids for a listing, ordered by amount descending.
func (s *Store) GetBids(ctx context.Context, listingID string) ([]Bid, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, listing_id, bidder_id, amount, message, created_at
		FROM bids WHERE listing_id = ? ORDER BY amount DESC, created_at
	`, listingID)
	if err != nil {
		return nil, fmt.Errorf("get bids: %w", err)
	}
	defer rows.Close()

	var bids []Bid
	for rows.Next() {
		var b Bid
		var createdAt string
		if err := rows.Scan(&b.ID, &b.ListingID, &b.BidderID, &b.Amount, &b.Message, &createdAt); err != nil {
			return nil, fmt.Errorf("scan bid: %w", err)
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		bids = append(bids, b)
	}
	return bids, rows.Err()
}

// UpdateListingStatus updates the status of a listing.
func (s *Store) UpdateListingStatus(ctx context.Context, listingID, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE listings SET status = ? WHERE id = ?", status, listingID)
	if err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("listing not found: %s", listingID)
	}
	return nil
}

// GetBestBid returns the highest bid for a listing (amount descending).
func (s *Store) GetBestBid(ctx context.Context, listingID string) (*Bid, error) {
	var b Bid
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, listing_id, bidder_id, amount, message, created_at
		FROM bids WHERE listing_id = ? ORDER BY amount DESC, created_at LIMIT 1
	`, listingID).Scan(&b.ID, &b.ListingID, &b.BidderID, &b.Amount, &b.Message, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no bids for listing: %s", listingID)
	}
	if err != nil {
		return nil, fmt.Errorf("get best bid: %w", err)
	}
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &b, nil
}

// GetExpiredListings returns active listings that have passed their expiry.
func (s *Store) GetExpiredListings(ctx context.Context) ([]Listing, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, description, desired_price, min_price, strategy, status, created_at, expires_at
		FROM listings WHERE status IN ('active', 'negotiating') AND expires_at <= ?
	`, now)
	if err != nil {
		return nil, fmt.Errorf("get expired listings: %w", err)
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		var createdAt, expiresAt string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Title, &l.Description, &l.DesiredPrice, &l.MinPrice,
			&l.Strategy, &l.Status, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan expired listing: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		listings = append(listings, l)
	}
	return listings, rows.Err()
}
