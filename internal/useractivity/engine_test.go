package useractivity

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) *Engine {
	t.Helper()

	db, err := sql.Open("sqlite", "file:useractivity_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	// Create the tables that the engine reads from
	_, err = pstore.DB().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS negotiation_sessions (
			id TEXT PRIMARY KEY,
			vendor TEXT,
			sku TEXT,
			strategy TEXT,
			status TEXT DEFAULT 'active',
			current_offer REAL DEFAULT 0,
			list_price REAL DEFAULT 0,
			rounds_complete INTEGER DEFAULT 0,
			outcome TEXT DEFAULT '',
			created_at TEXT,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create negotiation_sessions: %v", err)
	}

	_, err = pstore.DB().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS deal_outcomes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor TEXT,
			sku TEXT,
			list_price REAL,
			final_price REAL,
			discount_pct REAL,
			created_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create deal_outcomes: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewEngine(pstore.DB(), logger)
}

func TestActivityWithData(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	// Seed session data
	_, err := eng.db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (id, vendor, strategy, outcome, created_at)
		VALUES ('s1', 'VendorA', 'aggressive', 'won', datetime('now', '-1 days'))
	`)
	if err != nil {
		t.Fatalf("seed session 1: %v", err)
	}
	_, err = eng.db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (id, vendor, strategy, outcome, created_at)
		VALUES ('s2', 'VendorB', 'balanced', 'won', datetime('now', '-2 days'))
	`)
	if err != nil {
		t.Fatalf("seed session 2: %v", err)
	}
	_, err = eng.db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (id, vendor, strategy, outcome, created_at)
		VALUES ('s3', 'VendorA', 'aggressive', 'lost', datetime('now', '-3 days'))
	`)
	if err != nil {
		t.Fatalf("seed session 3: %v", err)
	}

	// Seed deal outcome
	_, err = eng.db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, created_at)
		VALUES ('VendorA', 'SKU1', 100.0, 80.0, 20.0, datetime('now', '-1 days'))
	`)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	report, err := eng.Report(ctx, "user1", "90d")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if report.UserID != "user1" {
		t.Errorf("expected user_id 'user1', got %q", report.UserID)
	}
	if report.Period != "90d" {
		t.Errorf("expected period '90d', got %q", report.Period)
	}
	if report.TotalSessions != 3 {
		t.Errorf("expected 3 total sessions, got %d", report.TotalSessions)
	}
	if report.CompletedNegotiations != 3 {
		t.Errorf("expected 3 completed negotiations, got %d", report.CompletedNegotiations)
	}
	if report.TotalSavings <= 0 {
		t.Errorf("expected positive total savings, got %f", report.TotalSavings)
	}
	if report.ActiveDays <= 0 {
		t.Errorf("expected positive active days, got %d", report.ActiveDays)
	}
	if report.LastActive == "" {
		t.Error("expected non-empty last_active")
	}

	// Check favorite strategies
	if len(report.FavoriteStrategies) == 0 {
		t.Error("expected at least 1 favorite strategy")
	} else {
		found := false
		for _, s := range report.FavoriteStrategies {
			if s.Strategy == "aggressive" && s.Count >= 2 {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'aggressive' strategy with count >= 2, got %+v", report.FavoriteStrategies)
		}
	}

	// Check top vendors
	if len(report.TopVendors) == 0 {
		t.Error("expected at least 1 top vendor")
	}
}

func TestActivityEmptyData(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	report, err := eng.Report(ctx, "", "30d")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if report.TotalSessions != 0 {
		t.Errorf("expected 0 sessions for empty data, got %d", report.TotalSessions)
	}
	if report.CompletedNegotiations != 0 {
		t.Errorf("expected 0 completed negotiations, got %d", report.CompletedNegotiations)
	}
	if report.TotalSavings != 0 {
		t.Errorf("expected 0 savings for empty data, got %f", report.TotalSavings)
	}
	if report.ActiveDays != 0 {
		t.Errorf("expected 0 active days, got %d", report.ActiveDays)
	}
	if len(report.FavoriteStrategies) != 0 {
		t.Errorf("expected 0 strategies, got %d", len(report.FavoriteStrategies))
	}
	if len(report.TopVendors) != 0 {
		t.Errorf("expected 0 vendors, got %d", len(report.TopVendors))
	}
}

func TestActivityFilterByPeriod(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	// Seed a very old session outside 30d range but within 90d range
	_, err := eng.db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (id, vendor, strategy, outcome, created_at)
		VALUES ('s1', 'VendorA', 'aggressive', 'won', datetime('now', '-60 days'))
	`)
	if err != nil {
		t.Fatalf("seed old session: %v", err)
	}

	// Report with 90d period should include it
	report90, err := eng.Report(ctx, "", "90d")
	if err != nil {
		t.Fatalf("Report 90d: %v", err)
	}
	if report90.TotalSessions != 1 {
		t.Errorf("expected 1 session for 90d, got %d", report90.TotalSessions)
	}

	// Report with 30d period should NOT include it
	report30, err := eng.Report(ctx, "", "30d")
	if err != nil {
		t.Fatalf("Report 30d: %v", err)
	}
	if report30.TotalSessions != 0 {
		t.Errorf("expected 0 sessions for 30d, got %d", report30.TotalSessions)
	}
}
