package effectiveness

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

	// Create tables needed for effectiveness score
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS negotiation_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL UNIQUE,
		vendor TEXT NOT NULL,
		sku TEXT NOT NULL DEFAULT '',
		strategy TEXT NOT NULL,
		budget REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		current_offer REAL NOT NULL DEFAULT 0,
		list_price REAL NOT NULL DEFAULT 0,
		rounds_completed INTEGER NOT NULL DEFAULT 0,
		outcome TEXT,
		user_id TEXT DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

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
	);

	CREATE TABLE IF NOT EXISTS user_streaks (
		user_id TEXT PRIMARY KEY,
		current_streak INTEGER DEFAULT 0,
		longest_streak INTEGER DEFAULT 0,
		last_negotiation_at TEXT,
		total_savings REAL DEFAULT 0,
		total_deals INTEGER DEFAULT 0
	)
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

func TestCompositeScoreCalculation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Seed negotiation sessions with won/lost outcomes (for win rate)
	_, err := db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (session_id, vendor, strategy, status, outcome, created_at, updated_at)
		VALUES ('s1', 'VendorA', 'competitive', 'closed', 'won', '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'),
		       ('s2', 'VendorA', 'competitive', 'closed', 'won', '2026-05-10T00:00:00Z', '2026-05-10T00:00:00Z'),
		       ('s3', 'VendorA', 'competitive', 'closed', 'won', '2026-05-15T00:00:00Z', '2026-05-15T00:00:00Z'),
		       ('s4', 'VendorB', 'collaborative', 'closed', 'lost', '2026-05-20T00:00:00Z', '2026-05-20T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	// Seed deal_outcomes with good discounts
	_, err = db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, list_price, final_price, discount_pct, created_at)
		VALUES ('VendorA', 1000, 800, 20, '2026-05-01T00:00:00Z'),
		       ('VendorA', 2000, 1400, 30, '2026-05-10T00:00:00Z'),
		       ('VendorA', 5000, 3500, 30, '2026-05-15T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	// Seed user streak
	_, err = db.ExecContext(ctx, `
		INSERT INTO user_streaks (user_id, current_streak, longest_streak, last_negotiation_at)
		VALUES ('user1', 3, 5, '2026-05-20T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed streak: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	score, err := eng.Score(ctx, "user1", "90d")
	if err != nil {
		t.Fatalf("Score: %v", err)
	}

	if score.OverallScore <= 0 {
		t.Fatalf("expected positive overall score, got %.2f", score.OverallScore)
	}
	if score.OverallScore > 100 {
		t.Fatalf("expected score <= 100, got %.2f", score.OverallScore)
	}

	if len(score.Components) != 4 {
		t.Fatalf("expected 4 components, got %d", len(score.Components))
	}

	// Verify component names
	expectedNames := []string{"Win Rate", "Discount Depth", "Savings Volume", "Streak Consistency"}
	for i, c := range score.Components {
		if c.Name != expectedNames[i] {
			t.Fatalf("component %d: expected %s, got %s", i, expectedNames[i], c.Name)
		}
		if c.Score < 0 || c.Score > 100 {
			t.Fatalf("component %s score %.2f out of range", c.Name, c.Score)
		}
	}

	// Win rate should be 75% (3 won / 4 total)
	if score.Components[0].Score != 75 {
		t.Fatalf("expected win rate score 75, got %.2f", score.Components[0].Score)
	}

	if len(score.Trend) == 0 {
		t.Fatal("expected trend data")
	}
	if len(score.Tips) == 0 {
		t.Fatal("expected tips")
	}
}

func TestMissingDataReturnsZeros(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	score, err := eng.Score(context.Background(), "", "90d")
	if err != nil {
		t.Fatalf("Score empty: %v", err)
	}

	if score.OverallScore != 0 {
		t.Fatalf("expected overall score 0, got %.2f", score.OverallScore)
	}
	for _, c := range score.Components {
		if c.Score != 0 {
			t.Fatalf("expected component %s score 0, got %.2f", c.Name, c.Score)
		}
	}
	if len(score.Tips) == 0 {
		t.Fatal("expected tips even with zero data")
	}
}

func TestTrendOverTime(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Seed deals across multiple months
	_, err := db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, list_price, final_price, discount_pct, created_at)
		VALUES ('VendorA', 1000, 900, 10, '2026-01-15T00:00:00Z'),
		       ('VendorA', 1000, 800, 20, '2026-02-10T00:00:00Z'),
		       ('VendorA', 1000, 700, 30, '2026-03-05T00:00:00Z'),
		       ('VendorA', 1000, 600, 40, '2026-04-20T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("seed deals: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	eng := NewEngine(db, logger)

	score, err := eng.Score(ctx, "", "1y")
	if err != nil {
		t.Fatalf("Score: %v", err)
	}

	// Should have multiple trend points (one per month)
	if len(score.Trend) < 3 {
		t.Fatalf("expected at least 3 trend points, got %d", len(score.Trend))
	}

	// Verify trend points are in order
	for i := 1; i < len(score.Trend); i++ {
		if score.Trend[i].Date <= score.Trend[i-1].Date {
			t.Fatalf("trend not sorted: %s <= %s", score.Trend[i].Date, score.Trend[i-1].Date)
		}
	}
}
