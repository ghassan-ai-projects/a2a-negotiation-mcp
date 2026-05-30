package negotiation

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) (*Engine, *pricing.Store) {
	t.Helper()
	store, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// Insert test vendor + pricing
	ctx := context.Background()
	_, err = store.DB().ExecContext(ctx,
		"INSERT INTO vendors (name, category) VALUES (?, ?)",
		"Slack", "Communication")
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	var vid int64
	err = store.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", "Slack").Scan(&vid)
	if err != nil {
		t.Fatalf("get vendor id: %v", err)
	}
	_, err = store.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, vid, "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month")
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	engine := NewEngine(store)
	return engine, store
}

func TestGetStrategy(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"aggressive", true},
		{"balanced", true},
		{"conservative", true},
		{"nonexistent", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := GetStrategy(tc.name)
			if tc.expected && s == nil {
				t.Errorf("expected strategy %s to exist", tc.name)
			}
			if !tc.expected && s != nil {
				t.Errorf("expected strategy %s to be nil", tc.name)
			}
		})
	}
}

func TestAvailableStrategies(t *testing.T) {
	strategies := AvailableStrategies()
	if len(strategies) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(strategies))
	}

	// Verify aggressiveness values
	if s := strategies["aggressive"]; s.InitialDiscountPct != 0.30 {
		t.Errorf("aggressive initial discount: expected 0.30, got %f", s.InitialDiscountPct)
	}
	if s := strategies["conservative"]; s.InitialDiscountPct != 0.10 {
		t.Errorf("conservative initial discount: expected 0.10, got %f", s.InitialDiscountPct)
	}
}

func TestCreateSession(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	session, err := engine.CreateSession(ctx, "Slack", "Pro", "balanced", 0, nil)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if session.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", session.Vendor)
	}
	if session.Status != "active" {
		t.Errorf("expected status active, got %s", session.Status)
	}
	if session.CurrentOffer <= 0 {
		t.Errorf("expected positive initial offer, got %f", session.CurrentOffer)
	}
	// Balanced strategy: 20% off 8.75 = 7.00
	expectedOffer := 8.75 * 0.80
	if session.CurrentOffer != expectedOffer {
		t.Errorf("expected offer %f, got %f", expectedOffer, session.CurrentOffer)
	}
}

func TestCreateSession_InvalidStrategy(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	_, err := engine.CreateSession(ctx, "Slack", "Pro", "crazy", 0, nil)
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
}

func TestCreateSession_UnknownVendor(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	_, err := engine.CreateSession(ctx, "NonExistent", "", "balanced", 0, nil)
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}
}

func TestRunNegotiation_Accept(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// Create session with aggressive strategy (30% off 8.75 = 6.125)
	session, err := engine.CreateSession(ctx, "Slack", "Pro", "aggressive", 0, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	// Use auto-approve threshold to accept immediately
	result, rounds, err := engine.RunNegotiation(ctx, session, 0, 7.00)
	if err != nil {
		t.Fatalf("RunNegotiation: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("expected status completed, got %s", result.Status)
	}
	if result.Outcome != "accepted" {
		t.Errorf("expected outcome accepted, got %s", result.Outcome)
	}
	if len(rounds) < 1 {
		t.Errorf("expected at least 1 round, got %d", len(rounds))
	}
}

func TestRunNegotiation_BudgetAccept(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// Create session with balanced strategy + budget constraint
	session, err := engine.CreateSession(ctx, "Slack", "Pro", "balanced", 7.00, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	result, _, err := engine.RunNegotiation(ctx, session, 0, 0)
	if err != nil {
		t.Fatalf("RunNegotiation: %v", err)
	}
	if result.Outcome != "accepted" {
		t.Errorf("expected outcome accepted, got %s", result.Outcome)
	}
	if result.RoundsComplete <= 0 {
		t.Errorf("expected positive rounds, got %d", result.RoundsComplete)
	}
	if result.CurrentOffer > 7.00 {
		t.Errorf("expected current offer <= 7.00 (budget), got %f", result.CurrentOffer)
	}
}

func TestRunNegotiation_WalkAway(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	session, err := engine.CreateSession(ctx, "Slack", "Pro", "balanced", 2.00, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Session should walk away or reject since budget is way below market
	result, _, err := engine.RunNegotiation(ctx, session, 0, 0)
	if err != nil {
		t.Fatalf("RunNegotiation: %v", err)
	}
	// Should have some outcome
	if result.Outcome == "" {
		t.Error("expected non-empty outcome")
	}
}

func TestComputeSavings(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	estimate, err := engine.ComputeSavings(ctx, "Slack", 10000)
	if err != nil {
		t.Fatalf("ComputeSavings: %v", err)
	}
	if estimate.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", estimate.Vendor)
	}
	if estimate.CurrentSpend != 10000 {
		t.Errorf("expected spend 10000, got %f", estimate.CurrentSpend)
	}
	if estimate.SavingsPercentage <= 0 {
		t.Errorf("expected positive savings percentage, got %f", estimate.SavingsPercentage)
	}
}

func TestComputeSavings_UnknownVendor(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	_, err := engine.ComputeSavings(ctx, "NonExistent", 10000)
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}
}
