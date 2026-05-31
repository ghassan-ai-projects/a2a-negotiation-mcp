package budget

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
)

// Engine manages budget vs actual calculations.
type Engine struct {
	store       *Store
	historyDB   *sql.DB
	logger      *slog.Logger
}

// NewEngine creates a budget Engine.
func NewEngine(store *Store, historyDB *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{
		store:     store,
		historyDB: historyDB,
		logger:    logger,
	}
}

// Dashboard computes the budget vs actual dashboard for the given period.
// period options: "monthly", "quarterly", "yearly". Default "monthly".
func (e *Engine) Dashboard(ctx context.Context, period string) (BudgetDashboard, error) {
	if period == "" {
		period = "monthly"
	}

	budgets, err := e.store.ListBudgets(ctx)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("list budgets: %w", err)
	}

	var totalBudget, totalActual float64
	var byVendor []VendorBudget
	var warnings []Warning
	vendorActuals := make(map[string]float64)

	// For each budget, compute actual spend from deal_outcomes
	for _, b := range budgets {
		actual, err := e.vendorActualSpend(ctx, b.Vendor)
		if err != nil {
			e.logger.Warn("dashboard: vendor actual spend query failed", "vendor", b.Vendor, "error", err)
			continue
		}

		vendorActuals[b.Vendor] = actual
		totalBudget += b.BudgetAmount
		totalActual += actual

		vb := VendorBudget{
			Vendor:       b.Vendor,
			BudgetAmount: b.BudgetAmount,
			PeriodMonth:  b.PeriodMonth,
		}
		byVendor = append(byVendor, vb)

		if actual > b.BudgetAmount {
			vPct := ((actual - b.BudgetAmount) / b.BudgetAmount) * 100
			if vPct > 10 {
				warnings = append(warnings, Warning{
					Vendor:      b.Vendor,
					Budget:      b.BudgetAmount,
					Actual:      actual,
					VariancePct: math.Round(vPct*100) / 100,
					Message:     fmt.Sprintf("%s overspent by %.1f%% (budget: $%.2f, actual: $%.2f)", b.Vendor, vPct, b.BudgetAmount, actual),
				})
			}
		}
	}

	variance := totalActual - totalBudget
	var variancePct float64
	if totalBudget > 0 {
		variancePct = math.Round((variance/totalBudget)*10000) / 100
	}

	trend, err := e.monthlyTrend(ctx)
	if err != nil {
		e.logger.Warn("dashboard: monthly trend query failed", "error", err)
		trend = []BudgetTrend{}
	}

	dash := BudgetDashboard{
		Period:       period,
		TotalBudget:  math.Round(totalBudget*100) / 100,
		TotalActual:  math.Round(totalActual*100) / 100,
		Variance:     math.Round(variance*100) / 100,
		VariancePct:  variancePct,
		ByVendor:     byVendor,
		MonthlyTrend: trend,
		Warnings:     warnings,
	}
	if dash.Warnings == nil {
		dash.Warnings = []Warning{}
	}
	if dash.ByVendor == nil {
		dash.ByVendor = []VendorBudget{}
	}

	return dash, nil
}

func (e *Engine) vendorActualSpend(ctx context.Context, vendor string) (float64, error) {
	var actual sql.NullFloat64
	err := e.historyDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(final_price), 0)
		FROM deal_outcomes
		WHERE vendor = ?
	`, vendor).Scan(&actual)
	if err != nil {
		return 0, err
	}
	return actual.Float64, nil
}

func (e *Engine) monthlyTrend(ctx context.Context) ([]BudgetTrend, error) {
	rows, err := e.historyDB.QueryContext(ctx, `
		SELECT strftime('%Y-%m', created_at) AS month,
		       COALESCE(SUM(final_price), 0) AS total_spend
		FROM deal_outcomes
		GROUP BY strftime('%Y-%m', created_at)
		ORDER BY month ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// We need to merge budget data with trend
	budgets, err := e.store.ListBudgets(ctx)
	if err != nil {
		return nil, err
	}
	// Build budget sum per month (budgets aren't per-month, so spread budget across months)
	// Simplified: use period_month from budget as target
	budgetByMonth := make(map[string]float64)
	for _, b := range budgets {
		if b.PeriodMonth != "" {
			budgetByMonth[b.PeriodMonth] += b.BudgetAmount
		}
	}

	var trend []BudgetTrend
	for rows.Next() {
		var month string
		var actual float64
		if err := rows.Scan(&month, &actual); err != nil {
			return nil, err
		}
		budget := budgetByMonth[month]
		trend = append(trend, BudgetTrend{
			Month:  month,
			Budget: math.Round(budget*100) / 100,
			Actual: math.Round(actual*100) / 100,
		})
	}
	return trend, rows.Err()
}

// Store returns the underlying store (for MCP tool handlers that need it).
func (e *Engine) Store() *Store {
	return e.store
}
