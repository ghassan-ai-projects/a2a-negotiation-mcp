package budgetalerts

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func inMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEngine_CheckBudgets_Thresholds(t *testing.T) {
	db := inMemoryDB(t)

	// Create spend_budgets table and deal_outcomes for cross-store queries
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS spend_budgets (
			vendor TEXT PRIMARY KEY,
			budget_amount REAL NOT NULL
		);
		INSERT INTO spend_budgets VALUES ('Slack', 1000);
		INSERT INTO spend_budgets VALUES ('GitHub', 2000);
		INSERT INTO spend_budgets VALUES ('AWS', 500);

		CREATE TABLE IF NOT EXISTS deal_outcomes (
			vendor TEXT,
			final_price REAL
		);
		INSERT INTO deal_outcomes VALUES ('Slack', 850);    -- 85% → info (>80%)
		INSERT INTO deal_outcomes VALUES ('GitHub', 1900);  -- 95% → warning (>90%)
		INSERT INTO deal_outcomes VALUES ('AWS', 600);      -- 120% → critical (>100%)
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(s, db, testLogger())
	alerts, err := eng.CheckBudgets(context.Background())
	if err != nil {
		t.Fatalf("CheckBudgets: %v", err)
	}

	if len(alerts) != 3 {
		t.Fatalf("got %d alerts, want 3", len(alerts))
	}

	// Slack: 850/1000 = 85% → info
	if alerts[0].Level != LevelInfo {
		t.Errorf("Slack level = %s, want info", alerts[0].Level)
	}
	if alerts[0].ConsumedPct < 84 || alerts[0].ConsumedPct > 86 {
		t.Errorf("Slack consumed_pct = %f, want ~85", alerts[0].ConsumedPct)
	}

	// GitHub: 1900/2000 = 95% → warning
	if alerts[1].Level != LevelWarning {
		t.Errorf("GitHub level = %s, want warning", alerts[1].Level)
	}

	// AWS: 600/500 = 120% → critical
	if alerts[2].Level != LevelCritical {
		t.Errorf("AWS level = %s, want critical", alerts[2].Level)
	}
}

func TestEngine_CheckBudgets_UnderThreshold(t *testing.T) {
	db := inMemoryDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS spend_budgets (
			vendor TEXT PRIMARY KEY,
			budget_amount REAL NOT NULL
		);
		INSERT INTO spend_budgets VALUES ('Slack', 1000);

		CREATE TABLE IF NOT EXISTS deal_outcomes (
			vendor TEXT,
			final_price REAL
		);
		INSERT INTO deal_outcomes VALUES ('Slack', 500);  -- 50% → no alert
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(s, db, testLogger())
	alerts, err := eng.CheckBudgets(context.Background())
	if err != nil {
		t.Fatalf("CheckBudgets: %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("got %d alerts, want 0 (50%% spend is under 80%% threshold)", len(alerts))
	}
}

func TestEngine_AlertHistory(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Background()
	now := "2026-01-15T12:00:00Z"

	records := []*BudgetAlertHistory{
		{Vendor: "Slack", Budget: 1000, Actual: 950, ConsumedPct: 95, Level: LevelWarning, CreatedAt: now},
		{Vendor: "Slack", Budget: 1000, Actual: 1100, ConsumedPct: 110, Level: LevelCritical, CreatedAt: now},
		{Vendor: "Slack", Budget: 1000, Actual: 850, ConsumedPct: 85, Level: LevelInfo, CreatedAt: now},
	}
	for _, r := range records {
		if err := s.Save(ctx, r); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	// List with limit
	history, err := s.List(ctx, "Slack", 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("got %d history records, want 2", len(history))
	}

	// Default limit (10)
	history, err = s.List(ctx, "Slack", 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("got %d history records, want 3", len(history))
	}
}
