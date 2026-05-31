package spendingcaps

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages spending caps in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a spendingcaps Store backed by the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate spendingcaps: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS spending_caps (
		vendor TEXT NOT NULL PRIMARY KEY,
		soft_cap REAL NOT NULL DEFAULT 0,
		hard_cap REAL NOT NULL DEFAULT 0,
		period TEXT NOT NULL DEFAULT 'monthly',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetCap creates or updates a spending cap for a vendor.
func (s *Store) SetCap(ctx context.Context, vendor string, softCap, hardCap float64, period string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO spending_caps (vendor, soft_cap, hard_cap, period, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor) DO UPDATE SET
			soft_cap = excluded.soft_cap,
			hard_cap = excluded.hard_cap,
			period = excluded.period,
			updated_at = excluded.updated_at
	`, vendor, softCap, hardCap, period, now, now)
	return err
}

// GetCap returns the spending cap for a vendor.
func (s *Store) GetCap(ctx context.Context, vendor string) (*SpendingCap, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT vendor, soft_cap, hard_cap, period, created_at, updated_at
		FROM spending_caps WHERE vendor = ?
	`, vendor)

	var sc SpendingCap
	err := row.Scan(&sc.Vendor, &sc.SoftCap, &sc.HardCap, &sc.Period, &sc.CreatedAt, &sc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sc, nil
}

// ListCaps returns all spending caps.
func (s *Store) ListCaps(ctx context.Context) ([]SpendingCap, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, soft_cap, hard_cap, period, created_at, updated_at
		FROM spending_caps ORDER BY vendor
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SpendingCap
	for rows.Next() {
		var sc SpendingCap
		if err := rows.Scan(&sc.Vendor, &sc.SoftCap, &sc.HardCap, &sc.Period, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, sc)
	}
	return result, rows.Err()
}

// DeleteCap removes a spending cap for a vendor.
func (s *Store) DeleteCap(ctx context.Context, vendor string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM spending_caps WHERE vendor = ?`, vendor)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("spending cap not found for vendor %q", vendor)
	}
	return nil
}
