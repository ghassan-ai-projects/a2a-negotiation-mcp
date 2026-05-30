package winloss

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// Engine computes win/loss analytics from deal outcomes and sessions.
type Engine struct {
	historyStore *history.Store
	logger       *slog.Logger
}

// NewEngine creates a win/loss analysis engine.
func NewEngine(historyStore *history.Store, logger *slog.Logger) *Engine {
	return &Engine{historyStore: historyStore, logger: logger}
}

// Analyze generates a WinLossReport for the given filters.
// vendor and strategy are optional filters; period is "all", "30d", "90d", or "1y".
func (e *Engine) Analyze(ctx context.Context, vendor, strategy, period string) (*WinLossReport, error) {
	e.logger.Debug("winloss analyze", "vendor", vendor, "strategy", strategy, "period", period)

	where, args := buildWhere(vendor, strategy, period)

	if period == "" {
		period = "all"
	}

	// Pre-validate period
	switch period {
	case "30d", "90d", "1y", "all":
	default:
		return nil, fmt.Errorf("invalid period %q: use 30d, 90d, 1y, or all", period)
	}

	sqlWhere := joinWhere(where, "s")

	// Aggregate outcomes from sessions
	query := `SELECT s.outcome, COUNT(*) FROM negotiation_sessions s WHERE 1=1` + sqlWhere + ` GROUP BY s.outcome`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query session outcomes: %w", err)
	}
	defer rows.Close()

	var won, lost, pending int
	for rows.Next() {
		var outcome string
		var count int
		if err := rows.Scan(&outcome, &count); err != nil {
			return nil, fmt.Errorf("scan outcome: %w", err)
		}
		switch outcome {
		case "won":
			won = count
		case "lost":
			lost = count
		default:
			pending += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Also count deal_outcomes as closed-won deals
	dealWhere := joinWhere(where, "d")
	dealRows, err := e.historyStore.DB().QueryContext(ctx, `SELECT COUNT(*) FROM deal_outcomes d WHERE 1=1`+dealWhere, args...)
	if err == nil {
		var dealCount int
		if dealRows.Next() {
			dealRows.Scan(&dealCount)
		}
		dealRows.Close()
		won += dealCount
	}

	totalDeals := won + lost + pending
	winRate := 0.0
	if won+lost > 0 {
		winRate = float64(won) / float64(won+lost) * 100
	}

	report := &WinLossReport{
		Period:     period,
		TotalDeals: totalDeals,
		Won:        won,
		Lost:       lost,
		Pending:    pending,
		WinRate:    winRate,
	}

	// By strategy breakdown (only won/lost sessions)
	e.buildStrategyBreakdown(ctx, where, args, report)
	// By vendor breakdown
	e.buildVendorBreakdown(ctx, where, args, report)
	// Monthly trend
	e.buildMonthlyTrend(ctx, where, args, report)

	return report, nil
}

func (e *Engine) buildStrategyBreakdown(ctx context.Context, where string, args []any, report *WinLossReport) {
	sqlWhere := joinWhere(where, "s")
	query := `SELECT s.strategy,
		SUM(CASE WHEN s.outcome = 'won' THEN 1 ELSE 0 END) as won_count,
		SUM(CASE WHEN s.outcome = 'lost' THEN 1 ELSE 0 END) as lost_count
		FROM negotiation_sessions s WHERE 1=1` + sqlWhere + ` AND s.outcome IN ('won','lost')
		GROUP BY s.strategy`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		e.logger.Warn("strategy breakdown query failed", "error", err.Error())
		return
	}
	defer rows.Close()

	var breakdowns []StrategyBreakdown
	for rows.Next() {
		var strat string
		var w, l int
		if err := rows.Scan(&strat, &w, &l); err != nil {
			continue
		}
		sr := 0.0
		if w+l > 0 {
			sr = float64(w) / float64(w+l) * 100
		}
		breakdowns = append(breakdowns, StrategyBreakdown{
			Strategy: strat,
			Won:      w,
			Lost:     l,
			WinRate:  sr,
		})
	}
	report.ByStrategy = breakdowns
}

