package reports

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// Engine composes data from multiple analytics sources to build custom reports.
type Engine struct {
	historyStore *history.Store
}

// NewEngine creates a reports engine.
func NewEngine(historyStore *history.Store) *Engine {
	return &Engine{historyStore: historyStore}
}

// Build generates a custom report based on the requested sections.
func (e *Engine) Build(ctx context.Context, req ReportRequest) (*ReportResult, error) {
	start := time.Now()
	period := req.Period
	if period == "" {
		period = "all"
	}

	db := e.historyStore.DB()
	sections := make(map[string]any)

	for _, section := range req.Sections {
		switch section {
		case "savings":
			data, err := e.savingsSection(ctx, db, period, req.Vendor)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		case "vendor_breakdown":
			data, err := e.vendorBreakdownSection(ctx, db, period)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		case "win_loss":
			data, err := e.winLossSection(ctx, db, period, req.Vendor)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		case "benchmarks":
			data, err := e.benchmarksSection(ctx, db, period, req.Vendor)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		case "budget":
			data, err := e.budgetSection(ctx, db, req.Vendor)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		case "trends":
			data, err := e.trendsSection(ctx, db, period)
			if err != nil {
				return nil, fmt.Errorf("section %q: %w", section, err)
			}
			sections[section] = data

		default:
			return nil, fmt.Errorf("unknown section %q", section)
		}
	}

	return &ReportResult{
		Sections:     sections,
		GeneratedAt:  start.UTC().Format(time.RFC3339),
		SectionCount: len(sections),
	}, nil
}

// ─── Section Builders ───

