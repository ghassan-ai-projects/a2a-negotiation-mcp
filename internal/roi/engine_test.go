package roi

import (
	"context"
	"database/sql"
	"testing"
)

func setupROIStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestCalculate_Basic(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	calc, err := eng.Calculate(ctx, 100000, 85000, 15000, 5000)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// AnnualSavings = 100000 - 85000 = 15000
	if calc.AnnualSavings != 15000 {
		t.Errorf("expected annual savings 15000, got %f", calc.AnnualSavings)
	}

	// NetAnnualSavings = 15000 - 5000 = 10000
	// TotalInvestment = 15000 + 5000 = 20000
	// ROIPct = (10000 - 15000/3) / 20000 * 100 = (10000-5000)/20000*100 = 25%
	if calc.ROIPct != 25.0 {
		t.Errorf("expected ROI pct 25, got %f", calc.ROIPct)
	}

	// PaybackMonths = 20000 / (15000/12) = 20000/1250 = 16
	if calc.PaybackMonths != 16.0 {
		t.Errorf("expected payback 16 months, got %f", calc.PaybackMonths)
	}

	// Savings1Y = 10000 - 15000 = -5000
	if calc.Savings1Y != -5000 {
		t.Errorf("expected savings 1y -5000, got %f", calc.Savings1Y)
	}

	// Savings3Y = 10000*3 - 15000 = 15000
	if calc.Savings3Y != 15000 {
		t.Errorf("expected savings 3y 15000, got %f", calc.Savings3Y)
	}

	// Savings5Y = 10000*5 - 15000 = 35000
	if calc.Savings5Y != 35000 {
		t.Errorf("expected savings 5y 35000, got %f", calc.Savings5Y)
	}

	// NPV = -15000 + 10000/1.08 + 10000/1.08^2 + ... + 10000/1.08^5
	// Should be positive
	if calc.NPV <= 0 {
		t.Errorf("expected positive NPV, got %f", calc.NPV)
	}
}

func TestCalculate_ZeroInvestment(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	// No implementation costs, no overhead
	calc, err := eng.Calculate(ctx, 100000, 80000, 0, 0)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// AnnualSavings = 20000
	// ROIPct = 0 (no investment)
	if calc.ROIPct != 0 {
		t.Errorf("expected ROI pct 0 for zero investment, got %f", calc.ROIPct)
	}

	// PaybackMonths = 0 (no investment)
	if calc.PaybackMonths != 0 {
		t.Errorf("expected payback 0 for zero investment, got %f", calc.PaybackMonths)
	}
}

func TestCalculate_InvalidInput(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	tests := []struct {
		name                                string
		current, negotiated, impl, overhead float64
	}{
		{"zero current", 0, 100, 0, 0},
		{"zero negotiated", 100, 0, 0, 0},
		{"negative current", -100, 50, 0, 0},
		{"price exceeds current", 100, 200, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := eng.Calculate(ctx, tc.current, tc.negotiated, tc.impl, tc.overhead)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestSaveAndGetByID(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	calc, err := eng.Calculate(ctx, 50000, 42000, 8000, 2000)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	saved, err := eng.Save(ctx, calc, "user-1")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Retrieve by ID
	got, err := store.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Vendor != "" {
		t.Errorf("expected empty vendor, got %s", got.Vendor)
	}
	if got.AnnualSavings != 8000 {
		t.Errorf("expected annual savings 8000, got %f", got.AnnualSavings)
	}
}

func TestListByUser(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	// Save a calculation with vendor set
	calc, _ := eng.Calculate(ctx, 100000, 90000, 0, 0)
	calc.Vendor = "VendorA"
	eng.Save(ctx, calc, "user-1")

	calc2, _ := eng.Calculate(ctx, 200000, 160000, 20000, 5000)
	calc2.Vendor = "VendorB"
	eng.Save(ctx, calc2, "user-1")

	calc3, _ := eng.Calculate(ctx, 50000, 45000, 5000, 1000)
	calc3.Vendor = "VendorC"
	eng.Save(ctx, calc3, "user-2")

	// user-1 should have 2
	list, err := eng.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 for user-1, got %d", len(list))
	}

	// user-2 should have 1
	list2, err := eng.ListByUser(ctx, "user-2")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("expected 1 for user-2, got %d", len(list2))
	}

	// unknown user should have 0
	list3, err := eng.ListByUser(ctx, "no-such-user")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list3) != 0 {
		t.Errorf("expected 0 for unknown user, got %d", len(list3))
	}
}

func TestGetByID_NotFound(t *testing.T) {
	store := setupROIStore(t)
	ctx := context.Background()

	got, err := store.GetByID(ctx, 9999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestCalculate_HighSavings(t *testing.T) {
	store := setupROIStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	// Very favorable deal: high savings, low costs
	calc, err := eng.Calculate(ctx, 1000000, 500000, 50000, 10000)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// AnnualSavings = 500000
	if calc.AnnualSavings != 500000 {
		t.Errorf("expected 500000, got %f", calc.AnnualSavings)
	}

	// NetAnnualSavings = 500000 - 10000 = 490000
	// ROIPct = (490000 - 50000/3) / 60000 * 100 = (490000 - 16666.67) / 60000 * 100 = 788.89
	if calc.ROIPct < 700 {
		t.Errorf("expected high ROI pct, got %f", calc.ROIPct)
	}

	// PaybackMonths should be very short
	if calc.PaybackMonths < 1 || calc.PaybackMonths > 3 {
		t.Errorf("expected payback ~1.2 months, got %f", calc.PaybackMonths)
	}
}
