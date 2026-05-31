package budgetalerts

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Engine implements budget threshold alert logic.
type Engine struct {
	store  *Store
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a budget alert engine.
func NewEngine(store *Store, db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		db:     db,
		logger: logger,
	}
}

// Store returns the underlying store for MCP handlers.
func (e *Engine) Store() *Store {
	return e.store
}

type budgetRow struct {
	Vendor string
	Amount float64
}

// CheckBudgets evaluates all budgets against actual spend and returns alerts.
func (e *Engine) CheckBudgets(ctx context.Context) ([]BudgetAlert, error) {
	budgets, err := e.loadBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("check budgets: load: %w", err)
	}

	var alerts []BudgetAlert
	now := time.Now().UTC().Format(time.RFC3339)

	for _, b := range budgets {
		actual, err := e.vendorActualSpend(ctx, b.Vendor)
		if err != nil {
			e.logger.Warn("query actual spend", "vendor", b.Vendor, "error", err)
			continue
		}

		if b.Amount <= 0 {
			continue
		}
		consumedPct := (actual / b.Amount) * 100

		alert := BudgetAlert{
			Vendor:      b.Vendor,
			Budget:      b.Amount,
			Actual:      actual,
			ConsumedPct: consumedPct,
		}

		switch {
		case consumedPct > 100:
			alert.Level = LevelCritical
			alert.Action = "immediate review needed - budget exceeded"
		case consumedPct > 90:
			alert.Level = LevelWarning
			alert.Action = "budget nearly exhausted - reduce spending"
		case consumedPct > 80:
			alert.Level = LevelInfo
			alert.Action = "budget running high - monitor closely"
		default:
			continue
		}

		alerts = append(alerts, alert)

		hist := &BudgetAlertHistory{
			Vendor:      b.Vendor,
			Budget:      b.Amount,
			Actual:      actual,
			ConsumedPct: consumedPct,
			Level:       alert.Level,
			CreatedAt:   now,
		}
		if err := e.store.Save(ctx, hist); err != nil {
			e.logger.Warn("save alert history", "vendor", b.Vendor, "error", err)
		}
	}

	return alerts, nil
}

func (e *Engine) loadBudgets(ctx context.Context) ([]budgetRow, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT vendor, budget_amount FROM spend_budgets`)
	if err != nil {
		return nil, fmt.Errorf("query budgets: %w", err)
	}
	defer rows.Close()

	var budgets []budgetRow
	for rows.Next() {
		var b budgetRow
		if err := rows.Scan(&b.Vendor, &b.Amount); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

func (e *Engine) vendorActualSpend(ctx context.Context, vendor string) (float64, error) {
	var total sql.NullFloat64
	err := e.db.QueryRowContext(ctx, `
		SELECT SUM(final_price) FROM deal_outcomes WHERE vendor = ?`, vendor).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("query actual spend: %w", err)
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}
