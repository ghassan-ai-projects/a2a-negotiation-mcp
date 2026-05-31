package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// Engine generates Prometheus-format metrics from the history store.
type Engine struct {
	historyStore *history.Store
}

// NewEngine creates a metrics engine backed by a history store.
func NewEngine(historyStore *history.Store) *Engine {
	return &Engine{historyStore: historyStore}
}

// Generate queries the history store and returns a MetricsPayload in
// Prometheus exposition format.
func (e *Engine) Generate(ctx context.Context) (MetricsPayload, error) {
	var lines []string

	// Negotiation totals by outcome
	lines = append(lines, "# HELP negotiation_total Total negotiations by outcome")
	lines = append(lines, "# TYPE negotiation_total counter")

	won, lost, err := e.outcomeCounts(ctx)
	if err != nil {
		return MetricsPayload{}, fmt.Errorf("outcome counts: %w", err)
	}
	lines = append(lines, fmt.Sprintf(`negotiation_total{status="won"} %d`, won))
	lines = append(lines, fmt.Sprintf(`negotiation_total{status="lost"} %d`, lost))

	// Webhook test count
	webhookCount, err := e.webhookTestCount(ctx)
	if err != nil {
		return MetricsPayload{}, fmt.Errorf("webhook count: %w", err)
	}
	lines = append(lines, "")
	lines = append(lines, "# HELP webhook_tests_total Total webhook test requests")
	lines = append(lines, "# TYPE webhook_tests_total counter")
	lines = append(lines, fmt.Sprintf("webhook_tests_total %d", webhookCount))

	// Deal counts
	dealCount, err := e.dealCount(ctx)
	if err != nil {
		return MetricsPayload{}, fmt.Errorf("deal count: %w", err)
	}
	lines = append(lines, "")
	lines = append(lines, "# HELP deal_outcomes_total Total deal outcomes recorded")
	lines = append(lines, "# TYPE deal_outcomes_total counter")
	lines = append(lines, fmt.Sprintf("deal_outcomes_total %d", dealCount))

	// Savings total
	savings, err := e.totalSavings(ctx)
	if err != nil {
		return MetricsPayload{}, fmt.Errorf("total savings: %w", err)
	}
	lines = append(lines, "")
	lines = append(lines, "# HELP savings_total Cumulative savings from all deals")
	lines = append(lines, "# TYPE savings_total gauge")
	lines = append(lines, fmt.Sprintf("savings_total %.2f", savings))

	// Active sessions
	active, err := e.activeSessionCount(ctx)
	if err != nil {
		return MetricsPayload{}, fmt.Errorf("active sessions: %w", err)
	}
	lines = append(lines, "")
	lines = append(lines, "# HELP active_sessions Currently active negotiation sessions")
	lines = append(lines, "# TYPE active_sessions gauge")
	lines = append(lines, fmt.Sprintf("active_sessions %d", active))

	// Total tools registered
	lines = append(lines, "")
	lines = append(lines, "# HELP tools_total Total registered MCP tools")
	lines = append(lines, "# TYPE tools_total gauge")

	return MetricsPayload{
		Content: strings.Join(lines, "\n") + "\n",
	}, nil
}

func (e *Engine) outcomeCounts(ctx context.Context) (won, lost int64, err error) {
	rows, err := e.historyStore.DB().QueryContext(ctx,
		`SELECT outcome, COUNT(*) FROM negotiation_sessions GROUP BY outcome`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var outcome string
		var count int64
		if err := rows.Scan(&outcome, &count); err != nil {
			return 0, 0, err
		}
		switch strings.ToLower(outcome) {
		case "won":
			won = count
		case "lost":
			lost = count
		}
	}
	return won, lost, rows.Err()
}

func (e *Engine) webhookTestCount(ctx context.Context) (int64, error) {
	var count int64
	err := e.historyStore.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM webhook_configs`).Scan(&count)
	if err != nil {
		// Table may not exist if webhooks were never initialized
		return 0, nil
	}
	return count, nil
}

func (e *Engine) dealCount(ctx context.Context) (int64, error) {
	var count int64
	err := e.historyStore.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM deal_outcomes`).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

func (e *Engine) totalSavings(ctx context.Context) (float64, error) {
	var total *float64
	err := e.historyStore.DB().QueryRowContext(ctx,
		`SELECT SUM(list_price * discount_pct / 100.0 * CAST(seats AS REAL) * CAST(term_months AS REAL)) FROM deal_outcomes`).Scan(&total)
	if err != nil {
		return 0, nil
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

func (e *Engine) activeSessionCount(ctx context.Context) (int64, error) {
	var count int64
	err := e.historyStore.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM negotiation_sessions WHERE status NOT IN ('completed', 'cancelled')`).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// ── helpers for testing ──

func FormatPrometheus(lines []MetricLine) string {
	seen := make(map[string]bool)
	var b strings.Builder

	// Help + type — emit once per metric name
	for _, l := range lines {
		if !seen[l.Name] {
			seen[l.Name] = true
			fmt.Fprintf(&b, "# HELP %s \n", l.Name)
			fmt.Fprintf(&b, "# TYPE %s gauge\n", l.Name)
		}
		if len(l.Labels) == 0 {
			fmt.Fprintf(&b, "%s %g\n", l.Name, l.Value)
		} else {
			parts := make([]string, 0, len(l.Labels))
			for k, v := range l.Labels {
				parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
			}
			sort.Strings(parts)
			fmt.Fprintf(&b, "%s{%s} %g\n", l.Name, strings.Join(parts, ","), l.Value)
		}
	}
	return b.String()
}
