package summarizer

import (
	"context"
	"testing"
)

func TestSummarize_Brief(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	result, err := e.Summarize(ctx, "session-123", "brief")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SessionID != "session-123" {
		t.Errorf("expected session-123, got %s", result.SessionID)
	}
	if result.Style != "brief" {
		t.Errorf("expected style brief, got %s", result.Style)
	}
	if result.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if result.WordCount <= 0 {
		t.Errorf("expected positive word count, got %d", result.WordCount)
	}
	if len(result.KeyPoints) < 3 {
		t.Errorf("expected at least 3 key points, got %d", len(result.KeyPoints))
	}
	// brief: 3-4 sentences
	words := result.WordCount
	if words < 20 || words > 80 {
		t.Errorf("brief summary word count %d outside expected range (20-80)", words)
	}
}

func TestSummarize_Detailed(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	result, err := e.Summarize(ctx, "session-456", "detailed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SessionID != "session-456" {
		t.Errorf("expected session-456, got %s", result.SessionID)
	}
	if result.Style != "detailed" {
		t.Errorf("expected style detailed, got %s", result.Style)
	}
	if result.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if result.WordCount <= 0 {
		t.Errorf("expected positive word count, got %d", result.WordCount)
	}
	if len(result.KeyPoints) < 3 {
		t.Errorf("expected at least 3 key points, got %d", len(result.KeyPoints))
	}
	// detailed: 8-10 sentences
	words := result.WordCount
	if words < 60 || words > 160 {
		t.Errorf("detailed summary word count %d outside expected range (60-160)", words)
	}
}

func TestSummarize_BulletPoints(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	result, err := e.Summarize(ctx, "session-789", "bullet_points")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SessionID != "session-789" {
		t.Errorf("expected session-789, got %s", result.SessionID)
	}
	if result.Style != "bullet_points" {
		t.Errorf("expected style bullet_points, got %s", result.Style)
	}
	if result.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if result.WordCount <= 0 {
		t.Errorf("expected positive word count, got %d", result.WordCount)
	}
	if len(result.KeyPoints) < 3 {
		t.Errorf("expected at least 3 key points, got %d", len(result.KeyPoints))
	}
	// bullet_points: 5-7 bullet points
	if !contains(result.Summary, "•") {
		t.Error("expected bullet points in summary")
	}
}

func TestSummarize_InvalidStyleError(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	_, err := e.Summarize(ctx, "session-000", "invalid_style")
	if err == nil {
		t.Fatal("expected error for invalid style")
	}
}

func TestSummarize_EmptySessionIDError(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	_, err := e.Summarize(ctx, "", "brief")
	if err == nil {
		t.Fatal("expected error for empty sessionID")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
