package limitedoffer

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite DB with the pricing schema and seed data.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?cache=shared")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS vendors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		category TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS pricing_snapshot (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_id INTEGER NOT NULL REFERENCES vendors(id),
		sku TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		list_price REAL NOT NULL DEFAULT 0,
		min_observed REAL NOT NULL DEFAULT 0,
		max_observed REAL NOT NULL DEFAULT 0,
		typical_pct REAL NOT NULL DEFAULT 0,
		unit TEXT NOT NULL DEFAULT 'per_seat_month',
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(vendor_id, sku)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	_, err = db.Exec(`INSERT INTO vendors (id, name, category) VALUES (1, 'Slack', 'Communication')`)
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit) VALUES (1, 'Pro', 'Pro plan', 8.75, 6.50, 8.00, 18, 'per_seat_month')`)
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	return db
}

func TestAnalyze_ExpiringOffer(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	input := &OfferInput{
		Vendor:       "Slack",
		SKU:          "Pro",
		OfferPrice:   6.00,
		CurrentPrice: 8.75,
		ExpiresAt:    time.Now().UTC().Add(2 * 24 * time.Hour), // 2 days = critical
	}

	result, err := eng.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if result.Urgency != "critical" {
		t.Errorf("expected urgency=critical for <3 days, got %s", result.Urgency)
	}
	if result.Recommendation != "accept" {
		t.Errorf("expected recommendation=accept for positive savings, got %s", result.Recommendation)
	}
	if result.Savings <= 0 {
		t.Errorf("expected positive savings, got %.2f", result.Savings)
	}
	if result.DaysRemaining <= 0 || result.DaysRemaining > 3 {
		t.Errorf("expected days_remaining between 0 and 3, got %.2f", result.DaysRemaining)
	}
	if result.VsBestPricePct <= 0 {
		t.Errorf("expected vs_best_price_pct > 0, got %.2f", result.VsBestPricePct)
	}
}

func TestAnalyze_ExpiredOffer(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	input := &OfferInput{
		Vendor:       "Slack",
		SKU:          "Pro",
		OfferPrice:   7.00,
		CurrentPrice: 8.75,
		ExpiresAt:    time.Now().UTC().Add(-1 * 24 * time.Hour), // expired
	}

	result, err := eng.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if result.DaysRemaining != 0 {
		t.Errorf("expected days_remaining=0 for expired offer, got %.2f", result.DaysRemaining)
	}
}

func TestAnalyze_Profitable(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	// Use current_spend: spend is 1000, offer is 800 => savings = 200
	input := &OfferInput{
		Vendor:       "Slack",
		SKU:          "Pro",
		OfferPrice:   800,
		CurrentSpend: 1000,
		ExpiresAt:    time.Now().UTC().Add(10 * 24 * time.Hour),
	}

	result, err := eng.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if result.Savings != 200 {
		t.Errorf("expected savings=200 (1000-800), got %.2f", result.Savings)
	}
	if result.Recommendation != "accept" {
		t.Errorf("expected recommendation=accept, got %s", result.Recommendation)
	}
	if result.Urgency != "normal" {
		t.Errorf("expected urgency=normal for >7 days, got %s", result.Urgency)
	}
}
