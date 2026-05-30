package roi

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages ROI calculation persistence backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate roi: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS roi_calculations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		current_spend REAL NOT NULL,
		negotiated_price REAL NOT NULL,
		implementation_costs REAL NOT NULL DEFAULT 0,
		annual_overhead REAL NOT NULL DEFAULT 0,
		annual_savings REAL NOT NULL,
		roi_pct REAL NOT NULL,
		payback_months REAL NOT NULL,
		savings_1y REAL NOT NULL,
		savings_3y REAL NOT NULL,
		savings_5y REAL NOT NULL,
		npv REAL NOT NULL,
		user_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Save persists an ROI calculation.
func (s *Store) Save(ctx context.Context, calc *ROICalculation) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO roi_calculations
			(vendor, current_spend, negotiated_price, implementation_costs, annual_overhead,
			 annual_savings, roi_pct, payback_months, savings_1y, savings_3y, savings_5y, npv, user_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, calc.Vendor, calc.CurrentSpend, calc.NegotiatedPrice, calc.ImplementationCosts, calc.AnnualOverhead,
		calc.AnnualSavings, calc.ROIPct, calc.PaybackMonths, calc.Savings1Y, calc.Savings3Y, calc.Savings5Y,
		calc.NPV, calc.UserID, calc.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save roi calculation: %w", err)
	}
	id, _ := result.LastInsertId()
	calc.ID = id
	return nil
}

// GetByID retrieves a single ROI calculation by ID.
func (s *Store) GetByID(ctx context.Context, id int64) (*ROICalculation, error) {
	var c ROICalculation
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, current_spend, negotiated_price, implementation_costs, annual_overhead,
		       annual_savings, roi_pct, payback_months, savings_1y, savings_3y, savings_5y, npv, user_id, created_at
		FROM roi_calculations WHERE id = ?
	`, id).Scan(&c.ID, &c.Vendor, &c.CurrentSpend, &c.NegotiatedPrice, &c.ImplementationCosts, &c.AnnualOverhead,
		&c.AnnualSavings, &c.ROIPct, &c.PaybackMonths, &c.Savings1Y, &c.Savings3Y, &c.Savings5Y,
		&c.NPV, &c.UserID, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get roi by id: %w", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &c, nil
}

// ListByUser returns all ROI calculations for a user, ordered by newest first.
func (s *Store) ListByUser(ctx context.Context, userID string) ([]ROICalculation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, current_spend, negotiated_price, implementation_costs, annual_overhead,
		       annual_savings, roi_pct, payback_months, savings_1y, savings_3y, savings_5y, npv, user_id, created_at
		FROM roi_calculations WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list roi by user: %w", err)
	}
	defer rows.Close()

	var results []ROICalculation
	for rows.Next() {
		var c ROICalculation
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Vendor, &c.CurrentSpend, &c.NegotiatedPrice, &c.ImplementationCosts, &c.AnnualOverhead,
			&c.AnnualSavings, &c.ROIPct, &c.PaybackMonths, &c.Savings1Y, &c.Savings3Y, &c.Savings5Y,
			&c.NPV, &c.UserID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan roi: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		results = append(results, c)
	}
	return results, rows.Err()
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