func (e *Engine) savingsSection(ctx context.Context, db *sql.DB, period, vendor string) (map[string]any, error) {
	where, args := periodAndVendorClause(period, vendor)

	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(list_price), 0), COALESCE(SUM(final_price), 0), COUNT(*)
		FROM deal_outcomes WHERE 1=1 %s`, where)

	var totalList, totalFinal float64
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&totalList, &totalFinal, &count); err != nil {
		return nil, err
	}

	savings := totalList - totalFinal
	savingsPct := 0.0
	if totalList > 0 {
		savingsPct = (savings / totalList) * 100
	}

	return map[string]any{
		"total_list_price":   totalList,
		"total_final_price":  totalFinal,
		"total_savings":      savings,
		"savings_percentage": savingsPct,
		"deal_count":         count,
	}, nil
}

func (e *Engine) vendorBreakdownSection(ctx context.Context, db *sql.DB, period string) (map[string]any, error) {
	where, args := periodClause(period)

	query := fmt.Sprintf(`
		SELECT vendor, COUNT(*) AS deal_count, COALESCE(SUM(final_price), 0) AS total_spend
		FROM deal_outcomes WHERE 1=1 %s
		GROUP BY vendor ORDER BY total_spend DESC`, where)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type vendorEntry struct {
		Vendor     string  `json:"vendor"`
		DealCount  int     `json:"deal_count"`
		TotalSpend float64 `json:"total_spend"`
	}

	var entries []vendorEntry
	for rows.Next() {
		var ve vendorEntry
		if err := rows.Scan(&ve.Vendor, &ve.DealCount, &ve.TotalSpend); err != nil {
			return nil, err
		}
		entries = append(entries, ve)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"vendors": entries,
		"total":   len(entries),
	}, nil
}

func (e *Engine) winLossSection(ctx context.Context, db *sql.DB, period, vendor string) (map[string]any, error) {
	where, args := periodAndVendorClause(period, vendor)

	query := fmt.Sprintf(`
		SELECT outcome, COUNT(*) AS count
		FROM negotiation_sessions WHERE 1=1 %s AND outcome IN ('won', 'lost')
		GROUP BY outcome`, where)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	won, lost := 0, 0
	for rows.Next() {
		var outcome string
		var count int
		if err := rows.Scan(&outcome, &count); err != nil {
			return nil, err
		}
		switch outcome {
		case "won":
			won = count
		case "lost":
			lost = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	total := won + lost
	winRate := 0.0
	if total > 0 {
		winRate = float64(won) / float64(total) * 100
	}

	return map[string]any{
		"won":      won,
		"lost":     lost,
		"total":    total,
		"win_rate": winRate,
	}, nil
}

func (e *Engine) benchmarksSection(ctx context.Context, db *sql.DB, period, vendor string) (map[string]any, error) {
	where, args := periodAndVendorClause(period, vendor)

	query := fmt.Sprintf(`
		SELECT AVG(list_price), AVG(final_price), AVG(discount_pct), COUNT(*)
		FROM deal_outcomes WHERE 1=1 %s`, where)

	var avgList, avgFinal, avgDiscount sql.NullFloat64
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&avgList, &avgFinal, &avgDiscount, &count); err != nil {
		return nil, err
	}

	return map[string]any{
		"avg_list_price":   avgList.Float64,
		"avg_final_price":  avgFinal.Float64,
		"avg_discount_pct": avgDiscount.Float64,
		"deal_count":       count,
	}, nil
}

func (e *Engine) budgetSection(ctx context.Context, db *sql.DB, vendor string) (map[string]any, error) {
	where := ""
	args := []any{}
	if vendor != "" {
		where = "AND vendor = ?"
		args = append(args, vendor)
	}

	query := fmt.Sprintf(`
		SELECT vendor, budget_amount
		FROM spend_budgets WHERE 1=1 %s`, where)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type budgetEntry struct {
		Vendor       string  `json:"vendor"`
		BudgetAmount float64 `json:"budget_amount"`
	}

	var entries []budgetEntry
	totalBudget := 0.0
	for rows.Next() {
		var be budgetEntry
		if err := rows.Scan(&be.Vendor, &be.BudgetAmount); err != nil {
			return nil, err
		}
		entries = append(entries, be)
		totalBudget += be.BudgetAmount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Also query actual spend from deal_outcomes
	var totalSpend float64
	if vendor != "" {
		_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(final_price), 0) FROM deal_outcomes WHERE vendor = ?", vendor).Scan(&totalSpend)
	} else {
		_ = db.QueryRowContext(ctx, "SELECT COALESCE(SUM(final_price), 0) FROM deal_outcomes").Scan(&totalSpend)
	}

	return map[string]any{
		"budgets":      entries,
		"total_budget": totalBudget,
		"total_spend":  totalSpend,
		"remaining":    totalBudget - totalSpend,
		"vendor_count": len(entries),
	}, nil
}

func (e *Engine) trendsSection(ctx context.Context, db *sql.DB, period string) (map[string]any, error) {
	where, args := periodClause(period)

	query := fmt.Sprintf(`
		SELECT strftime('%%Y-%%m', created_at) AS month, COUNT(*) AS deal_count
		FROM deal_outcomes WHERE 1=1 %s
		GROUP BY month ORDER BY month ASC`, where)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type monthlyEntry struct {
		Month     string `json:"month"`
		DealCount int    `json:"deal_count"`
	}

	var entries []monthlyEntry
	for rows.Next() {
		var me monthlyEntry
		if err := rows.Scan(&me.Month, &me.DealCount); err != nil {
			return nil, err
		}
		entries = append(entries, me)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"monthly": entries,
		"total":   len(entries),
	}, nil
}

// ─── Helpers ───

func periodClause(period string) (string, []any) {
	return periodAndVendorClause(period, "")
}

func periodAndVendorClause(period, vendor string) (string, []any) {
	var clauses []string
	var args []any

	switch period {
	case "30d":
		clauses = append(clauses, "created_at >= datetime('now', '-30 days')")
	case "90d":
		clauses = append(clauses, "created_at >= datetime('now', '-90 days')")
	case "1y":
		clauses = append(clauses, "created_at >= datetime('now', '-12 months')")
	}

	if vendor != "" {
		clauses = append(clauses, "vendor = ?")
		args = append(args, vendor)
	}

	where := ""
	for _, c := range clauses {
		where += " AND " + c
	}

	if where == "" {
		return "", args
	}
	return where, args
}
