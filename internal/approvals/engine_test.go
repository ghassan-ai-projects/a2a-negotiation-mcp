package approvals

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupApprovalsStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestRequestApproval_CreatesPending(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a, err := eng.RequestApproval(ctx, "session-1", "High spend", 50000)
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}

	if a.SessionID != "session-1" {
		t.Errorf("expected session_id 'session-1', got %q", a.SessionID)
	}
	if a.Status != ApprovalPending {
		t.Errorf("expected status 'pending', got %q", a.Status)
	}
	if a.Threshold != 50000 {
		t.Errorf("expected threshold 50000, got %f", a.Threshold)
	}
	if a.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestApprove_UpdatesStatus(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a, err := eng.RequestApproval(ctx, "session-1", "Need approval", 10000)
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}

	approved, err := eng.Approve(ctx, a.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}

	if approved.Status != ApprovalApproved {
		t.Errorf("expected status 'approved', got %q", approved.Status)
	}
	if approved.ResolvedAt == nil || *approved.ResolvedAt == "" {
		t.Error("expected resolved_at to be set")
	}
}

func TestReject_UpdatesStatus(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a, err := eng.RequestApproval(ctx, "session-1", "Too expensive", 5000)
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}

	rejected, err := eng.Reject(ctx, a.ID)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}

	if rejected.Status != ApprovalRejected {
		t.Errorf("expected status 'rejected', got %q", rejected.Status)
	}
	if rejected.ResolvedAt == nil || *rejected.ResolvedAt == "" {
		t.Error("expected resolved_at to be set")
	}
}

func TestListPending_OnlyPending(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a1, _ := eng.RequestApproval(ctx, "session-1", "Pending 1", 0)
	eng.RequestApproval(ctx, "session-2", "Pending 2", 0)
	a3, _ := eng.RequestApproval(ctx, "session-3", "Will be approved", 0)

	eng.Approve(ctx, a3.ID)
	eng.Approve(ctx, a1.ID)

	pending, err := eng.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}

	// Should have 1 pending (session-2)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0].SessionID != "session-2" {
		t.Errorf("expected pending session-2, got %q", pending[0].SessionID)
	}
}

func TestApprove_AlreadyResolved_Error(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a, _ := eng.RequestApproval(ctx, "session-1", "Test", 0)
	eng.Approve(ctx, a.ID)

	_, err := eng.Approve(ctx, a.ID)
	if err == nil {
		t.Fatal("expected error when approving already-resolved approval")
	}
}

func TestReject_AlreadyResolved_Error(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	a, _ := eng.RequestApproval(ctx, "session-1", "Test", 0)
	eng.Reject(ctx, a.ID)

	_, err := eng.Reject(ctx, a.ID)
	if err == nil {
		t.Fatal("expected error when rejecting already-resolved approval")
	}
}

func TestApprove_NotFound(t *testing.T) {
	store := setupApprovalsStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	_, err := eng.Approve(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent approval")
	}
}
