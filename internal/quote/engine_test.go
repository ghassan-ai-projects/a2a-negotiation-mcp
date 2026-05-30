package quote

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) (*Engine, *pricing.Store) {
	t.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pstore.Close() })

	seedTestData(t, pstore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(pstore, logger)
	return eng, pstore
}

func seedTestData(t *testing.T, store *pricing.Store) {
	t.Helper()
	ctx := context.Background()

	vendors := []struct {
		name, category, sku, desc string
		listPrice, minObs, maxObs float64
		typicalPct                float64
		unit                      string
	}{
		{"Salesforce", "CRM", "Enterprise", "Enterprise per seat", 165.00, 110.00, 155.00, 28, "per_seat_month"},
		{"Slack", "Communication", "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month"},
	}

	for _, v := range vendors {
		_, err := store.DB().ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
			v.name, v.category)
		if err != nil {
			t.Fatalf("insert vendor %s: %v", v.name, err)
		}
		var vid int64
		err = store.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", v.name).Scan(&vid)
		if err != nil {
			t.Fatalf("get vendor id %s: %v", v.name, err)
		}
		_, err = store.DB().ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
		`, vid, v.sku, v.desc, v.listPrice, v.minObs, v.maxObs, v.typicalPct, v.unit)
		if err != nil {
			t.Fatalf("insert pricing %s/%s: %v", v.name, v.sku, err)
		}
	}
}

func TestParseSalesforceQuote(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "$145/seat/month for 50 seats of Salesforce Enterprise",
	}

	result, err := eng.AnalyzeQuote(ctx, input)
	if err != nil {
		t.Fatalf("AnalyzeQuote failed: %v", err)
	}

	if result.Quote.Vendor != "Salesforce" {
		t.Errorf("expected vendor Salesforce, got %s", result.Quote.Vendor)
	}
	if result.Quote.Seats != 50 {
		t.Errorf("expected seats 50, got %d", result.Quote.Seats)
	}
	if result.Quote.PricePerUnit != 145.00 {
		t.Errorf("expected price 145.00, got %.2f", result.Quote.PricePerUnit)
	}
	if result.Quote.TermMonths != 1 {
		t.Errorf("expected term 1 (monthly), got %d", result.Quote.TermMonths)
	}
	if result.Quote.ListPrice != 165.00 {
		t.Errorf("expected list price 165.00, got %.2f", result.Quote.ListPrice)
	}
	if len(result.MarketRange) != 2 {
		t.Errorf("expected market range with 2 values, got %d", len(result.MarketRange))
	}
}

func TestParseSlackQuote(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "Slack Pro - $8.75/user/month, 100 users, annual billing",
	}

	result, err := eng.AnalyzeQuote(ctx, input)
	if err != nil {
		t.Fatalf("AnalyzeQuote failed: %v", err)
	}

	if result.Quote.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", result.Quote.Vendor)
	}
	if result.Quote.Seats != 100 {
		t.Errorf("expected seats 100, got %d", result.Quote.Seats)
	}
	if result.Quote.PricePerUnit != 8.75 {
		t.Errorf("expected price 8.75, got %.2f", result.Quote.PricePerUnit)
	}
	// "annual billing" → TermMonths = 12
	if result.Quote.TermMonths != 12 {
		t.Errorf("expected term 12 (annual), got %d", result.Quote.TermMonths)
	}
}

func TestParseWithExplicitVendorSKU(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	// Explicit vendor/sku should override regex extraction
	input := QuoteInput{
		RawText: "Some random text with $50/user for 20 people",
		Vendor:  "Salesforce",
		SKU:     "Enterprise",
	}

	result, err := eng.AnalyzeQuote(ctx, input)
	if err != nil {
		t.Fatalf("AnalyzeQuote failed: %v", err)
	}

	if result.Quote.Vendor != "Salesforce" {
		t.Errorf("expected vendor Salesforce from explicit input, got %s", result.Quote.Vendor)
	}
	if result.Quote.SKU != "Enterprise" {
		t.Errorf("expected SKU Enterprise from explicit input, got %s", result.Quote.SKU)
	}
	// The regex still extracts price/quantity from text
	if result.Quote.PricePerUnit != 50.00 {
		t.Errorf("expected price 50.00 from text, got %.2f", result.Quote.PricePerUnit)
	}
	if result.Quote.Seats != 20 {
		t.Errorf("expected seats 20 from text, got %d", result.Quote.Seats)
	}
}

func TestUnknownVendor(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "$10/user/month for 5 users of SomeUnknownVendor",
	}

	result, err := eng.AnalyzeQuote(ctx, input)
	if err == nil {
		t.Fatal("expected error for unknown vendor, got nil")
	}
	if result != nil {
		t.Fatal("expected nil result for unknown vendor")
	}
}

func TestUnknownVendorReturnsError(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "random text without any recognizable pattern",
	}

	_, err := eng.AnalyzeQuote(ctx, input)
	if err == nil {
		t.Fatal("expected error for unparseable input")
	}
}

func TestCrossReferenceMarketRange(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "$145/seat/month for 50 seats of Salesforce Enterprise",
	}

	result, err := eng.AnalyzeQuote(ctx, input)
	if err != nil {
		t.Fatalf("AnalyzeQuote failed: %v", err)
	}

	// Salesforce Enterprise: market range [110, 155]
	if len(result.MarketRange) != 2 {
		t.Fatalf("expected 2 market range values, got %d", len(result.MarketRange))
	}
	if result.MarketRange[0] != 110.00 {
		t.Errorf("expected market min 110.00, got %.2f", result.MarketRange[0])
	}
	if result.MarketRange[1] != 155.00 {
		t.Errorf("expected market max 155.00, got %.2f", result.MarketRange[1])
	}

	// Counter-offer should be within market range
	if result.CounterOfferMin > result.CounterOfferMax {
		t.Errorf("counter-offer min (%.2f) > max (%.2f)", result.CounterOfferMin, result.CounterOfferMax)
	}
}

func TestGenerateCounterOfferContainsKeyPoints(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	input := QuoteInput{
		RawText: "$145/seat/month for 50 seats of Salesforce Enterprise",
	}

	analysis, err := eng.AnalyzeQuote(ctx, input)
	if err != nil {
		t.Fatalf("AnalyzeQuote failed: %v", err)
	}

	text, err := eng.GenerateCounterOffer(ctx, analysis)
	if err != nil {
		t.Fatalf("GenerateCounterOffer failed: %v", err)
	}

	// Check key negotiation points in output
	checkpoints := []string{
		"COUNTER-OFFER",
		"Salesforce",
		"$145.00",
		"50",
		"Market Range",
		"Suggested Range",
	}
	for _, cp := range checkpoints {
		if !contains(text, cp) {
			t.Errorf("expected counter-offer to contain %q", cp)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
