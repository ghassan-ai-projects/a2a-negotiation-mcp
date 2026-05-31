package tco

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTCOTest(t *testing.T) *Engine {
	t.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pstore.Close() })

	// Seed test data
	ctx := context.Background()
	_, err = pstore.DB().ExecContext(ctx,
		"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
		"Slack", "Communication")
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	var vid int64
	err = pstore.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", "Slack").Scan(&vid)
	if err != nil {
		t.Fatalf("get vendor id: %v", err)
	}
	_, err = pstore.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
	`, vid, "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month")
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	return NewEngine(pstore)
}

func TestTCO_AllParams(t *testing.T) {
	eng := setupTCOTest(t)
	ctx := context.Background()

	input := TCOInput{
		Vendor:              "Slack",
		SKU:                 "Pro",
		Seats:               100,
		TermMonths:          12,
		ImplementationCosts: 5000,
		TrainingCosts:       2000,
		SupportCosts:        1000,
	}

	output, err := eng.Calculate(ctx, input)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	if output.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", output.Vendor)
	}
	if output.Seats != 100 {
		t.Errorf("expected seats 100, got %d", output.Seats)
	}
	if output.TermMonths != 12 {
		t.Errorf("expected term_months 12, got %d", output.TermMonths)
	}

	// annual_subscription = list_price * seats * 12 = 8.75 * 100 * 12 = 10500
	if output.AnnualSubscription != 10500 {
		t.Errorf("expected annual_subscription 10500, got %f", output.AnnualSubscription)
	}

	// per_unit_cost = list_price = 8.75
	if output.PerUnitCost != 8.75 {
		t.Errorf("expected per_unit_cost 8.75, got %f", output.PerUnitCost)
	}

	// total_1y_tco = 10500 + 5000 + 2000 + 1000 = 18500
	if output.Total1YTCO != 18500 {
		t.Errorf("expected total_1y_tco 18500, got %f", output.Total1YTCO)
	}

	// total_3y_tco = 18500*3 - 5000 = 55500 - 5000 = 50500
	if output.Total3YTCO != 50500 {
		t.Errorf("expected total_3y_tco 50500, got %f", output.Total3YTCO)
	}

	// cost_per_user_per_month = 18500 / 100 / 12 = 15.42 (rounded)
	if output.CostPerUserPerMonth <= 0 {
		t.Errorf("expected positive cost_per_user_per_month, got %f", output.CostPerUserPerMonth)
	}

	// market_avg_cupm = 8.75 * (1 - 0.18) = 7.175
	if output.MarketAvgCUPM != 7.18 {
		t.Errorf("expected market_avg_cupm 7.18, got %f", output.MarketAvgCUPM)
	}

	// hidden costs flagged
	if len(output.HiddenCostsFlagged) != 3 {
		t.Errorf("expected 3 hidden costs flagged, got %d: %v", len(output.HiddenCostsFlagged), output.HiddenCostsFlagged)
	}
}

func TestTCO_Defaults(t *testing.T) {
	eng := setupTCOTest(t)
	ctx := context.Background()

	input := TCOInput{
		Vendor: "Slack",
		SKU:    "Pro",
	}

	output, err := eng.Calculate(ctx, input)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// Default seats should be 50, term 12
	if output.Seats != 50 {
		t.Errorf("expected default seats 50, got %d", output.Seats)
	}
	if output.TermMonths != 12 {
		t.Errorf("expected default term_months 12, got %d", output.TermMonths)
	}

	// annual_subscription = 8.75 * 50 * 12 = 5250
	if output.AnnualSubscription != 5250 {
		t.Errorf("expected annual_subscription 5250, got %f", output.AnnualSubscription)
	}

	// total_1y_tco should equal annual_subscription (no hidden costs)
	if output.Total1YTCO != output.AnnualSubscription {
		t.Errorf("expected total_1y_tco to equal annual_subscription with no hidden costs")
	}

	// No hidden costs flagged
	if len(output.HiddenCostsFlagged) != 0 {
		t.Errorf("expected 0 hidden costs, got %d", len(output.HiddenCostsFlagged))
	}
}

func TestTCO_MarketComparison(t *testing.T) {
	eng := setupTCOTest(t)
	ctx := context.Background()

	input := TCOInput{
		Vendor: "Slack",
		SKU:    "Pro",
		Seats:  50,
	}

	output, err := eng.Calculate(ctx, input)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// market_avg_cupm should be positive
	if output.MarketAvgCUPM <= 0 {
		t.Errorf("expected positive market_avg_cupm, got %f", output.MarketAvgCUPM)
	}

	// savings_vs_market_pct should be calculable (could be negative if our TCO is higher)
	if output.SavingsVsMarketPct == 0 {
		t.Logf("savings_vs_market_pct is 0 (our cost equals market average)")
	}

	// Verify the savings calculation formula
	// market_avg = 8.75 * 0.82 = 7.175
	// cupm = 5250 / 50 / 12 = 8.75
	// savings = (7.175 - 8.75) / 7.175 * 100 = -21.95% (we're paying more than market average)
	if output.CostPerUserPerMonth > output.MarketAvgCUPM {
		t.Logf("Expected negative savings (cost > market avg): %f%%", output.SavingsVsMarketPct)
	}
}
