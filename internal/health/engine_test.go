package health

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func setupTest(t *testing.T) (*Engine, *Store) {
	t.Helper()

	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := NewEngine(store, logger)

	// Seed known health signals for common vendors
	seedSignals(t, engine)

	return engine, store
}

func seedSignals(t *testing.T, engine *Engine) {
	t.Helper()
	ctx := context.Background()

	type seedSignal struct {
		vendor, signalType, detail string
		weight                     int
	}

	signals := []seedSignal{
		// Salesforce: recent layoffs (weight -15), steady growth (+5)
		{"Salesforce", "layoff", "Salesforce laid off 10% of workforce in 2025", -15},
		{"Salesforce", "growth", "Salesforce reported 8% YoY revenue growth", +5},

		// Slack: acquired by Salesforce (0)
		{"Slack", "acquisition", "Slack acquired by Salesforce in 2021", 0},

		// GitHub: strong growth (+15), acquired by Microsoft (0) — net +15
		{"GitHub", "growth", "GitHub reached 100M+ developers", +15},
		{"GitHub", "acquisition", "GitHub acquired by Microsoft in 2018", 0},

		// DigitalOcean: recent layoffs (-15), IPO (+10) — net -5
		{"DigitalOcean", "layoff", "DigitalOcean laid off 11% of staff in 2024", -15},
		{"DigitalOcean", "ipo", "DigitalOcean went public via direct listing", +10},
	}

	for _, s := range signals {
		// Use AddSignal for zero-weight signals (no default weight substitution)
		if s.weight == 0 {
			if err := engine.Store().AddSignal(ctx, s.vendor, s.signalType, "crunchbase", s.detail, 0); err != nil {
				t.Fatalf("seed signal %s/%s: %v", s.vendor, s.signalType, err)
			}
		} else {
			if err := engine.RecordSignal(ctx, s.vendor, s.signalType, "crunchbase", s.detail, s.weight); err != nil {
				t.Fatalf("seed signal %s/%s: %v", s.vendor, s.signalType, err)
			}
		}
	}
	// Recalculate scores for seeded vendors
	for _, vendor := range []string{"Salesforce", "Slack", "GitHub", "DigitalOcean"} {
		if _, err := engine.CalculateScore(ctx, vendor); err != nil {
			t.Fatalf("CalculateScore(%s): %v", vendor, err)
		}
	}
}

// ─── Score Tests ───

func TestCalculateScore_Salesforce(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	vh, err := engine.CalculateScore(ctx, "Salesforce")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// Baseline 50 + layoff(-15) + growth(+5) = 40
	if vh.Score != 40 {
		t.Errorf("expected score 40, got %d", vh.Score)
	}
	if vh.Category != "stable" {
		t.Errorf("expected category 'stable', got %s", vh.Category)
	}
	if len(vh.Signals) != 2 {
		t.Errorf("expected 2 signals, got %d", len(vh.Signals))
	}
}

func TestCalculateScore_GitHub(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	vh, err := engine.CalculateScore(ctx, "GitHub")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// Baseline 50 + growth(+15) + acquisition(0) = 65
	if vh.Score != 65 {
		t.Errorf("expected score 65, got %d", vh.Score)
	}
	if vh.Category != "growing" {
		t.Errorf("expected category 'growing', got %s", vh.Category)
	}
}

func TestCalculateScore_DigitalOcean(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	vh, err := engine.CalculateScore(ctx, "DigitalOcean")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// Baseline 50 + layoff(-15) + ipo(+10) = 45
	if vh.Score != 45 {
		t.Errorf("expected score 45, got %d", vh.Score)
	}
	if vh.Category != "stable" {
		t.Errorf("expected category 'stable', got %s", vh.Category)
	}
}

func TestCalculateScore_UnknownVendor(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	vh, err := engine.CalculateScore(ctx, "NewVendor")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// No signals = baseline 50
	if vh.Score != 50 {
		t.Errorf("expected score 50, got %d", vh.Score)
	}
	if vh.Category != "stable" {
		t.Errorf("expected category 'stable', got %s", vh.Category)
	}
}

func TestCalculateScore_Clamping(t *testing.T) {
	engine, store := setupTest(t)
	ctx := context.Background()

	// Add an extreme negative signals that should clamp to minimum
	_ = store.AddSignal(ctx, "CrisisVendor", "layoff", "manual", "Massive layoffs", -100)
	_ = store.AddSignal(ctx, "CrisisVendor", "lawsuit", "manual", "Major lawsuit", -100)
	_ = store.AddSignal(ctx, "CrisisVendor", "layoff", "manual", "More layoffs", -100)

	vh, err := engine.CalculateScore(ctx, "CrisisVendor")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	if vh.Score != 1 {
		t.Errorf("expected score clamped to 1 (50-300=-250 clamped), got %d", vh.Score)
	}
	if vh.Category != "struggling" {
		t.Errorf("expected category 'struggling', got %s", vh.Category)
	}
}

func TestCalculateScore_ClampingMax(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	_ = engine.RecordSignal(ctx, "BoomVendor", "ipo", "news", "Huge IPO", 0)
	_ = engine.RecordSignal(ctx, "BoomVendor", "growth", "news", "Massive growth", 0)
	_ = engine.RecordSignal(ctx, "BoomVendor", "funding", "news", "New funding round", 0)
	// 50 + 20 + 15 + 10 = 95

	vh, err := engine.CalculateScore(ctx, "BoomVendor")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	if vh.Score != 95 {
		t.Errorf("expected score 95, got %d", vh.Score)
	}
	if vh.Category != "growing" {
		t.Errorf("expected category 'growing', got %s", vh.Category)
	}
}

