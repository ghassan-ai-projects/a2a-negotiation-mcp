package savingsrealization

import (
	"context"
	"database/sql"
	"fmt"
)

// Store manages savings realization records in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a savingsrealization Store backed by the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate savingsrealization: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS savings_realization (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		vendor TEXT NOT NULL,
		projected_amount REAL NOT NULL DEFAULT 0,
		actual_amount REAL NOT NULL DEFAULT 0,
		period TEXT NOT NULL DEFAULT 'monthly',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Record inserts a new savings realization record.
func (s *Store) Record(ctx context.Context, sessionID, vendor string, projectedAmount, actualAmount float64, period string) (*SavingsRealization, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO savings_realization (session_id, vendor, projected_amount, actual_amount, period, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`, sessionID, vendor, projectedAmount, actualAmount, period)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &SavingsRealization{
		ID:              id,
		SessionID:       sessionID,
		Vendor:          vendor,
		ProjectedAmount: projectedAmount,
		ActualAmount:    actualAmount,
		Period:          period,
	}, nil
}

// ListByVendor returns all realization records for a vendor.
func (s *Store) ListByVendor(ctx context.Context, vendor string) ([]SavingsRealization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, vendor, projected_amount, actual_amount, period, created_at
		FROM savings_realization
		WHERE vendor = ?
		ORDER BY created_at DESC
	`, vendor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SavingsRealization
	for rows.Next() {
		var sr SavingsRealization
		if err := rows.Scan(&sr.ID, &sr.SessionID, &sr.Vendor, &sr.ProjectedAmount, &sr.ActualAmount, &sr.Period, &sr.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, sr)
	}
	return result, rows.Err()
}

// GetReport returns all realization records for the given period.
func (s *Store) GetReport(ctx context.Context, period string) ([]SavingsRealization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, vendor, projected_amount, actual_amount, period, created_at
		FROM savings_realization
		WHERE period = ?
		ORDER BY vendor, created_at DESC
	`, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SavingsRealization
	for rows.Next() {
		var sr SavingsRealization
		if err := rows.Scan(&sr.ID, &sr.SessionID, &sr.Vendor, &sr.ProjectedAmount, &sr.ActualAmount, &sr.Period, &sr.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, sr)
	}
	return result, rows.Err()
}
