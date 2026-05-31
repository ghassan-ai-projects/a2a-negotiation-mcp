package commlog

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) (*Engine, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(store, db, logger)
	return eng, db
}

func TestLogEntry(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	entry, err := eng.Log(ctx, "Slack", "email", "Sent renewal proposal", "Proposed 15% discount for 3-year term")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	if entry.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if entry.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", entry.Vendor)
	}
	if entry.CommType != "email" {
		t.Errorf("expected comm_type email, got %s", entry.CommType)
	}
	if entry.Summary != "Sent renewal proposal" {
		t.Errorf("expected summary 'Sent renewal proposal', got %s", entry.Summary)
	}
	if entry.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestListByVendor(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	_, err := eng.Log(ctx, "Slack", "email", "First contact", "")
	if err != nil {
		t.Fatalf("Log 1: %v", err)
	}
	_, err = eng.Log(ctx, "Slack", "call", "Follow-up call", "Discussed pricing")
	if err != nil {
		t.Fatalf("Log 2: %v", err)
	}
	_, err = eng.Log(ctx, "Zoom", "email", "Zoom intro", "")
	if err != nil {
		t.Fatalf("Log 3: %v", err)
	}

	result, err := eng.History(ctx, "Slack", 10)
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected total_count=2 for Slack, got %d", result.TotalCount)
	}
	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries for Slack, got %d", len(result.Entries))
	}
}

func TestListByVendor_NoEntries(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	result, err := eng.History(ctx, "UnknownVendor", 10)
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	if result.TotalCount != 0 {
		t.Errorf("expected total_count=0, got %d", result.TotalCount)
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
}
