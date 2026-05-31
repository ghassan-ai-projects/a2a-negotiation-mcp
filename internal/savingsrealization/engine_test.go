package savingsrealization

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

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
	eng := NewEngine(store, logger)
	return eng, store, db
}

func TestRecord(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()

	rec, err := eng.Record(ctx, "s1", "Slack", 1000, 800, "monthly")
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if rec.ProjectedAmount != 1000 {
		t.Errorf("expected projected 1000, got %f", rec.ProjectedAmount)
	}
	if rec.ActualAmount != 800 {
		t.Errorf("expected actual 800, got %f", rec.ActualAmount)
	}
}

func TestGetReport_WithData(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()
	eng.Record(ctx, "s1", "Slack", 1000, 800, "90d")
	eng.Record(ctx, "s2", "Slack", 500, 500, "90d")

	report, err := eng.GetReport(ctx, "90d")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if report.TotalProjected <= 0 {
		t.Errorf("expected positive projected, got %f", report.TotalProjected)
	}
	if report.RealizationRate <= 0 {
		t.Errorf("expected positive realization rate, got %f", report.RealizationRate)
	}
}

func TestGetReport_Empty(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()

	report, err := eng.GetReport(ctx, "90d")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if report.TotalProjected != 0 {
		t.Errorf("expected 0 projected with no data, got %f", report.TotalProjected)
	}
}
