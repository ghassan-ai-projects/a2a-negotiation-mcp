package batchnegotiation

import (
	"context"
	"database/sql"
	"fmt"
)

// BatchRecord is a stored batch negotiation record.
type BatchRecord struct {
	ID           string  `json:"id"`
	VendorCount  int     `json:"vendor_count"`
	TotalSavings float64 `json:"total_savings"`
	DurationMs   int64   `json:"duration_ms"`
	CreatedAt    string  `json:"created_at"`
}

// Store persists batch negotiation records.
type Store struct {
	db *sql.DB
}

// NewStore creates a batchnegotiation Store.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate batchnegotiation: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS batch_negotiations (
			id TEXT PRIMARY KEY,
			vendor_count INTEGER NOT NULL DEFAULT 0,
			total_savings REAL NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)
	`)
	return err
}

// Save persists a batch record.
func (s *Store) Save(ctx context.Context, record *BatchRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO batch_negotiations (id, vendor_count, total_savings, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, record.ID, record.VendorCount, record.TotalSavings, record.DurationMs, record.CreatedAt)
	if err != nil {
		return fmt.Errorf("save batch: %w", err)
	}
	return nil
}

// List returns all batch records ordered by creation time descending.
func (s *Store) List(ctx context.Context) ([]BatchRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor_count, total_savings, duration_ms, created_at
		FROM batch_negotiations
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()

	var records []BatchRecord
	for rows.Next() {
		var r BatchRecord
		if err := rows.Scan(&r.ID, &r.VendorCount, &r.TotalSavings, &r.DurationMs, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan batch: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
