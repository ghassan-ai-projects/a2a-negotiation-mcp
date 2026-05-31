package webhooklog

import (
	"context"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func seedEvents(t *testing.T, s *Store, ctx context.Context) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, ev := range []struct {
		eventType string
		payload   string
		status    string
	}{
		{"contract.signed", `{"contract_id":1}`, "success"},
		{"contract.expired", `{"contract_id":2}`, "success"},
		{"payment.failed", `{"amount":500}`, "failed"},
	} {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO webhook_event_log (event_type, payload, status, attempts, created_at)
			VALUES (?, ?, ?, 0, ?)
		`, ev.eventType, ev.payload, ev.status, now)
		if err != nil {
			t.Fatalf("seed event: %v", err)
		}
	}
}

func TestListEvents(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()
	seedEvents(t, s, ctx)

	events, err := s.ListEvents(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestListEvents_FilteredByStatus(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()
	seedEvents(t, s, ctx)

	events, err := s.ListEvents(ctx, "failed", 0)
	if err != nil {
		t.Fatalf("ListEvents(failed): %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 failed event, got %d", len(events))
	}
	if events[0].EventType != "payment.failed" {
		t.Errorf("expected payment.failed, got %s", events[0].EventType)
	}
}

func TestListEvents_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	events, err := s.ListEvents(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestGetEvent(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()
	seedEvents(t, s, ctx)

	events, err := s.ListEvents(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no events seeded")
	}

	got, err := s.GetEvent(ctx, events[0].ID)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.ID != events[0].ID {
		t.Errorf("expected id %d, got %d", events[0].ID, got.ID)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetEvent(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent event")
	}
}

func TestReplayEvent(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()
	seedEvents(t, s, ctx)

	events, err := s.ListEvents(ctx, "", 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no events seeded")
	}

	replayed, err := s.ReplayEvent(ctx, events[0].ID)
	if err != nil {
		t.Fatalf("ReplayEvent: %v", err)
	}
	if replayed.Status != "replayed" {
		t.Errorf("expected status 'replayed', got %s", replayed.Status)
	}
	if replayed.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", replayed.Attempts)
	}
}

func TestReplayEvent_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.ReplayEvent(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent event")
	}
}

func TestGetStats(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()
	seedEvents(t, s, ctx)

	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalEvents != 3 {
		t.Errorf("expected 3 total events, got %d", stats.TotalEvents)
	}
	if stats.SuccessRate != 2.0/3.0 {
		t.Errorf("expected success rate ~0.667, got %f", stats.SuccessRate)
	}
	if stats.AvgAttempts != 0.0 {
		t.Errorf("expected avg attempts 0, got %f", stats.AvgAttempts)
	}
	if len(stats.StatusBreakdown) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(stats.StatusBreakdown))
	}
	if stats.StatusBreakdown["success"] != 2 {
		t.Errorf("expected 2 success events, got %d", stats.StatusBreakdown["success"])
	}
	if stats.StatusBreakdown["failed"] != 1 {
		t.Errorf("expected 1 failed event, got %d", stats.StatusBreakdown["failed"])
	}
}

func TestGetStats_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalEvents != 0 {
		t.Errorf("expected 0 events, got %d", stats.TotalEvents)
	}
	if stats.SuccessRate != 0.0 {
		t.Errorf("expected 0 success rate, got %f", stats.SuccessRate)
	}
	if stats.AvgAttempts != 0.0 {
		t.Errorf("expected 0 avg attempts, got %f", stats.AvgAttempts)
	}
	if len(stats.StatusBreakdown) != 0 {
		t.Errorf("expected empty status breakdown, got %d", len(stats.StatusBreakdown))
	}
}
