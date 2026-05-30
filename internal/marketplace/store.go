package marketplace

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides marketplace data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate marketplace: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mp_listings (
		id TEXT PRIMARY KEY,
		vendor TEXT,
		sku TEXT,
		seats INTEGER,
		orig_price REAL,
		ask_price REAL,
		min_price REAL,
		status TEXT DEFAULT 'active',
		seller_id TEXT,
		created_at TEXT,
		expires_at TEXT
	);

	CREATE TABLE IF NOT EXISTS mp_offers (
		id TEXT PRIMARY KEY,
		listing_id TEXT REFERENCES mp_listings(id),
		buyer_id TEXT,
		seats INTEGER,
		max_price REAL,
		status TEXT DEFAULT 'pending',
		created_at TEXT
	);

	CREATE TABLE IF NOT EXISTS mp_transactions (
		id TEXT PRIMARY KEY,
		listing_id TEXT,
		vendor TEXT,
		sku TEXT,
		seats INTEGER,
		price_per_seat REAL,
		total REAL,
		platform_fee REAL,
		seller_id TEXT,
		buyer_id TEXT,
		status TEXT DEFAULT 'completed',
		created_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_mp_listings_status ON mp_listings(status);
	CREATE INDEX IF NOT EXISTS idx_mp_listings_vendor_sku ON mp_listings(vendor, sku);
	CREATE INDEX IF NOT EXISTS idx_mp_offers_listing ON mp_offers(listing_id);
	CREATE INDEX IF NOT EXISTS idx_mp_transactions_created ON mp_transactions(created_at);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateListing inserts a new listing.
func (s *Store) CreateListing(ctx context.Context, l *Listing) error {
	l.ID = uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mp_listings (id, vendor, sku, seats, orig_price, ask_price, min_price, status, seller_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.ID, l.Vendor, l.SKU, l.Seats, l.OrigPrice, l.AskPrice, l.MinPrice, l.Status,
		l.SellerID, l.CreatedAt.Format(time.RFC3339), l.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create marketplace listing: %w", err)
	}
	return nil
}

// GetListing retrieves a listing by ID.
func (s *Store) GetListing(ctx context.Context, id string) (*Listing, error) {
	var l Listing
	var createdAt, expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, sku, seats, orig_price, ask_price, min_price, status, seller_id, created_at, expires_at
		FROM mp_listings WHERE id = ?
	`, id).Scan(&l.ID, &l.Vendor, &l.SKU, &l.Seats, &l.OrigPrice, &l.AskPrice, &l.MinPrice,
		&l.Status, &l.SellerID, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("marketplace listing not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get marketplace listing: %w", err)
	}
	l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	return &l, nil
}

// SearchListings finds active listings matching vendor and/or sku, ordered by ask_price ASC.
// Empty vendor/sku strings are treated as wildcards.
func (s *Store) SearchListings(ctx context.Context, vendor, sku string, maxSeats int) ([]Listing, error) {
	query := `SELECT id, vendor, sku, seats, orig_price, ask_price, min_price, status, seller_id, created_at, expires_at
		FROM mp_listings WHERE status = 'active'`
	args := []any{}

	if vendor != "" {
		query += " AND vendor = ?"
		args = append(args, vendor)
	}
	if sku != "" {
		query += " AND sku = ?"
		args = append(args, sku)
	}
	if maxSeats > 0 {
		query += " AND seats <= ?"
		args = append(args, maxSeats)
	}
	query += " ORDER BY ask_price ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search listings: %w", err)
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		var createdAt, expiresAt string
		if err := rows.Scan(&l.ID, &l.Vendor, &l.SKU, &l.Seats, &l.OrigPrice, &l.AskPrice, &l.MinPrice,
			&l.Status, &l.SellerID, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

// AddOffer inserts a new offer for a listing.
func (s *Store) AddOffer(ctx context.Context, o *Offer) error {
	o.ID = uuid.New().String()
	o.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mp_offers (id, listing_id, buyer_id, seats, max_price, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.ListingID, o.BuyerID, o.Seats, o.MaxPrice, o.Status, o.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("add offer: %w", err)
	}
	return nil
}

// GetOffers retrieves all offers for a listing, ordered by max_price descending.
func (s *Store) GetOffers(ctx context.Context, listingID string) ([]Offer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, listing_id, buyer_id, seats, max_price, status, created_at
		FROM mp_offers WHERE listing_id = ? ORDER BY max_price DESC, created_at
	`, listingID)
	if err != nil {
		return nil, fmt.Errorf("get offers: %w", err)
	}
	defer rows.Close()

	var offers []Offer
	for rows.Next() {
		var o Offer
		var createdAt string
		if err := rows.Scan(&o.ID, &o.ListingID, &o.BuyerID, &o.Seats, &o.MaxPrice, &o.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan offer: %w", err)
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		offers = append(offers, o)
	}
	return offers, rows.Err()
}

// UpdateOfferStatus updates the status of an offer.
func (s *Store) UpdateOfferStatus(ctx context.Context, offerID, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE mp_offers SET status = ? WHERE id = ?", status, offerID)
	if err != nil {
		return fmt.Errorf("update offer status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("offer not found: %s", offerID)
	}
	return nil
}

// UpdateListingStatus updates the status of a listing.
func (s *Store) UpdateListingStatus(ctx context.Context, listingID, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE mp_listings SET status = ? WHERE id = ?", status, listingID)
	if err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("listing not found: %s", listingID)
	}
	return nil
}

// CreateTransaction inserts a completed transaction.
func (s *Store) CreateTransaction(ctx context.Context, txn *Transaction) error {
	txn.ID = uuid.New().String()
	txn.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mp_transactions (id, listing_id, vendor, sku, seats, price_per_seat, total, platform_fee, seller_id, buyer_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, txn.ID, txn.ListingID, txn.Vendor, txn.SKU, txn.Seats, txn.PricePerSeat, txn.Total, txn.PlatformFee,
		txn.SellerID, txn.BuyerID, txn.Status, txn.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

// GetActiveListings returns all listings with status "active".
func (s *Store) GetActiveListings(ctx context.Context) ([]Listing, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, sku, seats, orig_price, ask_price, min_price, status, seller_id, created_at, expires_at
		FROM mp_listings WHERE status = 'active' ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get active listings: %w", err)
	}
	defer rows.Close()

	var listings []Listing
	for rows.Next() {
		var l Listing
		var createdAt, expiresAt string
		if err := rows.Scan(&l.ID, &l.Vendor, &l.SKU, &l.Seats, &l.OrigPrice, &l.AskPrice, &l.MinPrice,
			&l.Status, &l.SellerID, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan active listing: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		l.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

// GetRecentTransactions returns the most recent transactions.
func (s *Store) GetRecentTransactions(ctx context.Context, limit int) ([]Transaction, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, listing_id, vendor, sku, seats, price_per_seat, total, platform_fee, seller_id, buyer_id, status, created_at
		FROM mp_transactions ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent transactions: %w", err)
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		var createdAt string
		if err := rows.Scan(&t.ID, &t.ListingID, &t.Vendor, &t.SKU, &t.Seats, &t.PricePerSeat, &t.Total,
			&t.PlatformFee, &t.SellerID, &t.BuyerID, &t.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		txns = append(txns, t)
	}
	return txns, rows.Err()
}
