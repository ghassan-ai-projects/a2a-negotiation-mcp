package budgetmgmt

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) (*Engine, *Store, *sql.DB) {
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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, db, logger)
	return eng, store, db
}

func TestSetMonthlyBudget(t *testing.T) {
	eng, store, _ := setupTest(t)
	ctx := context.Background()
	err := eng.SetBudget(ctx, "Slack", "2026-06", 1000)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	b, err := store.GetMonthlyBudget(ctx, "Slack", "2026-06")
	if err != nil {
		t.Fatalf("GetMonthlyBudget: %v", err)
	}
	if b.BudgetAmount != 1000 {
		t.Errorf("expected budget 1000, got %f", b.BudgetAmount)
	}
}

func TestGetForecast(t *testing.T) {
	eng, store, _ := setupTest(t)
	ctx := context.Background()

	now := time.Now()
	month := now.Format("2006-01")
	err := eng.SetBudget(ctx, "Slack", month, 1000)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}

	forecast, err := eng.GetForecast(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetForecast: %v", err)
	}
	if forecast.YTDBudget != 1000 {
		t.Errorf("expected YTD budget 1000, got %f", forecast.YTDBudget)
	}
	_ = store
}

func TestRollover(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()
	eng.SetBudget(ctx, "Slack", "2026-06", 1000)
	_, err := eng.Rollover(ctx, "Slack", "2026-06", "2026-07")
	if err != nil {
		t.Fatalf("Rollover: %v", err)
	}
}
