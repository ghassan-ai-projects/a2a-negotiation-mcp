package alerthistory

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create the tables we'll query
	_, err = db.Exec(`
		CREATE TABLE budget_alert_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor TEXT NOT NULL,
			budget REAL NOT NULL,
			actual REAL NOT NULL,
			consumed_pct REAL NOT NULL,
			level TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE contracts (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			vendor TEXT,
			sku TEXT,
			seats INTEGER,
			current_price REAL,
			renewal_date TEXT,
			status TEXT DEFAULT 'active',
			last_negotiated_price REAL,
			created_at TEXT
		);
		CREATE TABLE price_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor TEXT NOT NULL,
			sku TEXT NOT NULL DEFAULT '',
			price REAL NOT NULL,
			list_price REAL NOT NULL DEFAULT 0,
			date TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db
}

func TestGetAlerts_WithBudgetData(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec(`
		INSERT INTO budget_alert_history (vendor, budget, actual, consumed_pct, level, created_at)
		VALUES ('Slack', 1000, 950, 95.0, 'warning', '2025-01-15T10:00:00Z')
	`)
	if err != nil {
		t.Fatalf("insert budget alert: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO contracts (id, user_id, vendor, sku, seats, current_price, renewal_date, status, last_negotiated_price, created_at)
		VALUES ('c1', 'user1', 'Slack', 'Pro', 50, 8.75, '2025-06-01T00:00:00Z', 'active', 0, '2025-01-01T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("insert contract: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	feed, err := eng.GetAlerts(context.Background(), "all", "", 10)
	if err != nil {
		t.Fatalf("GetAlerts: %v", err)
	}

	if len(feed.Entries) == 0 {
		t.Fatal("expected at least one alert entry")
	}

	// Check that the feed has grouped entries
	if len(feed.Grouped) == 0 {
		t.Fatal("expected grouped entries")
	}
}

func TestGetAlerts_EmptyReturnsNoEntries(t *testing.T) {
	db := setupTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(db, logger)

	feed, err := eng.GetAlerts(context.Background(), "all", "", 10)
	if err != nil {
		t.Fatalf("GetAlerts: %v", err)
	}

	if len(feed.Entries) != 0 {
		t.Errorf("expected 0 entries for empty DB, got %d", len(feed.Entries))
	}
	if len(feed.Grouped) != 0 {
		t.Errorf("expected empty grouped map, got %d keys", len(feed.Grouped))
	}
}