// ─── Leverage Tests ───

func TestGetLeverage_Medium_Salesforce(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// Already calculated in seed

	leverage, err := engine.GetLeverage(ctx, "Salesforce")
	if err != nil {
		t.Fatalf("GetLeverage: %v", err)
	}

	if leverage.Vendor != "Salesforce" {
		t.Errorf("expected vendor Salesforce, got %s", leverage.Vendor)
	}
	// Score 40 = medium leverage (30-60)
	if leverage.Leverage != "medium" {
		t.Errorf("expected leverage 'medium' for score 40, got %s", leverage.Leverage)
	}
}

func TestGetLeverage_Struggling(t *testing.T) {
	engine, store := setupTest(t)
	ctx := context.Background()

	// Create a vendor with only layoff signals to get struggling
	_ = store.AddSignal(ctx, "TroubledCorp", "layoff", "news", "Laid off 30%", -20)
	_ = store.AddSignal(ctx, "TroubledCorp", "lawsuit", "news", "Class action suit", -20)

	_, err := engine.CalculateScore(ctx, "TroubledCorp")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	leverage, err := engine.GetLeverage(ctx, "TroubledCorp")
	if err != nil {
		t.Fatalf("GetLeverage: %v", err)
	}

	// 50 + (-20) + (-20) = 10 → struggling → high leverage
	if leverage.Leverage != "high" {
		t.Errorf("expected leverage 'high' for score 10, got %s", leverage.Leverage)
	}
	if leverage.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestGetLeverage_Growing(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// Already calculated in seed — GitHub score 65
	leverage, err := engine.GetLeverage(ctx, "GitHub")
	if err != nil {
		t.Fatalf("GetLeverage: %v", err)
	}

	// 65 → growing → low leverage
	if leverage.Leverage != "low" {
		t.Errorf("expected leverage 'low' for score 65, got %s", leverage.Leverage)
	}
}

func TestGetLeverage_AutoCalculate(t *testing.T) {
	engine, store := setupTest(t)
	ctx := context.Background()

	// Add a signal without calculating first — GetLeverage should auto-calculate
	_ = store.AddSignal(ctx, "AutoCalc", "layoff", "news", "Some layoffs", -15)

	leverage, err := engine.GetLeverage(ctx, "AutoCalc")
	if err != nil {
		t.Fatalf("GetLeverage: %v", err)
	}

	if leverage.Vendor != "AutoCalc" {
		t.Errorf("expected vendor AutoCalc, got %s", leverage.Vendor)
	}
	// 50 + (-15) = 35 → stable → medium leverage
	if leverage.Leverage != "medium" {
		t.Errorf("expected leverage 'medium', got %s", leverage.Leverage)
	}
}

// ─── RecordSignal Tests ───

func TestRecordSignal_WithCustomWeight(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	if err := engine.RecordSignal(ctx, "CustomVendor", "lawsuit", "manual", "Patent lawsuit", -20); err != nil {
		t.Fatalf("RecordSignal: %v", err)
	}

	vh, err := engine.CalculateScore(ctx, "CustomVendor")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// 50 + (-20) = 30
	if vh.Score != 30 {
		t.Errorf("expected score 30, got %d", vh.Score)
	}
}

func TestRecordSignal_WithDefaultWeight(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// weight = 0 should use default weight for 'ipo' = +20
	if err := engine.RecordSignal(ctx, "DefaultVendor", "ipo", "news", "IPO announced", 0); err != nil {
		t.Fatalf("RecordSignal: %v", err)
	}

	vh, err := engine.CalculateScore(ctx, "DefaultVendor")
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	// 50 + 20 = 70
	if vh.Score != 70 {
		t.Errorf("expected score 70 for ipo signal, got %d", vh.Score)
	}
}

// ─── ListAll Tests ───

func TestListAll_SortedByScore(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	// All seeded vendors already have scores from seedSignals
	vendors, err := engine.Store().ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	// Should have 4 vendors: Salesforce(40), Slack(50), DigitalOcean(45), GitHub(65)
	if len(vendors) != 4 {
		t.Errorf("expected 4 vendors, got %d", len(vendors))
	}

	// Verify ascending order
	for i := 1; i < len(vendors); i++ {
		if vendors[i].Score < vendors[i-1].Score {
			t.Errorf("vendors not sorted ascending: %s(%d) before %s(%d)",
				vendors[i-1].Vendor, vendors[i-1].Score,
				vendors[i].Vendor, vendors[i].Score)
		}
	}

	// Verify signals are loaded
	for _, v := range vendors {
		if len(v.Signals) == 0 {
			t.Errorf("expected signals to be loaded for %s", v.Vendor)
		}
	}
}

func TestListAll_Empty(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	vendors, err := store.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	if len(vendors) != 0 {
		t.Errorf("expected empty list, got %d vendors", len(vendors))
	}
}

// ─── Signal Type Weights Tests ───

func TestSignalTypeWeights_Defaults(t *testing.T) {
	tests := []struct {
		signalType string
		expected   int
	}{
		{"layoff", -15},
		{"lawsuit", -20},
		{"funding", +10},
		{"growth", +15},
		{"ipo", +20},
		{"acquisition", +5},
	}

	for _, tc := range tests {
		t.Run(tc.signalType, func(t *testing.T) {
			got, ok := SignalTypeWeights[tc.signalType]
			if !ok {
				t.Fatalf("missing default weight for %s", tc.signalType)
			}
			if got != tc.expected {
				t.Errorf("expected %d for %s, got %d", tc.expected, tc.signalType, got)
			}
		})
	}
}
