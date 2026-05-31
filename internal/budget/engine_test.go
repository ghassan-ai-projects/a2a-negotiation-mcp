package budget

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite DB with deal_outcomes + user_streaks tables
// for testing budget dashboard queries against deal data.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Create spend_budgets (managed by budget store)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS spend_budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL UNIQUE,
		budget_amount REAL NOT NULL DEFAULT 0,
		period_month TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)
	`)
	if err != nil {
		t.Fatalf("create spend_budgets: %v", err)
	}

	// Create deal_outcomes (managed by history store - needed for dashboard tests)
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

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestSetGetDeleteBudget(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	ctx := context.Background()

	// Set budget
	err = store.SetBudget(ctx, "Acme Corp", 50000, "2026-05")
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}

	// Get budget
	b, err := store.GetBudget(ctx, "Acme Corp")
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b == nil {
		t.Fatal("expected budget, got nil")
	}
	if b.Vendor != "Acme Corp" || b.BudgetAmount != 50000 || b.PeriodMonth != "2026-05" {
		t.Fatalf("unexpected budget: %+v", b)
	}

	// Get non-existent budget
	nb, err := store.GetBudget(ctx, "NoCorp")
	if err != nil {
		t.Fatalf("GetBudget nonexistent: %v", err)
	}
	if nb != nil {
		t.Fatal("expected nil for nonexistent budget")
	}

	// List budgets
	budgets, err := store.ListBudgets(ctx)
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgets))
	}

	// Delete budget
	err = store.DeleteBudget(ctx, "Acme Corp")
	if err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}

	// Verify deleted
	budgets, err = store.ListBudgets(ctx)
	if err != nil {
		t.Fatalf("ListBudgets after delete: %v", err)
	}
	if len(budgets) != 0 {
		t.Fatalf("expected 0 budgets, got %d", len(budgets))
	}

	// Delete non-existent should error
	err = store.DeleteBudget(ctx, "NoCorp")
	if err == nil {
		t.Fatal("expected error deleting nonexistent budget")
	}
}

func TestDashboardWithData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	ctx := context.Background()

	// Seed budgets
	if err := store.SetBudget(ctx, "VendorA", 10000, "2026-05"); err != nil {
		t.Fatalf("SetBudget VendorA: %v", err)
	}
	if err := store.SetBudget(ctx, "VendorB", 20000, "2026-06"); err != nil {
		t.Fatalf("SetBudget VendorB: %v", err)
	}

	// Seed deal_outcomes (actual spend)
	_, err = db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, final_price, created_at)
		VALUES ('VendorA', 8000, '2026-05-15T00:00:00Z'),
		       ('VendorA', 3000, '2026-05-20T00:00:00Z'),
		       ('VendorB', 22000, '2026-06-10T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(store, db, logger)

	dash, err := eng.Dashboard(ctx, "monthly")
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	if dash.TotalBudget != 30000 {
		t.Fatalf("expected total_budget 30000, got %.2f", dash.TotalBudget)
	}
	if dash.TotalActual != 33000 {
		t.Fatalf("expected total_actual 33000, got %.2f", dash.TotalActual)
	}
	if len(dash.ByVendor) != 2 {
		t.Fatalf("expected 2 vendors, got %d", len(dash.ByVendor))
	}
	if len(dash.MonthlyTrend) == 0 {
		t.Fatal("expected monthly trend data")
	}
}

func TestOverspendWarning(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	ctx := context.Background()

	// Set budget that will be exceeded
	if err := store.SetBudget(ctx, "TightCorp", 1000, "2026-05"); err != nil {
		t.Fatalf("SetBudget TightCorp: %v", err)
	}

	// Seed deal with actual > budget * 1.1 (overspend > 10%)
	_, err = db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, final_price, created_at)
		VALUES ('TightCorp', 1500, '2026-05-10T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(store, db, logger)

	dash, err := eng.Dashboard(ctx, "monthly")
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	if len(dash.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(dash.Warnings))
	}
	if dash.Warnings[0].Vendor != "TightCorp" {
		t.Fatalf("expected warning for TightCorp, got %s", dash.Warnings[0].Vendor)
	}
}
