package toolbilling

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })
	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestSetToolPrice(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	tp, err := store.SetToolPrice(ctx, "negotiate_query_price", 0.05)
	if err != nil {
		t.Fatalf("SetToolPrice: %v", err)
	}
	if tp.ToolName != "negotiate_query_price" {
		t.Fatalf("expected tool_name 'negotiate_query_price', got %q", tp.ToolName)
	}
	if tp.PricePerCall != 0.05 {
		t.Fatalf("expected price_per_call 0.05, got %f", tp.PricePerCall)
	}

	// Update existing price
	tp, err = store.SetToolPrice(ctx, "negotiate_query_price", 0.03)
	if err != nil {
		t.Fatalf("SetToolPrice update: %v", err)
	}
	if tp.PricePerCall != 0.03 {
		t.Fatalf("expected updated price_per_call 0.03, got %f", tp.PricePerCall)
	}
}

func TestGetBillingReport(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// Log some usage
	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour).Format(time.RFC3339)
	to := now.Add(1 * time.Hour).Format(time.RFC3339)

	if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
		t.Fatalf("LogUsage: %v", err)
	}
	if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
		t.Fatalf("LogUsage: %v", err)
	}
	if err := store.LogUsage(ctx, "key-1", "negotiate_calculate_savings"); err != nil {
		t.Fatalf("LogUsage: %v", err)
	}
	// Different key — should not appear in report
	if err := store.LogUsage(ctx, "key-2", "negotiate_query_price"); err != nil {
		t.Fatalf("LogUsage: %v", err)
	}

	// Set a custom price for one tool
	_, err := store.SetToolPrice(ctx, "negotiate_query_price", 0.10)
	if err != nil {
		t.Fatalf("SetToolPrice: %v", err)
	}

	report, err := store.GetBillingReport(ctx, "key-1", from, to)
	if err != nil {
		t.Fatalf("GetBillingReport: %v", err)
	}

	if report.KeyID != "key-1" {
		t.Fatalf("expected key_id 'key-1', got %q", report.KeyID)
	}
	if report.TotalCalls != 3 {
		t.Fatalf("expected 3 total calls, got %d", report.TotalCalls)
	}
	if report.PeriodFrom != from || report.PeriodTo != to {
		t.Fatalf("period mismatch: from=%q to=%q", report.PeriodFrom, report.PeriodTo)
	}

	// 2 * 0.10 (custom price) + 1 * 0.01 (default price) = 0.21
	expectedCost := 2*0.10 + 1*0.01
	if math.Abs(report.TotalCost-expectedCost) > 1e-9 {
		t.Fatalf("expected total cost %.10f, got %.10f", expectedCost, report.TotalCost)
	}

	if len(report.PerTool) != 2 {
		t.Fatalf("expected 2 tools in per_tool, got %d", len(report.PerTool))
	}
	if report.PerTool["negotiate_query_price"] != 2 {
		t.Fatalf("expected 2 calls for negotiate_query_price, got %d", report.PerTool["negotiate_query_price"])
	}
	if report.PerTool["negotiate_calculate_savings"] != 1 {
		t.Fatalf("expected 1 call for negotiate_calculate_savings, got %d", report.PerTool["negotiate_calculate_savings"])
	}
}

func TestGetBillingReport_Empty(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour).Format(time.RFC3339)
	to := now.Add(1 * time.Hour).Format(time.RFC3339)

	report, err := store.GetBillingReport(ctx, "nonexistent-key", from, to)
	if err != nil {
		t.Fatalf("GetBillingReport: %v", err)
	}
	if report.TotalCalls != 0 {
		t.Fatalf("expected 0 total calls for nonexistent key, got %d", report.TotalCalls)
	}
	if report.TotalCost != 0 {
		t.Fatalf("expected 0 total cost for nonexistent key, got %f", report.TotalCost)
	}
	if len(report.PerTool) != 0 {
		t.Fatalf("expected empty per_tool, got %d entries", len(report.PerTool))
	}
}

func TestGetUsageTier(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// No usage — should be tier1
	tier, err := store.GetUsageTier(ctx, "key-1")
	if err != nil {
		t.Fatalf("GetUsageTier: %v", err)
	}
	if tier.CurrentTier != "tier1" {
		t.Fatalf("expected tier1 for no usage, got %q", tier.CurrentTier)
	}
	if tier.CallsThisMonth != 0 {
		t.Fatalf("expected 0 calls this month, got %d", tier.CallsThisMonth)
	}
	if tier.TierLimit != 100 {
		t.Fatalf("expected tier limit 100, got %d", tier.TierLimit)
	}

	// Log 50 calls — still tier1
	for i := 0; i < 50; i++ {
		if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
			t.Fatalf("LogUsage: %v", err)
		}
	}
	tier, err = store.GetUsageTier(ctx, "key-1")
	if err != nil {
		t.Fatalf("GetUsageTier: %v", err)
	}
	if tier.CurrentTier != "tier1" {
		t.Fatalf("expected tier1 for 50 calls, got %q", tier.CurrentTier)
	}

	// Log 60 more — total 110, should be tier2
	for i := 0; i < 60; i++ {
		if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
			t.Fatalf("LogUsage: %v", err)
		}
	}
	tier, err = store.GetUsageTier(ctx, "key-1")
	if err != nil {
		t.Fatalf("GetUsageTier: %v", err)
	}
	if tier.CurrentTier != "tier2" {
		t.Fatalf("expected tier2 for 110 calls, got %q", tier.CurrentTier)
	}
	if tier.TierLimit != 1000 {
		t.Fatalf("expected tier limit 1000, got %d", tier.TierLimit)
	}

	// Log 900 more — total 1010, should be tier3
	for i := 0; i < 900; i++ {
		if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
			t.Fatalf("LogUsage: %v", err)
		}
	}
	tier, err = store.GetUsageTier(ctx, "key-1")
	if err != nil {
		t.Fatalf("GetUsageTier: %v", err)
	}
	if tier.CurrentTier != "tier3" {
		t.Fatalf("expected tier3 for 1010 calls, got %q", tier.CurrentTier)
	}
	if tier.TierLimit != 0 {
		t.Fatalf("expected tier limit 0 (unlimited), got %d", tier.TierLimit)
	}
}

func TestLogUsage(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	if err := store.LogUsage(ctx, "key-1", "negotiate_query_price"); err != nil {
		t.Fatalf("LogUsage: %v", err)
	}

	// Verify the log entry via a billing report
	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour).Format(time.RFC3339)
	to := now.Add(1 * time.Hour).Format(time.RFC3339)

	report, err := store.GetBillingReport(ctx, "key-1", from, to)
	if err != nil {
		t.Fatalf("GetBillingReport: %v", err)
	}
	if report.TotalCalls != 1 {
		t.Fatalf("expected 1 call, got %d", report.TotalCalls)
	}
}
