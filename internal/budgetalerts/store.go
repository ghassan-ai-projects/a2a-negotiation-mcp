package budgetalerts

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides SQLite persistence for budget alert history.
type Store struct {
	db *sql.DB
}

// NewStore creates a new budget alert store sharing the given DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("budgetalerts migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS budget_alert_history (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor       TEXT NOT NULL,
			budget       REAL NOT NULL,
			actual       REAL NOT NULL,
			consumed_pct REAL NOT NULL,
			level        TEXT NOT NULL,
			created_at   TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	return err
}

// Save inserts a budget alert history record.
func (s *Store) Save(ctx context.Context, a *BudgetAlertHistory) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO budget_alert_history (vendor, budget, actual, consumed_pct, level, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, a.Vendor, a.Budget, a.Actual, a.ConsumedPct, string(a.Level), a.CreatedAt)
	return err
}

// List returns alert history for a vendor, ordered by most recent first.
func (s *Store) List(ctx context.Context, vendor string, limit int) ([]BudgetAlertHistory, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, budget, actual, consumed_pct, level, created_at
		FROM budget_alert_history
		WHERE vendor = ?
		ORDER BY created_at DESC
		LIMIT ?`, vendor, limit)
	if err != nil {
		return nil, fmt.Errorf("list alert history: %w", err)
	}
	defer rows.Close()

	var alerts []BudgetAlertHistory
	for rows.Next() {
		var a BudgetAlertHistory
		var levelStr string
		if err := rows.Scan(&a.ID, &a.Vendor, &a.Budget, &a.Actual, &a.ConsumedPct, &levelStr, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Level = Level(levelStr)
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}
