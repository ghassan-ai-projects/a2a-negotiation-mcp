package vendorspend

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Engine computes vendor spend analytics from deal_outcomes.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a vendorspend Engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Report generates a vendor spend report. If vendor is non-empty, filters to that vendor.
// period is a duration like "1y", "90d", "30d". Default "1y".
func (e *Engine) Report(ctx context.Context, vendor string, period string) (VendorSpendReport, error) {
	if period == "" {
		period = "1y"
	}

	since := parsePeriod(period)

	// Total spend for period
	totalSpend, err := e.totalSpendSince(ctx, since, vendor)
	if err != nil {
		return VendorSpendReport{}, fmt.Errorf("total spend: %w", err)
	}

	// By vendor breakdown
	entries, err := e.vendorBreakdown(ctx, since, vendor, totalSpend)
	if err != nil {
		return VendorSpendReport{}, fmt.Errorf("vendor breakdown: %w", err)
	}

	// Monthly trend
	trend, err := e.monthlyTrend(ctx, since, vendor)
	if err != nil {
		e.logger.Warn("vendor spend: monthly trend failed", "error", err)
		trend = []SpendTrendPoint{}
	}

	// Unique vendor count
	vendorCount := len(entries)
	totalSubs := 0
	for _, e := range entries {
		totalSubs += e.Subscriptions
	}

	// YoY change
	yoyChange := e.yoyChange(ctx, vendor)

	report := VendorSpendReport{
		Period:        period,
		TotalSpend:    math.Round(totalSpend*100) / 100,
		Vendors:       vendorCount,
		Subscriptions: totalSubs,
		ByVendor:      entries,
		MonthlyTrend:  trend,
		YoYChangePct:  yoyChange,
	}
	if report.ByVendor == nil {
		report.ByVendor = []VendorSpendEntry{}
	}
	if report.MonthlyTrend == nil {
		report.MonthlyTrend = []SpendTrendPoint{}
	}
	return report, nil
}

func (e *Engine) totalSpendSince(ctx context.Context, since time.Time, vendor string) (float64, error) {
	var total sql.NullFloat64
	var err error
	if vendor != "" {
		err = e.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(final_price), 0) FROM deal_outcomes WHERE created_at >= ? AND vendor = ?`,
			since.Format(time.RFC3339), vendor,
		).Scan(&total)
	} else {
		err = e.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(final_price), 0) FROM deal_outcomes WHERE created_at >= ?`,
			since.Format(time.RFC3339),
		).Scan(&total)
	}
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}

func (e *Engine) vendorBreakdown(ctx context.Context, since time.Time, filterVendor string, totalSpend float64) ([]VendorSpendEntry, error) {
	var rows *sql.Rows
	var err error

	if filterVendor != "" {
		rows, err = e.db.QueryContext(ctx, `
			SELECT vendor,
			       COALESCE(SUM(final_price), 0) AS total_spend,
			       COUNT(*) AS subscriptions,
			       COALESCE(AVG(final_price), 0) AS avg_cost
			FROM deal_outcomes
			WHERE created_at >= ? AND vendor = ?
			GROUP BY vendor
			ORDER BY total_spend DESC
		`, since.Format(time.RFC3339), filterVendor)
	} else {
		rows, err = e.db.QueryContext(ctx, `
			SELECT vendor,
			       COALESCE(SUM(final_price), 0) AS total_spend,
			       COUNT(*) AS subscriptions,
			       COALESCE(AVG(final_price), 0) AS avg_cost
			FROM deal_outcomes
			WHERE created_at >= ?
			GROUP BY vendor
			ORDER BY total_spend DESC
		`, since.Format(time.RFC3339))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []VendorSpendEntry
	for rows.Next() {
		var entry VendorSpendEntry
		if err := rows.Scan(&entry.Vendor, &entry.TotalSpend, &entry.Subscriptions, &entry.AvgCost); err != nil {
			return nil, err
		}
		if totalSpend > 0 {
			entry.SpendPct = math.Round((entry.TotalSpend/totalSpend)*10000) / 100
		}
		entry.TotalSpend = math.Round(entry.TotalSpend*100) / 100
		entry.AvgCost = math.Round(entry.AvgCost*100) / 100
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (e *Engine) monthlyTrend(ctx context.Context, since time.Time, vendor string) ([]SpendTrendPoint, error) {
	var rows *sql.Rows
	var err error

	if vendor != "" {
		rows, err = e.db.QueryContext(ctx, `
			SELECT strftime('%Y-%m', created_at) AS month,
			       COALESCE(SUM(final_price), 0) AS total_spend
			FROM deal_outcomes
			WHERE created_at >= ? AND vendor = ?
			GROUP BY strftime('%Y-%m', created_at)
			ORDER BY month ASC
		`, since.Format(time.RFC3339), vendor)
	} else {
		rows, err = e.db.QueryContext(ctx, `
			SELECT strftime('%Y-%m', created_at) AS month,
			       COALESCE(SUM(final_price), 0) AS total_spend
			FROM deal_outcomes
			WHERE created_at >= ?
			GROUP BY strftime('%Y-%m', created_at)
			ORDER BY month ASC
		`, since.Format(time.RFC3339))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []SpendTrendPoint
	for rows.Next() {
		var p SpendTrendPoint
		if err := rows.Scan(&p.Month, &p.Spend); err != nil {
			return nil, err
		}
		p.Spend = math.Round(p.Spend*100) / 100
		trend = append(trend, p)
	}
	return trend, rows.Err()
}

func (e *Engine) yoyChange(ctx context.Context, vendor string) float64 {
	now := time.Now().UTC()
	oneYearAgo := now.AddDate(-1, 0, 0)
	twoYearsAgo := now.AddDate(-2, 0, 0)

	currentPeriod, err := e.totalSpendSince(ctx, oneYearAgo, vendor)
	if err != nil || currentPeriod == 0 {
		return 0
	}

	prevPeriod, err := e.totalSpendSince(ctx, twoYearsAgo, vendor)
	if err != nil || prevPeriod == 0 {
		return 0
	}
	// Subtract current period from prev to get the period strictly one year before
	prevOnly := prevPeriod - currentPeriod
	if prevOnly <= 0 {
		return 0
	}

	change := ((currentPeriod - prevOnly) / prevOnly) * 100
	return math.Round(change*100) / 100
}

func parsePeriod(period string) time.Time {
	now := time.Now().UTC()
	switch {
	case len(period) > 1 && period[len(period)-1] == 'y':
		n := 0
		fmt.Sscanf(period, "%dy", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(-n, 0, 0)
	case len(period) > 2 && period[len(period)-1] == 'd':
		n := 0
		fmt.Sscanf(period, "%dd", &n)
		if n <= 0 {
			n = 90
		}
		return now.AddDate(0, 0, -n)
	case len(period) > 2 && period[len(period)-1] == 'm':
		n := 0
		fmt.Sscanf(period, "%dm", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(0, -n, 0)
	default:
		return now.AddDate(-1, 0, 0)
	}
}
