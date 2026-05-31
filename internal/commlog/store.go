package commlog

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides SQLite persistence for vendor communications.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store sharing the given DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("commlog migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS vendor_communications (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor      TEXT NOT NULL,
			comm_type   TEXT NOT NULL,
			summary     TEXT NOT NULL,
			detail      TEXT DEFAULT '',
			created_at  TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_vendor_communications_vendor ON vendor_communications(vendor);
	`)
	return err
}

// Log inserts a new communication entry.
func (s *Store) Log(ctx context.Context, vendor, commType, summary, detail string) (*CommEntry, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_communications (vendor, comm_type, summary, detail, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
	`, vendor, commType, summary, detail)
	if err != nil {
		return nil, fmt.Errorf("log communication: %w", err)
	}
	id, _ := result.LastInsertId()

	var createdAt string
	err = s.db.QueryRowContext(ctx, `
		SELECT created_at FROM vendor_communications WHERE id = ?
	`, id).Scan(&createdAt)
	if err != nil {
		return nil, fmt.Errorf("readback communication: %w", err)
	}

	return &CommEntry{
		ID:        id,
		Vendor:    vendor,
		CommType:  commType,
		Summary:   summary,
		Detail:    detail,
		CreatedAt: createdAt,
	}, nil
}

// ListByVendor returns communication entries for a vendor, most recent first.
func (s *Store) ListByVendor(ctx context.Context, vendor string, limit int) ([]CommEntry, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// Total count
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM vendor_communications WHERE vendor = ?
	`, vendor).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count communications: %w", err)
	}

	// Paginated results
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, comm_type, summary, detail, created_at
		FROM vendor_communications
		WHERE vendor = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, vendor, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list communications: %w", err)
	}
	defer rows.Close()

	var entries []CommEntry
	for rows.Next() {
		var e CommEntry
		if err := rows.Scan(&e.ID, &e.Vendor, &e.CommType, &e.Summary, &e.Detail, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan communication: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// Delete removes a communication entry by ID.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM vendor_communications WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete communication: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("communication not found: %d", id)
	}
	return nil
}
