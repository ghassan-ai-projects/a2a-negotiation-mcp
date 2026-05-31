package strategycomparison

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func TestCompareThreeStrategies(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()
	seedTestData(t, pstore)

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	result, err := eng.Compare(context.Background(), StrategyComparisonRequest{
		Vendor:     "Slack",
		SKU:        "Pro",
		Strategies: []string{"aggressive", "balanced", "conservative"},
	})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if result.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", result.Vendor)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}

	// Verify results have proper risk levels
	riskMap := map[string]string{}
	for _, r := range result.Results {
		riskMap[r.Strategy] = r.RiskLevel
	}
	if riskMap["aggressive"] != "high" {
		t.Errorf("expected aggressive risk=high, got %s", riskMap["aggressive"])
	}
	if riskMap["balanced"] != "medium" {
		t.Errorf("expected balanced risk=medium, got %s", riskMap["balanced"])
	}
	if riskMap["conservative"] != "low" {
		t.Errorf("expected conservative risk=low, got %s", riskMap["conservative"])
	}

	// Aggressive should have highest expected savings
	if result.Results[0].ExpectedSavings < result.Results[2].ExpectedSavings {
		// The best strategy should be aggressive (highest savings)
		if result.BestStrategy != "aggressive" {
			t.Errorf("expected best_strategy aggressive, got %s", result.BestStrategy)
		}
	}
}

func TestCompareSingleStrategy(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()
	seedTestData(t, pstore)

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	result, err := eng.Compare(context.Background(), StrategyComparisonRequest{
		Vendor:     "Slack",
		SKU:        "Pro",
		Strategies: []string{"balanced"},
	})
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Strategy != "balanced" {
		t.Errorf("expected strategy balanced, got %s", result.Results[0].Strategy)
	}
	if result.Results[0].RiskLevel != "medium" {
		t.Errorf("expected risk medium, got %s", result.Results[0].RiskLevel)
	}
	if result.Results[0].ExpectedSavings <= 0 {
		t.Errorf("expected positive expected_savings, got %f", result.Results[0].ExpectedSavings)
	}
	if result.BestStrategy != "balanced" {
		t.Errorf("expected best_strategy balanced, got %s", result.BestStrategy)
	}
}

func TestCompareUnknownStrategy(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()
	seedTestData(t, pstore)

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	eng := NewEngine(pstore.DB(), logger)

	_, err = eng.Compare(context.Background(), StrategyComparisonRequest{
		Vendor:     "Slack",
		SKU:        "Pro",
		Strategies: []string{"unknown_strategy"},
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
}

func seedTestData(t *testing.T, store *pricing.Store) {
	t.Helper()
	ctx := context.Background()
	store.DB().ExecContext(ctx,
		"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
		"Slack", "Communication")
	store.DB().ExecContext(ctx,
		"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
		"GitHub", "Developer")

	var vid int64
	store.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", "Slack").Scan(&vid)
	store.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
	`, vid, "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month")
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
func (discard) Close() error                { return nil }
