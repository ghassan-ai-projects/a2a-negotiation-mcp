package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides contract data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate calendar: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contracts (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		vendor TEXT,
		sku TEXT,
		seats INTEGER,
		current_price REAL,
		renewal_date TEXT,
		status TEXT DEFAULT 'active',
		last_negotiated_price REAL,
		created_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_contracts_vendor ON contracts(vendor);
	CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts(status);
	CREATE INDEX IF NOT EXISTS idx_contracts_renewal ON contracts(renewal_date);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateContract inserts a new contract.
func (s *Store) CreateContract(ctx context.Context, c *Contract) error {
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now().UTC()
	if c.Status == "" {
		c.Status = "active"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO contracts (id, user_id, vendor, sku, seats, current_price, renewal_date, status, last_negotiated_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.UserID, c.Vendor, c.SKU, c.Seats, c.CurrentPrice,
		c.RenewalDate.Format(time.RFC3339), c.Status, nullableFloat(c.LastNegotiatedPrice),
		c.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create contract: %w", err)
	}
	return nil
}

// GetContract retrieves a contract by ID.
func (s *Store) GetContract(ctx context.Context, id string) (*Contract, error) {
	var c Contract
	var renewalDate, createdAt string
	var lastPrice sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, vendor, sku, seats, current_price, renewal_date, status, last_negotiated_price, created_at
		FROM contracts WHERE id = ?
	`, id).Scan(&c.ID, &c.UserID, &c.Vendor, &c.SKU, &c.Seats, &c.CurrentPrice,
		&renewalDate, &c.Status, &lastPrice, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contract not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get contract: %w", err)
	}
	if lastPrice.Valid {
		c.LastNegotiatedPrice = lastPrice.Float64
	}
	c.RenewalDate, _ = time.Parse(time.RFC3339, renewalDate)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &c, nil
}

// ContractFilter holds optional filters for ListContracts.
type ContractFilter struct {
	Vendor       string
	Status       string
	ExpiringSoon int // if >0, only contracts renewing within this many days
}

// ListContracts returns contracts matching the given filter.
func (s *Store) ListContracts(ctx context.Context, filter ContractFilter) ([]Contract, error) {
	query := "SELECT id, user_id, vendor, sku, seats, current_price, renewal_date, status, last_negotiated_price, created_at FROM contracts WHERE 1=1"
	args := []any{}

	if filter.Vendor != "" {
		query += " AND vendor = ?"
		args = append(args, filter.Vendor)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.ExpiringSoon > 0 {
		query += " AND renewal_date BETWEEN datetime('now') AND datetime('now', '+' || ? || ' days')"
		args = append(args, filter.ExpiringSoon)
	}

	query += " ORDER BY renewal_date ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []Contract
	for rows.Next() {
		var c Contract
		var renewalDate, createdAt string
		var lastPrice sql.NullFloat64
		if err := rows.Scan(&c.ID, &c.UserID, &c.Vendor, &c.SKU, &c.Seats, &c.CurrentPrice,
			&renewalDate, &c.Status, &lastPrice, &createdAt); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		if lastPrice.Valid {
			c.LastNegotiatedPrice = lastPrice.Float64
		}
		c.RenewalDate, _ = time.Parse(time.RFC3339, renewalDate)
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		contracts = append(contracts, c)
	}
	return contracts, rows.Err()
}

// UpdateContract updates mutable fields of a contract.
func (s *Store) UpdateContract(ctx context.Context, c *Contract) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE contracts SET vendor=?, sku=?, seats=?, current_price=?, renewal_date=?, status=?, last_negotiated_price=?
		WHERE id=?
	`, c.Vendor, c.SKU, c.Seats, c.CurrentPrice, c.RenewalDate.Format(time.RFC3339),
		c.Status, nullableFloat(c.LastNegotiatedPrice), c.ID)
	if err != nil {
		return fmt.Errorf("update contract: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("contract not found: %s", c.ID)
	}
	return nil
}

// GetContractsExpiringSoon returns active contracts renewing within the given number of days.
func (s *Store) GetContractsExpiringSoon(ctx context.Context, daysAhead int) ([]Contract, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, vendor, sku, seats, current_price, renewal_date, status, last_negotiated_price, created_at
		FROM contracts
		WHERE status = 'active'
		  AND renewal_date BETWEEN datetime('now') AND datetime('now', '+' || ? || ' days')
		ORDER BY renewal_date ASC
	`, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("get contracts expiring soon: %w", err)
	}
	defer rows.Close()

	var contracts []Contract
	for rows.Next() {
		var c Contract
		var renewalDate, createdAt string
		var lastPrice sql.NullFloat64
		if err := rows.Scan(&c.ID, &c.UserID, &c.Vendor, &c.SKU, &c.Seats, &c.CurrentPrice,
			&renewalDate, &c.Status, &lastPrice, &createdAt); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		if lastPrice.Valid {
			c.LastNegotiatedPrice = lastPrice.Float64
		}
		c.RenewalDate, _ = time.Parse(time.RFC3339, renewalDate)
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		contracts = append(contracts, c)
	}
	return contracts, rows.Err()
}

// nullableFloat converts a float64 to sql.NullFloat64 for optional DB fields.
func nullableFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}
