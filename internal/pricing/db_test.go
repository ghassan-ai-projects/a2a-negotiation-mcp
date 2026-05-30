package pricing

import (
	"context"
	"testing"
)

func seedTestData(t *testing.T, store *Store) {
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
		_, err := store.db.ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
			v.name, v.category)
		if err != nil {
			t.Fatalf("insert vendor %s: %v", v.name, err)
		}
		var vid int64
		err = store.db.QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", v.name).Scan(&vid)
		if err != nil {
			t.Fatalf("get vendor id %s: %v", v.name, err)
		}
		_, err = store.db.ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
		`, vid, v.sku, v.desc, v.listPrice, v.minObs, v.maxObs, v.typicalPct, v.unit)
		if err != nil {
			t.Fatalf("insert pricing %s/%s: %v", v.name, v.sku, err)
		}
	}
}

func TestNewInMemoryStore(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()

	// Verify vendors table exists
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count)
	if err != nil {
		t.Fatalf("query vendors count: %v", err)
	}
}

func TestGetVendorID(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()
	seedTestData(t, store)

	ctx := context.Background()
	id, err := store.GetVendorID(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetVendorID(Slack): %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive vendor ID, got %d", id)
	}

	_, err = store.GetVendorID(ctx, "NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent vendor")
	}
}

func TestGetPricingByVendorSKU(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()
	seedTestData(t, store)

	ctx := context.Background()
	result, err := store.GetPricingByVendorSKU(ctx, "Slack", "Pro")
	if err != nil {
		t.Fatalf("GetPricingByVendorSKU(Slack, Pro): %v", err)
	}
	if result.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", result.Vendor)
	}
	if result.ListPrice != 8.75 {
		t.Errorf("expected list price 8.75, got %f", result.ListPrice)
	}
	if result.SuggestedMin <= 0 || result.SuggestedMax <= 0 {
		t.Errorf("expected positive suggested range, got min=%f max=%f", result.SuggestedMin, result.SuggestedMax)
	}
	if result.SuggestedMin >= result.SuggestedMax {
		t.Errorf("expected SuggestedMin < SuggestedMax, got min=%f max=%f", result.SuggestedMin, result.SuggestedMax)
	}

	// Test without SKU
	result2, err := store.GetPricingByVendorSKU(ctx, "Slack", "")
	if err != nil {
		t.Fatalf("GetPricingByVendorSKU(Slack, ''): %v", err)
	}
	if result2.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", result2.Vendor)
	}

	// Test unknown vendor
	_, err = store.GetPricingByVendorSKU(ctx, "UnknownCorp", "Pro")
	if err == nil {
		t.Fatal("expected error for unknown vendor")
	}
}

func TestGetMarketRange(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()
	seedTestData(t, store)

	ctx := context.Background()
	mr, err := store.GetMarketRange(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetMarketRange(Slack): %v", err)
	}
	if mr.Count < 2 {
		t.Errorf("expected at least 2 SKUs for Slack, got %d", mr.Count)
	}
	if mr.Min <= 0 || mr.Max <= 0 {
		t.Errorf("expected positive range, got min=%f max=%f", mr.Min, mr.Max)
	}
	if mr.Min > mr.Max {
		t.Errorf("expected min <= max, got min=%f max=%f", mr.Min, mr.Max)
	}
}

func TestListVendorsWithPricing(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()
	seedTestData(t, store)

	ctx := context.Background()
	vendors, err := store.ListVendorsWithPricing(ctx)
	if err != nil {
		t.Fatalf("ListVendorsWithPricing: %v", err)
	}
	if len(vendors) < 3 {
		t.Errorf("expected at least 3 vendors, got %d: %v", len(vendors), vendors)
	}
}

func TestEmptyDB(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore() failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = store.GetVendorID(ctx, "Anything")
	if err == nil {
		t.Fatal("expected error for empty DB")
	}

	vendors, err := store.ListVendorsWithPricing(ctx)
	if err != nil {
		t.Fatalf("ListVendorsWithPricing on empty DB: %v", err)
	}
	if len(vendors) != 0 {
		t.Errorf("expected 0 vendors, got %d", len(vendors))
	}
}
