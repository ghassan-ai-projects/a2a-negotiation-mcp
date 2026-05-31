package costallocation

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupCostAllocationTest(t *testing.T) (*Engine, *Store, *sql.DB) {
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

	// Create deal_outcomes table (normally created by history store)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS deal_outcomes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor TEXT NOT NULL,
			sku TEXT NOT NULL DEFAULT '',
			list_price REAL NOT NULL DEFAULT 0,
			final_price REAL NOT NULL DEFAULT 0,
			discount_pct REAL NOT NULL DEFAULT 0,
			seats INTEGER NOT NULL DEFAULT 0,
			term_months INTEGER NOT NULL DEFAULT 12,
			strategy TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create deal_outcomes: %v", err)
	}

	_, err = db.Exec(`INSERT INTO deal_outcomes (vendor, final_price, seats, created_at) VALUES ('Slack', 7.00, 100, datetime('now', '-30 days'))`)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}
	_, err = db.Exec(`INSERT INTO deal_outcomes (vendor, final_price, seats, created_at) VALUES ('Slack', 6.50, 50, datetime('now', '-60 days'))`)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	eng := NewEngine(store, db)
	return eng, store, db
}

func TestSetAllocation(t *testing.T) {
	eng, _, _ := setupCostAllocationTest(t)
	ctx := context.Background()

	alloc, err := eng.SetAllocation(ctx, "Slack", "Engineering", 60.0)
	if err != nil {
		t.Fatalf("SetAllocation: %v", err)
	}

	if alloc.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", alloc.Vendor)
	}
	if alloc.Department != "Engineering" {
		t.Errorf("expected department Engineering, got %s", alloc.Department)
	}
	if alloc.AllocationPct != 60.0 {
		t.Errorf("expected allocation_pct 60, got %f", alloc.AllocationPct)
	}
	if alloc.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
	if alloc.UpdatedAt == "" {
		t.Errorf("expected non-empty updated_at")
	}
}

func TestGetAllocation(t *testing.T) {
	eng, store, _ := setupCostAllocationTest(t)
	ctx := context.Background()

	// Set first
	eng.SetAllocation(ctx, "Slack", "Engineering", 60.0)

	// Get it via store
	alloc, err := store.Get(ctx, "Slack", "Engineering")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if alloc == nil {
		t.Fatal("expected non-nil allocation")
	}
	if alloc.AllocationPct != 60.0 {
		t.Errorf("expected 60%%, got %f", alloc.AllocationPct)
	}

	// Update
	eng.SetAllocation(ctx, "Slack", "Engineering", 75.0)
	alloc, err = store.Get(ctx, "Slack", "Engineering")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if alloc.AllocationPct != 75.0 {
		t.Errorf("expected 75%% after update, got %f", alloc.AllocationPct)
	}
}

func TestReport_WithData(t *testing.T) {
	eng, _, _ := setupCostAllocationTest(t)
	ctx := context.Background()

	// Set allocations
	eng.SetAllocation(ctx, "Slack", "Engineering", 60.0)
	eng.SetAllocation(ctx, "Slack", "Marketing", 40.0)

	report, err := eng.Report(ctx, "90d")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	// Slack deals: 7.00*100 + 6.50*50 = 700 + 325 = 1025
	// Engineering: 1025 * 0.6 = 615
	// Marketing: 1025 * 0.4 = 410
	if report.TotalSpend != 1025 {
		t.Errorf("expected total_spend 1025, got %f", report.TotalSpend)
	}
	if len(report.ByDepartment) != 2 {
		t.Errorf("expected 2 departments, got %d", len(report.ByDepartment))
	}
	if len(report.ByVendorDept) != 2 {
		t.Errorf("expected 2 vendor-dept entries, got %d", len(report.ByVendorDept))
	}
}

func TestReport_Empty(t *testing.T) {
	eng, _, _ := setupCostAllocationTest(t)
	ctx := context.Background()

	// No allocations set
	report, err := eng.Report(ctx, "90d")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if report.TotalSpend != 0 {
		t.Errorf("expected 0 total_spend with no allocations, got %f", report.TotalSpend)
	}
	if len(report.ByDepartment) != 0 {
		t.Errorf("expected 0 departments, got %d", len(report.ByDepartment))
	}
}
