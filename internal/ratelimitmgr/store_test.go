package ratelimitmgr

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

func TestGetConfig_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	configs, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if configs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}

func TestSetAndGetConfig(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	cfg, err := s.SetConfig(ctx, "test_tool", 50, 30)
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if cfg.ToolName != "test_tool" {
		t.Errorf("expected tool_name 'test_tool', got %s", cfg.ToolName)
	}
	if cfg.MaxCalls != 50 {
		t.Errorf("expected max_calls 50, got %d", cfg.MaxCalls)
	}
	if cfg.WindowSeconds != 30 {
		t.Errorf("expected window_seconds 30, got %d", cfg.WindowSeconds)
	}
	if cfg.UpdatedAt == "" {
		t.Errorf("expected non-empty updated_at")
	}

	configs, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].ToolName != "test_tool" {
		t.Errorf("expected 'test_tool', got %s", configs[0].ToolName)
	}
}

func TestSetConfig_UpdateExisting(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.SetConfig(ctx, "tool_a", 100, 60)
	if err != nil {
		t.Fatalf("SetConfig (initial): %v", err)
	}

	_, err = s.SetConfig(ctx, "tool_a", 200, 120)
	if err != nil {
		t.Fatalf("SetConfig (update): %v", err)
	}

	configs, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].MaxCalls != 200 {
		t.Errorf("expected max_calls 200, got %d", configs[0].MaxCalls)
	}
	if configs[0].WindowSeconds != 120 {
		t.Errorf("expected window_seconds 120, got %d", configs[0].WindowSeconds)
	}
}

func TestSetConfig_MultipleTools(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.SetConfig(ctx, "tool_a", 100, 60)
	if err != nil {
		t.Fatalf("SetConfig tool_a: %v", err)
	}
	_, err = s.SetConfig(ctx, "tool_b", 200, 120)
	if err != nil {
		t.Fatalf("SetConfig tool_b: %v", err)
	}
	_, err = s.SetConfig(ctx, "tool_c", 50, 30)
	if err != nil {
		t.Fatalf("SetConfig tool_c: %v", err)
	}

	configs, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(configs) != 3 {
		t.Fatalf("expected 3 configs, got %d", len(configs))
	}
}

func TestGetHits_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	hits, err := s.GetHits(ctx, "")
	if err != nil {
		t.Fatalf("GetHits: %v", err)
	}
	if hits == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(hits))
	}
}
