package health

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func setupReputationTest(t *testing.T) (*Engine, *Store) {
	t.Helper()

	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := NewEngine(store, logger)

	return engine, store
}

// ─── GetReputation Tests ───

func TestGetReputation_UnknownVendor_ReturnsZeroValue(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	vr, err := engine.GetReputation(ctx, "UnknownVendor")
	if err != nil {
		t.Fatalf("GetReputation: %v", err)
	}

	if vr.Vendor != "UnknownVendor" {
		t.Errorf("expected vendor 'UnknownVendor', got %q", vr.Vendor)
	}
	if vr.DealCount != 0 {
		t.Errorf("expected deal_count 0, got %d", vr.DealCount)
	}
	if vr.AvgDiscountPct != 0 {
		t.Errorf("expected avg_discount_pct 0, got %f", vr.AvgDiscountPct)
	}
	if vr.MaxDiscountPct != 0 {
		t.Errorf("expected max_discount_pct 0, got %f", vr.MaxDiscountPct)
	}
	if vr.WinRate != 0 {
		t.Errorf("expected win_rate 0, got %f", vr.WinRate)
	}
	if vr.Negotiability != "" {
		t.Errorf("expected negotiability empty, got %q", vr.Negotiability)
	}
}

// ─── UpdateReputation Tests ───

func TestUpdateReputation_SingleDeal(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	if err := engine.UpdateReputation(ctx, "FlexVendor", 25.0, true); err != nil {
		t.Fatalf("UpdateReputation: %v", err)
	}

	vr, err := engine.GetReputation(ctx, "FlexVendor")
	if err != nil {
		t.Fatalf("GetReputation: %v", err)
	}

	if vr.Vendor != "FlexVendor" {
		t.Errorf("expected vendor 'FlexVendor', got %q", vr.Vendor)
	}
	if vr.DealCount != 1 {
		t.Errorf("expected deal_count 1, got %d", vr.DealCount)
	}
	if vr.AvgDiscountPct != 25.0 {
		t.Errorf("expected avg_discount_pct 25, got %f", vr.AvgDiscountPct)
	}
	if vr.MaxDiscountPct != 25.0 {
		t.Errorf("expected max_discount_pct 25, got %f", vr.MaxDiscountPct)
	}
	if vr.WinRate != 1.0 {
		t.Errorf("expected win_rate 1.0, got %f", vr.WinRate)
	}
}

func TestUpdateReputation_MultipleDeals(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	// 3 deals: 10% success, 20% fail, 30% success
	if err := engine.UpdateReputation(ctx, "MultiVendor", 10.0, true); err != nil {
		t.Fatalf("UpdateReputation 1: %v", err)
	}
	if err := engine.UpdateReputation(ctx, "MultiVendor", 20.0, false); err != nil {
		t.Fatalf("UpdateReputation 2: %v", err)
	}
	if err := engine.UpdateReputation(ctx, "MultiVendor", 30.0, true); err != nil {
		t.Fatalf("UpdateReputation 3: %v", err)
	}

	vr, err := engine.GetReputation(ctx, "MultiVendor")
	if err != nil {
		t.Fatalf("GetReputation: %v", err)
	}

	if vr.DealCount != 3 {
		t.Errorf("expected deal_count 3, got %d", vr.DealCount)
	}
	// avg = (10 + 20 + 30) / 3 = 20
	if vr.AvgDiscountPct != 20.0 {
		t.Errorf("expected avg_discount_pct 20, got %f", vr.AvgDiscountPct)
	}
	// max = 30
	if vr.MaxDiscountPct != 30.0 {
		t.Errorf("expected max_discount_pct 30, got %f", vr.MaxDiscountPct)
	}
	// win_rate = 2/3 ≈ 0.666...
	if vr.WinRate != 2.0/3.0 {
		t.Errorf("expected win_rate ~0.666, got %f", vr.WinRate)
	}
}

// ─── RankFlexibility Tests ───

