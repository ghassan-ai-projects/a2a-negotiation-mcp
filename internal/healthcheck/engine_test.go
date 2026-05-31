package healthcheck

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCheck_DatabaseOK(t *testing.T) {
	db := newTestDB(t)
	startTime := time.Now()
	eng := NewEngine(db, 42, startTime, "")

	ctx := context.Background()
	result := eng.Check(ctx)

	if !result.DatabaseOK {
		t.Error("expected database_ok to be true")
	}
	if result.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", result.Status)
	}
	if result.ToolCount != 42 {
		t.Errorf("expected tool_count 42, got %d", result.ToolCount)
	}
	if result.UptimeSecs < 0 {
		t.Errorf("expected positive uptime, got %d", result.UptimeSecs)
	}
	if result.StartedAt == "" {
		t.Error("expected started_at to be set")
	}
}

func TestCheck_DatabaseNotOK(t *testing.T) {
	startTime := time.Now()
	// Create an engine with nil DB
	eng := NewEngine(nil, 0, startTime, "")

	ctx := context.Background()
	result := eng.Check(ctx)

	if result.DatabaseOK {
		t.Error("expected database_ok to be false with nil DB")
	}
	if result.Status != "degraded" {
		t.Errorf("expected status degraded, got %s", result.Status)
	}
}

func TestCheck_ToolCountGreaterThanZero(t *testing.T) {
	db := newTestDB(t)
	startTime := time.Now()
	eng := NewEngine(db, 5, startTime, "")

	ctx := context.Background()
	result := eng.Check(ctx)

	if result.ToolCount <= 0 {
		t.Errorf("expected tool_count > 0, got %d", result.ToolCount)
	}
}
