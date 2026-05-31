package pricingrefresh

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/trends"
	_ "modernc.org/sqlite"
)

// setupTest creates an in-memory DB with pricing data and a trends store.
func setupTest(t *testing.T) (*Engine, *trends.Store) {
	t.Helper()

	dbName := fmt.Sprintf("file:pricingrefresh_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := sql.Open("sqlite", dbName)
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

	_, err = db.Exec(`INSERT INTO vendors (id, name, category) VALUES (1, 'Slack', 'Communication'), (2, 'GitHub', 'Developer')`)
	if err != nil {
		t.Fatalf("insert vendors: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit) VALUES
		(1, 'Pro', 'Pro plan', 8.75, 6.50, 8.00, 18, 'per_seat_month'),
		(1, 'Business+', 'Business Plus', 15.00, 12.00, 14.50, 15, 'per_seat_month'),
		(2, 'Team', 'Team plan', 4.00, 3.00, 3.80, 15, 'per_seat_month')
	`)
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	trendsStore, err := trends.NewStore(db)
	if err != nil {
		t.Fatalf("create trends store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	return eng, trendsStore
}

func TestRefresh_SingleVendor(t *testing.T) {
	eng, trendsStore := setupTest(t)
	ctx := context.Background()

	result, err := eng.Refresh(ctx, []string{"Slack"}, trendsStore)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if result.VendorsRefreshed != 1 {
		t.Errorf("expected vendors_refreshed=1, got %d", result.VendorsRefreshed)
	}
	if result.RecordsUpdated != 2 {
		t.Errorf("expected records_updated=2 for Slack (2 SKUs), got %d", result.RecordsUpdated)
	}
	// DurationMs may be 0 for very fast operations; just check it's non-negative
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration_ms, got %d", result.DurationMs)
	}
}

func TestRefresh_Count(t *testing.T) {
	eng, trendsStore := setupTest(t)
	ctx := context.Background()

	result, err := eng.Refresh(ctx, nil, trendsStore)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if result.VendorsRefreshed != 2 {
		t.Errorf("expected vendors_refreshed=2, got %d", result.VendorsRefreshed)
	}
	if result.RecordsUpdated != 3 {
		t.Errorf("expected records_updated=3 (3 total SKUs), got %d", result.RecordsUpdated)
	}
}
