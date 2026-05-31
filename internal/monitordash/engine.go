package monitordash

import (
	"context"
	"math/rand"
	"time"
)

// Engine is a stateless engine for generating simulated live dashboard data.
type Engine struct{}

// NewEngine creates a new monitoring dashboard engine.
func NewEngine() *Engine {
	return &Engine{}
}

// GetDashboard generates simulated live monitoring data.
func (e *Engine) GetDashboard(_ context.Context) (*LiveDashboard, error) {
	now := time.Now().UTC()

	activeNegotiations := 3 + rand.Intn(13) // 3-15

	systemHealth := pickWeighted([]string{"healthy", "degraded", "healthy"}, []int{3, 1, 3})

	toolNames := []string{
		"negotiate_query_price",
		"negotiate_calculate_savings",
		"negotiate_create_session",
		"negotiate_run",
		"negotiate_history",
		"negotiate_strategies",
		"negotiate_run_parallel",
		"negotiate_create_group",
		"negotiate_join_group",
		"negotiate_compute_offer",
		"negotiate_group_status",
		"negotiate_add_contract",
		"negotiate_list_contracts",
	}

	calls := make([]ToolCallEntry, 10)
	for i := range calls {
		durationMs := 50 + rand.Intn(951) // 50-1000ms
		success := rand.Float64() >= 0.15

		t := rand.Intn(len(toolNames))
		callTime := now.Add(-time.Duration(i) * time.Second)

		calls[i] = ToolCallEntry{
			ToolName:   toolNames[t],
			DurationMs: durationMs,
			Success:    success,
			Timestamp:  callTime.Format(time.RFC3339),
		}
	}

	errorRate := rand.Float64() * 8.0 // 0.0-8.0
	uptimeSeconds := time.Now().Unix() % 86400
	totalTools := 150

	return &LiveDashboard{
		ActiveNegotiations: activeNegotiations,
		SystemHealth:       systemHealth,
		LastToolCalls:      calls,
		ErrorRate5Min:      errorRate,
		UptimeSeconds:      uptimeSeconds,
		TotalTools:         totalTools,
		Timestamp:          now.Format(time.RFC3339),
	}, nil
}

// pickWeighted selects a random string from values weighted by the corresponding weight in weights.
func pickWeighted(values []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return values[i]
		}
	}
	return values[len(values)-1]
}
