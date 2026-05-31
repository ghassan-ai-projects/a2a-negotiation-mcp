package shutdown

import (
	"database/sql"
	"time"
)

// closable is any resource with a Close() error method.
type Closable interface {
	Close() error
}

// Engine performs graceful shutdown of server resources.
type Engine struct{}

// NewEngine creates a shutdown engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Shutdown closes the database and all provided closable resources,
// returning a ShutdownResult with timing and cleanup details.
func (e *Engine) Shutdown(db *sql.DB, stores []Closable) ShutdownResult {
	start := time.Now()
	var cleaned []string

	// Close each closable resource
	for i, store := range stores {
		if store == nil {
			continue
		}
		if err := store.Close(); err != nil {
			cleaned = append(cleaned, describeResource(i, err))
		} else {
			cleaned = append(cleaned, describeResource(i, nil))
		}
	}

	// Close database connection
	if db != nil {
		if err := db.Close(); err != nil {
			cleaned = append(cleaned, "database (close error: "+err.Error()+")")
		} else {
			cleaned = append(cleaned, "database")
		}
	}

	duration := time.Since(start).Milliseconds()

	return ShutdownResult{
		Status:           "shutdown_complete",
		ResourcesCleaned: cleaned,
		DurationMs:       duration,
	}
}

func describeResource(idx int, err error) string {
	name := resourceName(idx)
	if err != nil {
		return name + " (error: " + err.Error() + ")"
	}
	return name
}

func resourceName(idx int) string {
	names := []string{
		"pricing_store",
		"history_store",
		"group_store",
		"sell_store",
		"calendar_store",
		"marketplace_store",
		"health_store",
		"sla_store",
		"webhook_store",
		"roi_store",
		"trends_store",
		"export_store",
		"notify_store",
		"budget_store",
		"price_alert_store",
		"budget_alert_store",
		"batch_negotiation_store",
		"workspace_store",
		"audit_log_store",
		"contract_templates_store",
		"shared_strategies_store",
		"notes_store",
		"approvals_store",
		"budget_mgmt_store",
		"spending_caps_store",
		"savings_realization_store",
		"cost_allocation_store",
		"gamification_store",
		"rate_limit_dashboard_store",
		"tool_stats_store",
	}
	if idx >= 0 && idx < len(names) {
		return names[idx]
	}
	return "store_" + time.Now().Format("150405")
}
