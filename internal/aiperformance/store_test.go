package aiperformance

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

func TestLogCall(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	entry, err := s.LogCall(ctx, "gpt-4", "negotiate_query_price", 1234, 567, true, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}

	if entry.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %s", entry.Model)
	}
	if entry.ToolName != "negotiate_query_price" {
		t.Errorf("expected tool_name 'negotiate_query_price', got %s", entry.ToolName)
	}
	if entry.LatencyMs != 1234 {
		t.Errorf("expected latency_ms 1234, got %d", entry.LatencyMs)
	}
	if entry.TokensUsed != 567 {
		t.Errorf("expected tokens_used 567, got %d", entry.TokensUsed)
	}
	if entry.Success != true {
		t.Errorf("expected success true, got %v", entry.Success)
	}
	if entry.NegotiationType != "price_query" {
		t.Errorf("expected negotiation_type 'price_query', got %s", entry.NegotiationType)
	}
	if entry.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if entry.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
}

func TestGetSummary_MultipleModels(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Log calls for two different models
	_, err := s.LogCall(ctx, "gpt-4", "tool_a", 100, 10, true, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}
	_, err = s.LogCall(ctx, "gpt-4", "tool_b", 200, 20, false, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}
	_, err = s.LogCall(ctx, "claude-3", "tool_c", 300, 30, true, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}

	summaries, err := s.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Check gpt-4 summary
	var gptSummary *ProviderSummary
	var claudeSummary *ProviderSummary
	for i := range summaries {
		switch summaries[i].Model {
		case "gpt-4":
			gptSummary = &summaries[i]
		case "claude-3":
			claudeSummary = &summaries[i]
		}
	}
	if gptSummary == nil {
		t.Fatal("expected gpt-4 summary")
	}
	if gptSummary.TotalCalls != 2 {
		t.Errorf("expected 2 total_calls for gpt-4, got %d", gptSummary.TotalCalls)
	}
	if gptSummary.SuccessRate != 0.5 {
		t.Errorf("expected 0.5 success_rate for gpt-4, got %f", gptSummary.SuccessRate)
	}
	if gptSummary.TotalTokens != 30 {
		t.Errorf("expected 30 total_tokens for gpt-4, got %d", gptSummary.TotalTokens)
	}

	// Check claude-3 summary
	if claudeSummary == nil {
		t.Fatal("expected claude-3 summary")
	}
	if claudeSummary.TotalCalls != 1 {
		t.Errorf("expected 1 total_calls for claude-3, got %d", claudeSummary.TotalCalls)
	}
	if claudeSummary.SuccessRate != 1.0 {
		t.Errorf("expected 1.0 success_rate for claude-3, got %f", claudeSummary.SuccessRate)
	}
}

func TestGetCalls_FilteredByModel(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.LogCall(ctx, "gpt-4", "tool_a", 100, 10, true, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}
	_, err = s.LogCall(ctx, "claude-3", "tool_b", 200, 20, true, "price_query")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}
	_, err = s.LogCall(ctx, "gpt-4", "tool_c", 300, 30, false, "create_session")
	if err != nil {
		t.Fatalf("LogCall: %v", err)
	}

	calls, err := s.GetCalls(ctx, "gpt-4", 10)
	if err != nil {
		t.Fatalf("GetCalls: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls for gpt-4, got %d", len(calls))
	}

	// Check both gpt-4 calls are present
	callMap := make(map[string]bool)
	for _, c := range calls {
		if c.ToolName == "tool_a" {
			callMap["tool_a"] = true
			if !c.Success {
				t.Error("expected tool_a success true")
			}
			if c.NegotiationType != "price_query" {
				t.Errorf("expected tool_a negotiation_type 'price_query', got %s", c.NegotiationType)
			}
		}
		if c.ToolName == "tool_c" {
			callMap["tool_c"] = true
			if c.Success {
				t.Error("expected tool_c success false")
			}
			if c.NegotiationType != "create_session" {
				t.Errorf("expected tool_c negotiation_type 'create_session', got %s", c.NegotiationType)
			}
		}
	}

	if !callMap["tool_a"] {
		t.Error("expected tool_a in results")
	}
	if !callMap["tool_c"] {
		t.Error("expected tool_c in results")
	}
}

func TestGetCalls_Limit(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := s.LogCall(ctx, "gpt-4", "tool", 100, 10, true, "price_query")
		if err != nil {
			t.Fatalf("LogCall: %v", err)
		}
	}

	calls, err := s.GetCalls(ctx, "gpt-4", 3)
	if err != nil {
		t.Fatalf("GetCalls: %v", err)
	}

	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
	}
}

func TestGetSummary_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	summaries, err := s.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if summaries == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries, got %d", len(summaries))
	}
}

func TestGetCalls_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	calls, err := s.GetCalls(ctx, "gpt-4", 10)
	if err != nil {
		t.Fatalf("GetCalls: %v", err)
	}
	if calls == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}
}
