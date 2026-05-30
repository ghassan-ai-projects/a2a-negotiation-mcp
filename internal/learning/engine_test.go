package learning

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupLearningTest(t *testing.T) *Engine {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	histStore, err := history.NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	logger := slog.Default()
	eng, err := NewEngine(histStore, logger)
	if err != nil {
		t.Fatalf("learning.NewEngine: %v", err)
	}
	return eng
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func seedOutcomes(t *testing.T, eng *Engine, vendor string) {
	t.Helper()
	ctx := context.Background()

	outcomes := []StrategyOutcome{
		{Vendor: vendor, SKU: "Pro", Strategy: "aggressive", DiscountPct: 0.30, RoundsComplete: 2, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 6.12, Timestamp: parseTime("2026-01-15T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "aggressive", DiscountPct: 0.28, RoundsComplete: 3, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 6.30, Timestamp: parseTime("2026-02-10T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "aggressive", DiscountPct: 0.32, RoundsComplete: 1, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 5.95, Timestamp: parseTime("2026-03-05T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "aggressive", DiscountPct: 0.31, RoundsComplete: 2, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 6.03, Timestamp: parseTime("2026-05-10T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "balanced", DiscountPct: 0.20, RoundsComplete: 4, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 7.00, Timestamp: parseTime("2026-01-20T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "balanced", DiscountPct: 0.18, RoundsComplete: 5, Outcome: "walked_away", TotalBefore: 8.75, TotalAfter: 7.17, Timestamp: parseTime("2026-02-20T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "conservative", DiscountPct: 0.10, RoundsComplete: 6, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 7.87, Timestamp: parseTime("2026-03-10T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "conservative", DiscountPct: 0.12, RoundsComplete: 5, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 7.70, Timestamp: parseTime("2026-04-01T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "conservative", DiscountPct: 0.11, RoundsComplete: 6, Outcome: "accepted", TotalBefore: 8.75, TotalAfter: 7.78, Timestamp: parseTime("2026-04-15T10:00:00Z")},
		{Vendor: vendor, SKU: "Pro", Strategy: "conservative", DiscountPct: 0.09, RoundsComplete: 7, Outcome: "walked_away", TotalBefore: 8.75, TotalAfter: 7.96, Timestamp: parseTime("2026-05-01T10:00:00Z")},
	}

	for _, o := range outcomes {
		if err := eng.RecordOutcome(ctx, o); err != nil {
			t.Fatalf("RecordOutcome: %v", err)
		}
	}
}

func TestRecordAndGetRecommendation(t *testing.T) {
	eng := setupLearningTest(t)
	ctx := context.Background()

	seedOutcomes(t, eng, "Slack")

	rec, err := eng.GetRecommendation(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}

	if rec.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", rec.Vendor)
	}
	if rec.RecommendedStrategy != "aggressive" {
		t.Errorf("expected recommended strategy 'aggressive' (highest avg discount ~0.30), got %s", rec.RecommendedStrategy)
	}
	if rec.TotalDeals != 10 {
		t.Errorf("expected 10 total deals, got %d", rec.TotalDeals)
	}
	// 4 aggressive deals => confidence medium (3-9)
	if rec.Confidence != "medium" {
		t.Errorf("expected confidence 'medium' (4 aggressive deals), got %s", rec.Confidence)
	}

	if len(rec.Breakdown) != 3 {
		t.Errorf("expected 3 strategies in breakdown, got %d", len(rec.Breakdown))
	}

	agg := rec.Breakdown["aggressive"]
	bal := rec.Breakdown["balanced"]
	con := rec.Breakdown["conservative"]

	if agg.AvgDiscount <= 0 {
		t.Errorf("expected positive avg discount for aggressive, got %f", agg.AvgDiscount)
	}
	if agg.AvgDiscount <= bal.AvgDiscount {
		t.Errorf("expected aggressive avg discount > balanced avg discount")
	}
	if agg.AvgDiscount <= con.AvgDiscount {
		t.Errorf("expected aggressive avg discount > conservative avg discount")
	}
	if agg.TotalDeals != 4 {
		t.Errorf("expected 4 aggressive deals, got %d", agg.TotalDeals)
	}
}

func TestGetRecommendation_NoDeals(t *testing.T) {
	eng := setupLearningTest(t)
	ctx := context.Background()

	rec, err := eng.GetRecommendation(ctx, "UnknownVendor")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}

	if rec.RecommendedStrategy != "balanced" {
		t.Errorf("expected fallback strategy 'balanced', got %s", rec.RecommendedStrategy)
	}
	if rec.Confidence != "low" {
		t.Errorf("expected confidence 'low', got %s", rec.Confidence)
	}
	if rec.TotalDeals != 0 {
		t.Errorf("expected 0 deals, got %d", rec.TotalDeals)
	}
	if len(rec.Breakdown) != 0 {
		t.Errorf("expected empty breakdown, got %d items", len(rec.Breakdown))
	}
}

func TestGetRecommendation_PicksHighestAvgDiscount(t *testing.T) {
	eng := setupLearningTest(t)
	ctx := context.Background()

	outcomes := []StrategyOutcome{
		{Vendor: "MultiVendor", SKU: "X", Strategy: "aggressive", DiscountPct: 0.25, RoundsComplete: 3, Outcome: "accepted", TotalBefore: 100, TotalAfter: 75},
		{Vendor: "MultiVendor", SKU: "X", Strategy: "balanced", DiscountPct: 0.20, RoundsComplete: 4, Outcome: "accepted", TotalBefore: 100, TotalAfter: 80},
		{Vendor: "MultiVendor", SKU: "X", Strategy: "conservative", DiscountPct: 0.15, RoundsComplete: 5, Outcome: "accepted", TotalBefore: 100, TotalAfter: 85},
		{Vendor: "MultiVendor", SKU: "X", Strategy: "balanced", DiscountPct: 0.22, RoundsComplete: 4, Outcome: "accepted", TotalBefore: 100, TotalAfter: 78},
		{Vendor: "MultiVendor", SKU: "X", Strategy: "aggressive", DiscountPct: 0.27, RoundsComplete: 2, Outcome: "accepted", TotalBefore: 100, TotalAfter: 73},
	}
	for _, o := range outcomes {
		if err := eng.RecordOutcome(ctx, o); err != nil {
			t.Fatalf("RecordOutcome: %v", err)
		}
	}

	rec, err := eng.GetRecommendation(ctx, "MultiVendor")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}

	if rec.RecommendedStrategy != "aggressive" {
		t.Errorf("expected 'aggressive' (highest avg discount 0.26), got %s", rec.RecommendedStrategy)
	}
	if rec.Confidence != "low" {
		t.Errorf("expected confidence 'low' (<3 aggressive deals), got %s", rec.Confidence)
	}
}

func TestGetGlobalInsights(t *testing.T) {
	eng := setupLearningTest(t)
	ctx := context.Background()

	// No data yet
	insights, err := eng.GetGlobalInsights(ctx)
	if err != nil {
		t.Fatalf("GetGlobalInsights: %v", err)
	}

	totalOut, _ := insights["total_outcomes"].(int)
	if totalOut != 0 {
		t.Errorf("expected 0 outcomes, got %d", totalOut)
	}

	// Seed data
	seedOutcomes(t, eng, "Slack")
	outcomes := []StrategyOutcome{
		{Vendor: "GitHub", SKU: "Enterprise", Strategy: "aggressive", DiscountPct: 0.22, RoundsComplete: 3, Outcome: "accepted", TotalBefore: 21.00, TotalAfter: 16.38},
		{Vendor: "GitHub", SKU: "Enterprise", Strategy: "balanced", DiscountPct: 0.18, RoundsComplete: 4, Outcome: "accepted", TotalBefore: 21.00, TotalAfter: 17.22},
	}
	for _, o := range outcomes {
		if err := eng.RecordOutcome(ctx, o); err != nil {
			t.Fatalf("RecordOutcome: %v", err)
		}
	}

	insights, err = eng.GetGlobalInsights(ctx)
	if err != nil {
		t.Fatalf("GetGlobalInsights: %v", err)
	}

	totalOut, _ = insights["total_outcomes"].(int)
	if totalOut != 12 {
		t.Errorf("expected 12 outcomes, got %d", totalOut)
	}

	// strategies is a []strategyStats — cannot type-assert to []interface{}
	// Check via reflection on the map value
	overallWinRate, ok := insights["overall_win_rate"].(float64)
	if !ok {
		t.Fatal("expected overall_win_rate to be a float64")
	}
	if overallWinRate <= 0 {
		t.Errorf("expected positive win rate, got %f", overallWinRate)
	}
}

func TestConfidenceLevels(t *testing.T) {
	eng := setupLearningTest(t)
	ctx := context.Background()

	// Low confidence: 1 deal => best count = 1 < 3 => "low"
	outcome := StrategyOutcome{Vendor: "LowVendor", SKU: "A", Strategy: "aggressive", DiscountPct: 0.30, RoundsComplete: 1, Outcome: "accepted", TotalBefore: 100, TotalAfter: 70}
	if err := eng.RecordOutcome(ctx, outcome); err != nil {
		t.Fatalf("RecordOutcome: %v", err)
	}
	rec, err := eng.GetRecommendation(ctx, "LowVendor")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}
	if rec.Confidence != "low" {
		t.Errorf("1 deal: expected 'low', got %s", rec.Confidence)
	}

	// Medium confidence: 5 deals => best count = 5, 3 <= 5 < 10 => "medium"
	for i := 0; i < 4; i++ {
		if err := eng.RecordOutcome(ctx, StrategyOutcome{Vendor: "MedVendor", SKU: "A", Strategy: "aggressive", DiscountPct: 0.30, RoundsComplete: 2, Outcome: "accepted", TotalBefore: 100, TotalAfter: 70}); err != nil {
			t.Fatalf("RecordOutcome: %v", err)
		}
	}
	rec, err = eng.GetRecommendation(ctx, "MedVendor")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}
	if rec.Confidence != "medium" {
		t.Errorf("5 deals: expected 'medium', got %s", rec.Confidence)
	}

	// High confidence: 10 deals => best count = 10 >= 10 => "high"
	for i := 0; i < 10; i++ {
		if err := eng.RecordOutcome(ctx, StrategyOutcome{Vendor: "HighVendor", SKU: "A", Strategy: "aggressive", DiscountPct: 0.30, RoundsComplete: 2, Outcome: "accepted", TotalBefore: 100, TotalAfter: 70}); err != nil {
			t.Fatalf("RecordOutcome: %v", err)
		}
	}
	rec, err = eng.GetRecommendation(ctx, "HighVendor")
	if err != nil {
		t.Fatalf("GetRecommendation: %v", err)
	}
	if rec.Confidence != "high" {
		t.Errorf("10 deals: expected 'high', got %s", rec.Confidence)
	}
}
