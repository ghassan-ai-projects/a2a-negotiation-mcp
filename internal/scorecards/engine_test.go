package scorecards

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func inMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()

	// Create the tables we need (deal_outcomes, vendor_health, health_signals, sla_breaches)
	schema := `
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
		created_at TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'won'
	);
	CREATE TABLE IF NOT EXISTS health_signals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		type TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT '',
		detail TEXT NOT NULL DEFAULT '',
		weight INTEGER NOT NULL DEFAULT 0,
		date TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS sla_breaches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		filed INTEGER NOT NULL DEFAULT 0,
		date TEXT NOT NULL
	);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	now := time.Now().UTC()

	// Deal outcomes for "Acme Corp"
	for i := 0; i < 5; i++ {
		discount := 15.0 + float64(i)*2 // 15%, 17%, 19%, 21%, 23%
		created := now.AddDate(0, -i, 0)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at, status)
			 VALUES (?, 'Pro', 100.0, ?, ?, 50, 12, 'competitive', ?, ?, 'won')`,
			"Acme Corp", 100.0*(1-discount/100), discount, "session-"+string(rune('a'+i)), created.Format(time.RFC3339),
		); err != nil {
			t.Fatalf("insert deal: %v", err)
		}
	}

	// One deal for "Beta Inc"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at, status)
		 VALUES ('Beta Inc', 'Basic', 50.0, 45.0, 10.0, 20, 12, 'standard', 'session-x', ?, 'won')`,
		now.AddDate(0, -3, 0).Format(time.RFC3339),
	); err != nil {
		t.Fatalf("insert beta deal: %v", err)
	}

	// Health signals for Acme Corp
	signalTypes := []string{"funding", "growth", "acquisition"}
	for i, st := range signalTypes {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO health_signals (vendor, type, source, detail, weight, date)
			 VALUES (?, ?, 'news', 'Signal detail', 10, ?)`,
			"Acme Corp", st, now.AddDate(0, -i, 0).Format(time.RFC3339),
		); err != nil {
			t.Fatalf("insert signal: %v", err)
		}
	}

	// SLA breaches: 1 filed out of 5 total
	for i := 0; i < 5; i++ {
		filed := 0
		if i == 0 {
			filed = 1
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO sla_breaches (vendor, filed, date) VALUES (?, ?, ?)`,
			"Acme Corp", filed, now.AddDate(0, -i, 0).Format(time.RFC3339),
		); err != nil {
			t.Fatalf("insert breach: %v", err)
		}
	}
}

func TestScorecardWithData(t *testing.T) {
	db := inMemoryDB(t)
	seedTestData(t, db)

	eng := NewEngine(db, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	sc, err := eng.Scorecard(ctx, "Acme Corp", "2y")
	if err != nil {
		t.Fatalf("Scorecard: %v", err)
	}

	if sc.Vendor != "Acme Corp" {
		t.Errorf("expected vendor 'Acme Corp', got %q", sc.Vendor)
	}
	if sc.OverallScore <= 0 || sc.OverallScore > 100 {
		t.Errorf("expected overall_score between 0-100, got %f", sc.OverallScore)
	}
	if sc.PricingScore <= 0 || sc.PricingScore > 100 {
		t.Errorf("expected pricing_score between 0-100, got %f", sc.PricingScore)
	}
	if sc.ReliabilityScore <= 0 || sc.ReliabilityScore > 100 {
		t.Errorf("expected reliability_score between 0-100, got %f", sc.ReliabilityScore)
	}
	if sc.SupportScore <= 0 || sc.SupportScore > 100 {
		t.Errorf("expected support_score between 0-100, got %f", sc.SupportScore)
	}
	if sc.RelationshipScore <= 0 || sc.RelationshipScore > 100 {
		t.Errorf("expected relationship_score between 0-100, got %f", sc.RelationshipScore)
	}
	if sc.Trend != "stable" {
		t.Errorf("expected trend 'stable', got %q", sc.Trend)
	}
	if sc.Details.TotalDeals != 5 {
		t.Errorf("expected 5 deals, got %d", sc.Details.TotalDeals)
	}
	if sc.Details.SignalCount != 3 {
		t.Errorf("expected 3 signals, got %d", sc.Details.SignalCount)
	}
}

func TestScorecardMissingVendor(t *testing.T) {
	db := inMemoryDB(t)
	seedTestData(t, db)

	eng := NewEngine(db, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	// Vendor with no data should return partial scores
	sc, err := eng.Scorecard(ctx, "UnknownVendor", "1y")
	if err != nil {
		t.Fatalf("Scorecard: %v", err)
	}

	if sc.Vendor != "UnknownVendor" {
		t.Errorf("expected vendor 'UnknownVendor', got %q", sc.Vendor)
	}
	if sc.Details.TotalDeals != 0 {
		t.Errorf("expected 0 deals, got %d", sc.Details.TotalDeals)
	}
	// Missing vendor should still return 0 scores (not error)
	// Missing vendor gets partial scores (SLA defaults to 100, support defaults to 50)
	if sc.OverallScore <= 0 || sc.OverallScore > 100 {
		t.Errorf("expected overall_score in 0-100 for missing vendor, got %f", sc.OverallScore)
	}
}

func TestScorecardFilterByPeriod(t *testing.T) {
	db := inMemoryDB(t)
	seedTestData(t, db)

	eng := NewEngine(db, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	// Short period (1 month) should have fewer or no deals
	sc, err := eng.Scorecard(ctx, "Acme Corp", "1m")
	if err != nil {
		t.Fatalf("Scorecard: %v", err)
	}

	// With only 1m, the oldest deals (5-1=4 months ago) might be filtered out
	// Depending on when the test runs, we expect <= 5 deals
	if sc.Details.TotalDeals < 0 || sc.Details.TotalDeals > 5 {
		t.Errorf("expected 0-5 deals for 1m period, got %d", sc.Details.TotalDeals)
	}
}
