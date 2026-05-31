package budgetmgmt

import (
	"context"
	"database/sql"
	"fmt"
)

// Store manages monthly budget allocations in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a budgetmgmt Store backed by the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate budgetmgmt: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS monthly_budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		month TEXT NOT NULL,
		budget_amount REAL NOT NULL DEFAULT 0,
		spent REAL NOT NULL DEFAULT 0,
		rolled_over REAL NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		UNIQUE(vendor, month)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetMonthlyBudget creates or updates a monthly budget allocation.
func (s *Store) SetMonthlyBudget(ctx context.Context, vendor, month string, budgetAmount float64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO monthly_budgets (vendor, month, budget_amount, created_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(vendor, month) DO UPDATE SET
			budget_amount = excluded.budget_amount
	`, vendor, month, budgetAmount)
	return err
}

// GetMonthlyBudget returns the budget for a specific vendor/month.
func (s *Store) GetMonthlyBudget(ctx context.Context, vendor, month string) (*MonthlyBudget, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT vendor, month, budget_amount, spent, rolled_over, created_at
		FROM monthly_budgets WHERE vendor = ? AND month = ?
	`, vendor, month)

	var mb MonthlyBudget
	err := row.Scan(&mb.Vendor, &mb.Month, &mb.BudgetAmount, &mb.Spent, &mb.RolledOver, &mb.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mb, nil
}

// GetYTD returns all budgets and actuals YTD for a vendor up to and including the given month.
func (s *Store) GetYTD(ctx context.Context, vendor, month string) ([]MonthlyBudget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, month, budget_amount, spent, rolled_over, created_at
		FROM monthly_budgets
		WHERE vendor = ? AND month <= ?
		ORDER BY month ASC
	`, vendor, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthlyBudget
	for rows.Next() {
		var mb MonthlyBudget
		if err := rows.Scan(&mb.Vendor, &mb.Month, &mb.BudgetAmount, &mb.Spent, &mb.RolledOver, &mb.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, mb)
	}
	return result, rows.Err()
}

// ListByVendor returns all monthly budgets for a vendor.
func (s *Store) ListByVendor(ctx context.Context, vendor string) ([]MonthlyBudget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, month, budget_amount, spent, rolled_over, created_at
		FROM monthly_budgets
		WHERE vendor = ?
		ORDER BY month ASC
	`, vendor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MonthlyBudget
	for rows.Next() {
		var mb MonthlyBudget
		if err := rows.Scan(&mb.Vendor, &mb.Month, &mb.BudgetAmount, &mb.Spent, &mb.RolledOver, &mb.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, mb)
	}
	return result, rows.Err()
}

// UpdateRollover updates the rolled_over field for a specific budget row.
func (s *Store) UpdateRollover(ctx context.Context, vendor, month string, rolledOver float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE monthly_budgets SET rolled_over = rolled_over + ?
		WHERE vendor = ? AND month = ?
	`, rolledOver, vendor, month)
	return err
}

// UpdateSpent updates the spent amount for a specific budget row.
func (s *Store) UpdateSpent(ctx context.Context, vendor, month string, spent float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE monthly_budgets SET spent = ?
		WHERE vendor = ? AND month = ?
	`, spent, vendor, month)
	return err
}
