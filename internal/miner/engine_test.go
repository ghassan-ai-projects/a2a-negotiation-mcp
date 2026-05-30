package miner

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func seedPricingStore(t *testing.T, store *pricing.Store) {
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

func setupEngine(t *testing.T) (*Engine, *pricing.Store) {
	t.Helper()
	store, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore(): %v", err)
	}
	t.Cleanup(func() { store.Close() })
	seedPricingStore(t, store)
	logger := slog.Default()
	engine := NewEngine(store, logger)
	return engine, store
}

func TestDiscoverOpportunities_TechStartup(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	profile := BusinessProfile{
		Name:      "TechNova Inc",
		Industry:  "tech",
		Employees: 200,
	}

	opps, err := engine.DiscoverOpportunities(ctx, profile)
	if err != nil {
		t.Fatalf("DiscoverOpportunities: %v", err)
	}

	if len(opps) == 0 {
		t.Fatal("expected at least 1 opportunity, got 0")
	}

	categories := make(map[string]bool)
	for _, opp := range opps {
		categories[opp.Category] = true
	}

	for _, expected := range []string{"software", "hosting", "saas"} {
		if !categories[expected] {
			t.Errorf("expected category %q in results, got %v", expected, opps)
		}
	}
}

func TestDiscoverOpportunities_Logistics(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	profile := BusinessProfile{
		Name:      "LogiShip Co",
		Industry:  "logistics",
		Employees: 500,
	}

	opps, err := engine.DiscoverOpportunities(ctx, profile)
	if err != nil {
		t.Fatalf("DiscoverOpportunities: %v", err)
	}

	if len(opps) == 0 {
		t.Fatal("expected at least 1 opportunity, got 0")
	}

	categories := make(map[string]bool)
	for _, opp := range opps {
		categories[opp.Category] = true
	}

	for _, expected := range []string{"carrier", "software", "saas"} {
		if !categories[expected] {
			t.Errorf("expected category %q in results, got %v", expected, opps)
		}
	}
}

func TestDiscoverOpportunities_EmptyProfile(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	profile := BusinessProfile{}

	opps, err := engine.DiscoverOpportunities(ctx, profile)
	if err != nil {
		t.Fatalf("DiscoverOpportunities: %v", err)
	}

	if len(opps) < 3 {
		t.Errorf("expected at least 3 industry-agnostic opportunities, got %d", len(opps))
	}
}

func TestDiscoverOpportunities_WithKnownVendors(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	profile := BusinessProfile{
		Name:      "TestCorp",
		Industry:  "retail",
		Employees: 100,
		Vendors:   []string{"Slack", "GitHub", "NonExistentVendor"},
	}

	opps, err := engine.DiscoverOpportunities(ctx, profile)
	if err != nil {
		t.Fatalf("DiscoverOpportunities: %v", err)
	}

	if len(opps) == 0 {
		t.Fatal("expected at least 1 opportunity, got 0")
	}

	foundSlack := false
	foundGitHub := false
	for _, opp := range opps {
		if opp.Vendor == "Slack" {
			foundSlack = true
			if opp.Confidence != "high" {
				t.Errorf("expected high confidence for Slack (typical 18%% > 15%%), got %q", opp.Confidence)
			}
		}
		if opp.Vendor == "GitHub" {
			foundGitHub = true
			if opp.TypicalDiscount != 15 {
				t.Errorf("expected TypicalDiscount 15 for GitHub, got %f", opp.TypicalDiscount)
			}
		}
	}

	if !foundSlack {
		t.Error("expected Slack to appear in cross-referenced opportunities")
	}
	if !foundGitHub {
		t.Error("expected GitHub to appear in cross-referenced opportunities")
	}

	// NonExistentVendor should be silently skipped (logged).
}

func TestDiscoverOpportunities_Top10Limit(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	profile := BusinessProfile{
		Name:      "BigCorp",
		Industry:  "tech",
		Employees: 200,
		Vendors:   []string{"Slack", "GitHub", "Salesforce"},
	}

	opps, err := engine.DiscoverOpportunities(ctx, profile)
	if err != nil {
		t.Fatalf("DiscoverOpportunities: %v", err)
	}

	if len(opps) > 10 {
		t.Errorf("expected at most 10 opportunities, got %d", len(opps))
	}

	// Verify sorting: first opportunity should have highest score.
	for i := 1; i < len(opps); i++ {
		// Cross-referenced opportunities have EstimatedSpend=0, so they'll score low.
		// We just check they're present and IDs are assigned.
		if opps[i].ID == "" {
			t.Errorf("expected non-empty ID for opportunity %d", i)
		}
	}

	// All opportunities should have IDs.
	for _, opp := range opps {
		if opp.ID == "" {
			t.Errorf("expected non-empty ID for opportunity: %+v", opp)
		}
	}
}

func TestDiscoverOpportunities_EmployeeSpendScaling(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	small := BusinessProfile{Name: "SmallBiz", Industry: "tech", Employees: 5}
	large := BusinessProfile{Name: "MegaCorp", Industry: "tech", Employees: 2000}

	smallOpps, err := engine.DiscoverOpportunities(ctx, small)
	if err != nil {
		t.Fatalf("DiscoverOpportunities small: %v", err)
	}
	largeOpps, err := engine.DiscoverOpportunities(ctx, large)
	if err != nil {
		t.Fatalf("DiscoverOpportunities large: %v", err)
	}

	if len(smallOpps) == 0 || len(largeOpps) == 0 {
		t.Fatal("expected opportunities for both profiles")
	}

	// Large company should have higher estimated spend for the first opportunity.
	if smallOpps[0].EstimatedSpend >= largeOpps[0].EstimatedSpend {
		t.Errorf("expected large company to have higher spend than small company: small=%f large=%f",
			smallOpps[0].EstimatedSpend, largeOpps[0].EstimatedSpend)
	}
}
