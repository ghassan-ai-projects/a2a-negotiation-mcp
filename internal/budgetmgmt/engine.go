package budgetmgmt

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Engine manages budget allocation, rollover, and forecasting.
type Engine struct {
	store     *Store
	historyDB *sql.DB
	logger    *slog.Logger
}

// NewEngine creates a budgetmgmt Engine.
func NewEngine(store *Store, historyDB *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{
		store:     store,
		historyDB: historyDB,
		logger:    logger,
	}
}

// SetBudget sets the monthly budget allocation for a vendor.
func (e *Engine) SetBudget(ctx context.Context, vendor, month string, amount float64) error {
	return e.store.SetMonthlyBudget(ctx, vendor, month, amount)
}

// GetDashboard returns a per-month budget dashboard with actuals from deal_outcomes.
func (e *Engine) GetDashboard(ctx context.Context, month string) (BudgetDashboard, error) {
	rows, err := e.store.db.QueryContext(ctx, `
		SELECT vendor, month, budget_amount, spent, rolled_over, created_at
		FROM monthly_budgets WHERE month = ?
	`, month)
	if err != nil {
		return BudgetDashboard{}, fmt.Errorf("query monthly budgets: %w", err)
	}
	defer rows.Close()

	var items []MonthlyBudget
	var totalBudget, totalSpent float64

	for rows.Next() {
		var mb MonthlyBudget
		if err := rows.Scan(&mb.Vendor, &mb.Month, &mb.BudgetAmount, &mb.Spent, &mb.RolledOver, &mb.CreatedAt); err != nil {
			return BudgetDashboard{}, err
		}

		// Query actual spend from deal_outcomes for this vendor in this month
		actual, err := e.vendorMonthSpend(ctx, mb.Vendor, month)
		if err != nil {
			e.logger.Warn("dashboard: vendor month spend query failed", "vendor", mb.Vendor, "error", err)
		} else {
			mb.Spent = actual
		}

		totalBudget += mb.BudgetAmount
		totalSpent += mb.Spent
		items = append(items, mb)
	}
	if err := rows.Err(); err != nil {
		return BudgetDashboard{}, err
	}

	if items == nil {
		items = []MonthlyBudget{}
	}

	return BudgetDashboard{
		Month:       month,
		TotalBudget: math.Round(totalBudget*100) / 100,
		TotalSpent:  math.Round(totalSpent*100) / 100,
		Items:       items,
	}, nil
}

// GetForecast computes YTD budget vs spend and projects annual based on trend.
func (e *Engine) GetForecast(ctx context.Context, vendor string) (*BudgetForecast, error) {
	now := time.Now()
	currentMonth := now.Format("2006-01")

	ytdBudgets, err := e.store.GetYTD(ctx, vendor, currentMonth)
	if err != nil {
		return nil, fmt.Errorf("get YTD: %w", err)
	}

	var ytdBudget, ytdSpent float64
	for _, b := range ytdBudgets {
		ytdBudget += b.BudgetAmount

		// Get actual spend from deal_outcomes for each month
		actual, err := e.vendorMonthSpend(ctx, vendor, b.Month)
		if err != nil {
			e.logger.Warn("forecast: vendor month spend failed", "vendor", vendor, "month", b.Month, "error", err)
			actual = b.Spent
		}
		ytdSpent += actual
	}

	ytdBudget = math.Round(ytdBudget*100) / 100
	ytdSpent = math.Round(ytdSpent*100) / 100

	monthInt := int(now.Month())
	remainingMonths := 12 - monthInt

	var projectedAnnual float64
	var status string
	if monthInt > 0 {
		monthlyAvgSpend := ytdSpent / float64(monthInt)
		projectedAnnual = ytdSpent + monthlyAvgSpend*float64(remainingMonths)
		projectedAnnual = math.Round(projectedAnnual*100) / 100

		annualizedBudget := (ytdBudget / float64(monthInt)) * 12
		if projectedAnnual <= annualizedBudget {
			status = "on_track"
		} else if projectedAnnual <= annualizedBudget*1.1 {
			status = "at_risk"
		} else {
			status = "over_budget"
		}
	} else {
		projectedAnnual = 0
		status = "no_data"
	}

	return &BudgetForecast{
		Vendor:          vendor,
		YTDBudget:       ytdBudget,
		YTDSpent:        ytdSpent,
		ProjectedAnnual: projectedAnnual,
		RemainingMonths: remainingMonths,
		Status:          status,
	}, nil
}

// Rollover moves unused budget from one month to another.
func (e *Engine) Rollover(ctx context.Context, vendor, fromMonth, toMonth string) (*RolloverResult, error) {
	// Get the source budget
	src, err := e.store.GetMonthlyBudget(ctx, vendor, fromMonth)
	if err != nil {
		return nil, fmt.Errorf("get source budget: %w", err)
	}
	if src == nil {
		return nil, fmt.Errorf("no budget found for vendor %q month %q", vendor, fromMonth)
	}

	// Calculate unused amount
	actual, err := e.vendorMonthSpend(ctx, vendor, fromMonth)
	if err != nil {
		e.logger.Warn("rollover: vendor month spend failed", "vendor", vendor, "month", fromMonth, "error", err)
		actual = src.Spent
	}

	unused := src.BudgetAmount - actual
	if unused <= 0 {
		return nil, fmt.Errorf("no unused budget to roll over from %s/%s (budget: %.2f, spent: %.2f)", vendor, fromMonth, src.BudgetAmount, actual)
	}

	// Ensure destination month has a budget row
	dst, err := e.store.GetMonthlyBudget(ctx, vendor, toMonth)
	if err != nil {
		return nil, fmt.Errorf("get destination budget: %w", err)
	}
	if dst == nil {
		if err := e.store.SetMonthlyBudget(ctx, vendor, toMonth, 0); err != nil {
			return nil, fmt.Errorf("create destination budget: %w", err)
		}
	}

	// Update rollover on source and add to destination
	if err := e.store.UpdateRollover(ctx, vendor, fromMonth, unused); err != nil {
		return nil, fmt.Errorf("update source rollover: %w", err)
	}

	// Add unused amount to destination budget
	_, err = e.store.db.ExecContext(ctx, `
		UPDATE monthly_budgets SET budget_amount = budget_amount + ?
		WHERE vendor = ? AND month = ?
	`, unused, vendor, toMonth)
	if err != nil {
		return nil, fmt.Errorf("update destination budget: %w", err)
	}

	return &RolloverResult{
		Vendor:    vendor,
		FromMonth: fromMonth,
		ToMonth:   toMonth,
		Amount:    math.Round(unused*100) / 100,
		Status:    "rolled_over",
	}, nil
}

func (e *Engine) vendorMonthSpend(ctx context.Context, vendor, month string) (float64, error) {
	var actual sql.NullFloat64
	err := e.historyDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(final_price), 0)
		FROM deal_outcomes
		WHERE vendor = ? AND strftime('%Y-%m', created_at) = ?
	`, vendor, month).Scan(&actual)
	if err != nil {
		return 0, err
	}
	return actual.Float64, nil
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
