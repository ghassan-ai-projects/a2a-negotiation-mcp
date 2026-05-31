package budget

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages budget targets in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a budget Store backed by the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate budget: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS spend_budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL UNIQUE,
		budget_amount REAL NOT NULL DEFAULT 0,
		period_month TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetBudget creates or updates a budget for a vendor.
func (s *Store) SetBudget(ctx context.Context, vendor string, budgetAmount float64, periodMonth string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO spend_budgets (vendor, budget_amount, period_month, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(vendor) DO UPDATE SET
			budget_amount = excluded.budget_amount,
			period_month = excluded.period_month,
			updated_at = excluded.updated_at
	`, vendor, budgetAmount, periodMonth, now, now)
	return err
}

// GetBudget returns the budget for a specific vendor.
func (s *Store) GetBudget(ctx context.Context, vendor string) (*VendorBudget, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT vendor, budget_amount, period_month
		FROM spend_budgets WHERE vendor = ?
	`, vendor)

	var vb VendorBudget
	err := row.Scan(&vb.Vendor, &vb.BudgetAmount, &vb.PeriodMonth)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &vb, nil
}

// ListBudgets returns all budgets.
func (s *Store) ListBudgets(ctx context.Context) ([]VendorBudget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, budget_amount, period_month
		FROM spend_budgets ORDER BY vendor
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []VendorBudget
	for rows.Next() {
		var vb VendorBudget
		if err := rows.Scan(&vb.Vendor, &vb.BudgetAmount, &vb.PeriodMonth); err != nil {
			return nil, err
		}
		result = append(result, vb)
	}
	return result, rows.Err()
}

// DeleteBudget removes a budget for a vendor.
func (s *Store) DeleteBudget(ctx context.Context, vendor string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM spend_budgets WHERE vendor = ?`, vendor)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("budget not found for vendor %q", vendor)
	}
	return nil
}