func TestRankFlexibility_CorrectlyOrdered(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	// Vendor A: avg 10% (rigid)
	// Vendor B: avg 30% (flexible)
	// Vendor C: avg 5% (very_rigid)
	// Expected order: B (30%) > A (10%) > C (5%)

	if err := engine.UpdateReputation(ctx, "VendorA", 5.0, true); err != nil {
		t.Fatalf("UpdateReputation A#1: %v", err)
	}
	if err := engine.UpdateReputation(ctx, "VendorA", 15.0, true); err != nil {
		t.Fatalf("UpdateReputation A#2: %v", err)
	}

	if err := engine.UpdateReputation(ctx, "VendorB", 30.0, true); err != nil {
		t.Fatalf("UpdateReputation B: %v", err)
	}

	if err := engine.UpdateReputation(ctx, "VendorC", 5.0, true); err != nil {
		t.Fatalf("UpdateReputation C: %v", err)
	}

	rankings, err := engine.RankFlexibility(ctx, 10)
	if err != nil {
		t.Fatalf("RankFlexibility: %v", err)
	}

	if len(rankings) < 3 {
		t.Fatalf("expected at least 3 vendors, got %d", len(rankings))
	}

	// B should be first (highest avg discount)
	if rankings[0].Vendor != "VendorB" {
		t.Errorf("expected VendorB first, got %s", rankings[0].Vendor)
	}
	// A should be second
	if rankings[1].Vendor != "VendorA" {
		t.Errorf("expected VendorA second, got %s", rankings[1].Vendor)
	}
	// C should be third
	if rankings[2].Vendor != "VendorC" {
		t.Errorf("expected VendorC third, got %s", rankings[2].Vendor)
	}
}

func TestRankFlexibility_Limit(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	if err := engine.UpdateReputation(ctx, "VendorA", 10.0, true); err != nil {
		t.Fatalf("UpdateReputation A: %v", err)
	}
	if err := engine.UpdateReputation(ctx, "VendorB", 20.0, true); err != nil {
		t.Fatalf("UpdateReputation B: %v", err)
	}
	if err := engine.UpdateReputation(ctx, "VendorC", 30.0, true); err != nil {
		t.Fatalf("UpdateReputation C: %v", err)
	}

	rankings, err := engine.RankFlexibility(ctx, 2)
	if err != nil {
		t.Fatalf("RankFlexibility: %v", err)
	}

	if len(rankings) != 2 {
		t.Errorf("expected 2 vendors with limit=2, got %d", len(rankings))
	}
}

// ─── Negotiability Labels Tests ───

func TestNegotiabilityLabels(t *testing.T) {
	tests := []struct {
		avgPct    float64
		expected  string
	}{
		{0, "very_rigid"},
		{4.9, "very_rigid"},
		{5.0, "rigid"},
		{14.9, "rigid"},
		{15.0, "neutral"},
		{24.9, "neutral"},
		{25.0, "flexible"},
		{39.9, "flexible"},
		{40.0, "very_flexible"},
		{99.0, "very_flexible"},
	}

	for _, tc := range tests {
		got := negotiabilityLabel(tc.avgPct)
		if got != tc.expected {
			t.Errorf("negotiabilityLabel(%f) = %q, want %q", tc.avgPct, got, tc.expected)
		}
	}
}

// ─── Integration test ───
// Verify UpdateReputation produces the correct negotiability label for
// a vendor with a specific avg discount.

func TestUpdateReputation_NegotiabilityLabel_Integration(t *testing.T) {
	engine, _ := setupReputationTest(t)
	ctx := context.Background()

	// Avg discount 12% → rigid
	if err := engine.UpdateReputation(ctx, "ToughVendor", 12.0, true); err != nil {
		t.Fatalf("UpdateReputation: %v", err)
	}

	vr, err := engine.GetReputation(ctx, "ToughVendor")
	if err != nil {
		t.Fatalf("GetReputation: %v", err)
	}

	if vr.Negotiability != "rigid" {
		t.Errorf("expected negotiability 'rigid' for 12%% avg discount, got %q", vr.Negotiability)
	}
}
