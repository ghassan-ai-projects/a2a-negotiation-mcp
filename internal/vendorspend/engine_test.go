package vendorspend

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

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

func TestVendorAggregation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Seed deals across multiple vendors
	_, err := db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, final_price, discount_pct, created_at)
		VALUES ('VendorA', 1000, 10, '2026-05-01T00:00:00Z'),
		       ('VendorA', 2000, 15, '2026-05-15T00:00:00Z'),
		       ('VendorB', 5000, 20, '2026-05-10T00:00:00Z'),
		       ('VendorC', 3000, 5, '2026-06-01T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	report, err := eng.Report(ctx, "", "1y")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if report.TotalSpend != 11000 {
		t.Fatalf("expected total_spend 11000, got %.2f", report.TotalSpend)
	}
	if report.Vendors != 3 {
		t.Fatalf("expected 3 vendors, got %d", report.Vendors)
	}
	if report.Subscriptions != 4 {
		t.Fatalf("expected 4 subscriptions, got %d", report.Subscriptions)
	}
	if len(report.ByVendor) != 3 {
		t.Fatalf("expected 3 by_vendor entries, got %d", len(report.ByVendor))
	}

	// Verify VendorB has highest spend
	if report.ByVendor[0].Vendor != "VendorB" {
		t.Fatalf("expected VendorB as top spender, got %s", report.ByVendor[0].Vendor)
	}
	if report.ByVendor[0].TotalSpend != 5000 {
		t.Fatalf("expected VendorB spend 5000, got %.2f", report.ByVendor[0].TotalSpend)
	}
}

func TestEmptyData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	report, err := eng.Report(context.Background(), "", "1y")
	if err != nil {
		t.Fatalf("Report empty: %v", err)
	}

	if report.TotalSpend != 0 {
		t.Fatalf("expected 0 total_spend, got %.2f", report.TotalSpend)
	}
	if report.Vendors != 0 {
		t.Fatalf("expected 0 vendors, got %d", report.Vendors)
	}
	if len(report.ByVendor) != 0 {
		t.Fatalf("expected 0 by_vendor entries, got %d", len(report.ByVendor))
	}
}

func TestFilterByVendor(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, final_price, discount_pct, created_at)
		VALUES ('VendorA', 1000, 10, '2026-05-01T00:00:00Z'),
		       ('VendorA', 2000, 15, '2026-05-15T00:00:00Z'),
		       ('VendorB', 5000, 20, '2026-05-10T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	report, err := eng.Report(ctx, "VendorA", "1y")
	if err != nil {
		t.Fatalf("Report VendorA: %v", err)
	}

	if report.TotalSpend != 3000 {
		t.Fatalf("expected total_spend 3000 for VendorA, got %.2f", report.TotalSpend)
	}
	if report.Vendors != 1 {
		t.Fatalf("expected 1 vendor, got %d", report.Vendors)
	}
	if report.Subscriptions != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", report.Subscriptions)
	}
	if len(report.ByVendor) != 1 {
		t.Fatalf("expected 1 by_vendor entry, got %d", len(report.ByVendor))
	}
	if report.ByVendor[0].Vendor != "VendorA" {
		t.Fatalf("expected VendorA entry, got %s", report.ByVendor[0].Vendor)
	}
}
