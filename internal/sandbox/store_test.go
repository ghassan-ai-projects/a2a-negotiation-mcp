package sandbox

import (
	"context"
	"testing"

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

func TestRecordAndGetHistory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.RecordExecution(ctx, "negotiate_query_price", `{"vendor":"Slack"}`, "result data")
	if err != nil {
		t.Fatalf("RecordExecution: %v", err)
	}

	if saved.ToolName != "negotiate_query_price" {
		t.Errorf("expected tool_name 'negotiate_query_price', got %s", saved.ToolName)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	history, err := s.GetHistory(ctx)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 execution, got %d", len(history))
	}

	if history[0].ID != saved.ID {
		t.Errorf("expected ID %d, got %d", saved.ID, history[0].ID)
	}
}

func TestGetHistory_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	history, err := s.GetHistory(ctx)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if history == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(history) != 0 {
		t.Errorf("expected 0 executions, got %d", len(history))
	}
}

func TestGetHistory_Limit(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		_, err := s.RecordExecution(ctx, "tool_a", "{}", "ok")
		if err != nil {
			t.Fatalf("RecordExecution %d: %v", i, err)
		}
	}

	history, err := s.GetHistory(ctx)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) > 20 {
		t.Errorf("expected at most 20 executions, got %d", len(history))
	}
}

func TestResetHistory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.RecordExecution(ctx, "tool_a", "{}", "result")
	if err != nil {
		t.Fatalf("RecordExecution: %v", err)
	}

	if err := s.ResetHistory(ctx); err != nil {
		t.Fatalf("ResetHistory: %v", err)
	}

	history, err := s.GetHistory(ctx)
	if err != nil {
		t.Fatalf("GetHistory after reset: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 executions after reset, got %d", len(history))
	}
}

func TestMultipleRecords(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	tools := []string{"tool_a", "tool_b", "tool_c"}
	for _, tool := range tools {
		_, err := s.RecordExecution(ctx, tool, "{}", "result")
		if err != nil {
			t.Fatalf("RecordExecution(%s): %v", tool, err)
		}
	}

	history, err := s.GetHistory(ctx)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 executions, got %d", len(history))
	}
}