func (e *Engine) buildVendorBreakdown(ctx context.Context, where string, args []any, report *WinLossReport) {
	sqlWhere := joinWhere(where, "s")
	query := `SELECT s.vendor,
		SUM(CASE WHEN s.outcome = 'won' THEN 1 ELSE 0 END) as won_count,
		SUM(CASE WHEN s.outcome = 'lost' THEN 1 ELSE 0 END) as lost_count
		FROM negotiation_sessions s WHERE 1=1` + sqlWhere + ` AND s.outcome IN ('won','lost')
		GROUP BY s.vendor`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		e.logger.Warn("vendor breakdown query failed", "error", err.Error())
		return
	}
	defer rows.Close()

	var breakdowns []VendorBreakdown
	for rows.Next() {
		var v string
		var w, l int
		if err := rows.Scan(&v, &w, &l); err != nil {
			continue
		}
		vr := 0.0
		if w+l > 0 {
			vr = float64(w) / float64(w+l) * 100
		}
		breakdowns = append(breakdowns, VendorBreakdown{
			Vendor:  v,
			Won:     w,
			Lost:    l,
			WinRate: vr,
		})
	}
	report.ByVendor = breakdowns
}

func (e *Engine) buildMonthlyTrend(ctx context.Context, where string, args []any, report *WinLossReport) {
	sqlWhere := joinWhere(where, "s")
	query := `SELECT strftime('%Y-%m', s.created_at) as month,
		SUM(CASE WHEN s.outcome = 'won' THEN 1 ELSE 0 END) as won_count,
		SUM(CASE WHEN s.outcome = 'lost' THEN 1 ELSE 0 END) as lost_count
		FROM negotiation_sessions s WHERE 1=1` + sqlWhere + `
		AND s.created_at >= datetime('now', '-12 months')
		AND s.outcome IN ('won','lost')
		GROUP BY month ORDER BY month ASC`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		e.logger.Warn("monthly trend query failed", "error", err.Error())
		return
	}
	defer rows.Close()

	var trends []MonthTrend
	for rows.Next() {
		var month string
		var w, l int
		if err := rows.Scan(&month, &w, &l); err != nil {
			continue
		}
		mr := 0.0
		if w+l > 0 {
			mr = float64(w) / float64(w+l) * 100
		}
		trends = append(trends, MonthTrend{
			Month:   month,
			Won:     w,
			Lost:    l,
			WinRate: mr,
		})
	}
	report.MonthlyTrend = trends
}

// buildWhere builds the WHERE clause suffix and args for session-based queries.
// The returned where string uses "?" placeholders and references the "s" alias.
func buildWhere(vendor, strategy, period string) (string, []any) {
	where := ""
	var args []any
	if vendor != "" {
		where += " AND vendor = ?"
		args = append(args, vendor)
	}
	if strategy != "" {
		where += " AND strategy = ?"
		args = append(args, strategy)
	}
	switch period {
	case "30d":
		where += " AND created_at >= datetime('now', '-30 days')"
	case "90d":
		where += " AND created_at >= datetime('now', '-90 days')"
	case "1y":
		where += " AND created_at >= datetime('now', '-1 year')"
	}
	return where, args
}

// joinWhere prepends the table alias to column references in the WHERE clause.
// E.g., " AND vendor = ?" with alias "s" returns " AND s.vendor = ?"
func joinWhere(where, alias string) string {
	if where == "" {
		return ""
	}
	result := ""
	i := 0
	for i < len(where) {
		ch := where[i]
		// Skip delimiters and operators
		if ch == ' ' || ch == '(' || ch == ')' || ch == ',' || ch == '=' || ch == '>' || ch == '<' || ch == '\'' || ch == '"' {
			result += string(ch)
			i++
			continue
		}
		// Read the word
		start := i
		for i < len(where) && where[i] != ' ' && where[i] != ')' && where[i] != ',' && where[i] != '=' && where[i] != '>' && where[i] != '<' && where[i] != '\'' && where[i] != '"' {
			i++
		}
		word := where[start:i]
		switch word {
		case "vendor", "strategy", "created_at", "outcome":
			result += alias + "." + word
		default:
			result += word
		}
	}
	return result
}
