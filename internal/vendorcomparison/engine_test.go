package vendorcomparison

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func seedPricingData(t *testing.T, store *pricing.Store) {
	t.Helper()
	ctx := context.Background()
	vendors := []struct {
		name, category, sku, desc string
		listPrice, minObs, maxObs float64
		typicalPct                float64
		unit                      string
	}{
		{"Slack", "Communication", "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month"},
		{"Slack", "Communication", "Enterprise", "Enterprise Grid", 28.00, 20.00, 26.00, 25, "per_seat_month"},
		{"GitHub", "Developer", "Team", "Team plan", 4.00, 3.00, 3.80, 15, "per_seat_month"},
		{"Salesforce", "CRM", "Enterprise", "Enterprise per seat", 165.00, 110.00, 155.00, 28, "per_seat_month"},
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

func TestCompareByCategory(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()
	seedPricingData(t, pstore)

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	result, err := eng.Compare(context.Background(), ComparisonRequest{Category: "Communication", Seats: 10})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if result.Category != "Communication" {
		t.Errorf("expected category Communication, got %s", result.Category)
	}
	if len(result.Comparisons) == 0 {
		t.Fatal("expected at least one comparison")
	}
	if result.TopPick == "" {
		t.Error("expected top_pick to be non-empty")
	}
	if result.AvgPrice <= 0 {
		t.Errorf("expected positive avg_price, got %f", result.AvgPrice)
	}

	// Verify comparisons are sorted by annual cost ascending
	for i := 1; i < len(result.Comparisons); i++ {
		if result.Comparisons[i].AnnualCost < result.Comparisons[i-1].AnnualCost {
			t.Errorf("comparisons not sorted by annual cost: %f < %f",
				result.Comparisons[i].AnnualCost, result.Comparisons[i-1].AnnualCost)
		}
	}
}

func TestCompareEmptyCategory(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()
	seedPricingData(t, pstore)

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	result, err := eng.Compare(context.Background(), ComparisonRequest{Category: "Unknown", Seats: 10})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Comparisons) != 0 {
		t.Errorf("expected empty comparisons for unknown category, got %d", len(result.Comparisons))
	}
	if result.TopPick != "" {
		t.Errorf("expected empty top_pick for unknown category, got %s", result.TopPick)
	}
}

func TestCompareHandlesUnknownCategory(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	result, err := eng.Compare(context.Background(), ComparisonRequest{Category: "Nonexistent", Seats: 10})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	if len(result.Comparisons) != 0 {
		t.Errorf("expected empty for nonexistent category, got %d", len(result.Comparisons))
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
func (discard) Close() error                { return nil }
