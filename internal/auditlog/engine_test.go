package auditlog

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) *Engine {
	t.Helper()

	db, err := sql.Open("sqlite", "file:auditlog_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("auditlog NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewEngine(store, logger)
}

func TestLogEntries(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	entry, err := eng.LogAction(ctx, "create_workspace", "user1", "Created workspace 'Test'")
	if err != nil {
		t.Fatalf("LogAction: %v", err)
	}
	if entry.ID <= 0 {
		t.Errorf("expected positive ID, got %d", entry.ID)
	}
	if entry.Action != "create_workspace" {
		t.Errorf("expected action 'create_workspace', got %q", entry.Action)
	}
	if entry.UserID != "user1" {
		t.Errorf("expected user_id 'user1', got %q", entry.UserID)
	}
	if entry.Details != "Created workspace 'Test'" {
		t.Errorf("expected details, got %q", entry.Details)
	}
	if entry.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestQueryByAction(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	_, err := eng.LogAction(ctx, "create", "user1", "Created")
	if err != nil {
		t.Fatalf("LogAction create: %v", err)
	}
	_, err = eng.LogAction(ctx, "update", "user1", "Updated")
	if err != nil {
		t.Fatalf("LogAction update: %v", err)
	}
	_, err = eng.LogAction(ctx, "delete", "user2", "Deleted")
	if err != nil {
		t.Fatalf("LogAction delete: %v", err)
	}

	// Query by action
	entries, err := eng.Search(ctx, "create", "", 10, "")
	if err != nil {
		t.Fatalf("Search by action: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for action 'create', got %d", len(entries))
	}
	if entries[0].Action != "create" {
		t.Errorf("expected action 'create', got %q", entries[0].Action)
	}
}

func TestQueryByUser(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	_, err := eng.LogAction(ctx, "action1", "user1", "First")
	if err != nil {
		t.Fatalf("LogAction action1: %v", err)
	}
	_, err = eng.LogAction(ctx, "action2", "user1", "Second")
	if err != nil {
		t.Fatalf("LogAction action2: %v", err)
	}
	_, err = eng.LogAction(ctx, "action3", "user2", "Third")
	if err != nil {
		t.Fatalf("LogAction action3: %v", err)
	}

	// Query by user
	entries, err := eng.Search(ctx, "", "user1", 10, "")
	if err != nil {
		t.Fatalf("Search by user: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for user 'user1', got %d", len(entries))
	}
	for _, e := range entries {
		if e.UserID != "user1" {
			t.Errorf("expected user_id 'user1', got %q", e.UserID)
		}
	}
}

func TestEmptyResult(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	entries, err := eng.Search(ctx, "nonexistent", "", 10, "")
	if err != nil {
		t.Fatalf("Search nonexistent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestAuditSummary(t *testing.T) {
	eng := setupTest(t)
	ctx := context.Background()

	_, err := eng.LogAction(ctx, "create", "user1", "Entry 1")
	if err != nil {
		t.Fatalf("LogAction: %v", err)
	}
	_, err = eng.LogAction(ctx, "create", "user2", "Entry 2")
	if err != nil {
		t.Fatalf("LogAction: %v", err)
	}
	_, err = eng.LogAction(ctx, "delete", "user1", "Entry 3")
	if err != nil {
		t.Fatalf("LogAction: %v", err)
	}

	summary, err := eng.Summary(ctx)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.TotalActions != 3 {
		t.Errorf("expected total_actions 3, got %d", summary.TotalActions)
	}
	if summary.ByAction["create"] != 2 {
		t.Errorf("expected 2 'create' actions, got %d", summary.ByAction["create"])
	}
	if summary.ByAction["delete"] != 1 {
		t.Errorf("expected 1 'delete' action, got %d", summary.ByAction["delete"])
	}
	if len(summary.ByDay) == 0 {
		t.Error("expected at least 1 day in by_day")
	}
}
