package metrics_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/metrics"
	_ "modernc.org/sqlite"
)

var sessionCounter int

func newTestStore(t *testing.T) *history.Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := history.NewStore(db)
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}
	return store
}

func seedSession(t *testing.T, store *history.Store, outcome string) {
	t.Helper()
	sessionCounter++
	ctx := context.Background()

	_, err := store.DB().ExecContext(ctx,
		`INSERT INTO negotiation_sessions (id, vendor, sku, strategy, budget, status, current_offer, list_price, rounds_complete, outcome, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		fmt.Sprintf("test-%s-%d", outcome, sessionCounter), "TestVendor", "TestSKU", "balanced", 100.0, "completed", 50.0, 100.0, 3, outcome)
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func seedDeal(t *testing.T, store *history.Store, listPrice, discountPct float64) {
	t.Helper()
	ctx := context.Background()

	_, err := store.DB().ExecContext(ctx,
		`INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		"TestVendor", "TestSKU", listPrice, listPrice*(1-discountPct/100), discountPct, 10, 12, "balanced", fmt.Sprintf("deal-session-%d", sessionCounter))
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}
}

func TestGenerateMetrics(t *testing.T) {
	store := newTestStore(t)
	eng := metrics.NewEngine(store)
	ctx := context.Background()

	// Seed sessions with outcomes
	seedSession(t, store, "won")
	seedSession(t, store, "won")
	seedSession(t, store, "lost")

	// Seed deals
	seedDeal(t, store, 100, 10)
	seedDeal(t, store, 200, 15)

	payload, err := eng.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if payload.Content == "" {
		t.Fatal("expected non-empty metrics payload")
	}

	if !contains(payload.Content, "# HELP negotiation_total") {
		t.Error("expected HELP for negotiation_total")
	}
	if !contains(payload.Content, `negotiation_total{status="won"} 2`) {
		t.Error("expected 2 won negotiations")
	}
	if !contains(payload.Content, `negotiation_total{status="lost"} 1`) {
		t.Error("expected 1 lost negotiation")
	}
	if !contains(payload.Content, "# HELP savings_total") {
		t.Error("expected HELP for savings_total")
	}
	if !contains(payload.Content, "# TYPE") {
		t.Error("expected TYPE declarations")
	}
}

func TestGenerateMetricsEmpty(t *testing.T) {
	store := newTestStore(t)
	eng := metrics.NewEngine(store)
	ctx := context.Background()

	payload, err := eng.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if payload.Content == "" {
		t.Fatal("expected non-empty metrics payload even with no data")
	}

	if !contains(payload.Content, `negotiation_total{status="won"} 0`) {
		t.Error("expected 0 won negotiations with no data")
	}
	if !contains(payload.Content, `negotiation_total{status="lost"} 0`) {
		t.Error("expected 0 lost negotiations with no data")
	}
	if !contains(payload.Content, "savings_total 0.00") {
		t.Error("expected savings_total 0.00 with no data")
	}
}

func TestFormatPrometheus(t *testing.T) {
	lines := []metrics.MetricLine{
		{Name: "test_metric", Value: 42, Labels: map[string]string{"env": "prod"}},
	}
	output := metrics.FormatPrometheus(lines)

	if !contains(output, "# HELP test_metric") {
		t.Error("expected HELP for test_metric")
	}
	if !contains(output, "# TYPE test_metric") {
		t.Error("expected TYPE for test_metric")
	}
	if !contains(output, `test_metric{env="prod"} 42`) {
		t.Error("expected formatted metric line")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
