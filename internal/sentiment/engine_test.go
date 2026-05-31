package sentiment

import (
	"context"
	"testing"
)

func TestAnalyze_Positive(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	text := "This is a great proposal! We are very happy and appreciate the excellent value you've offered."
	result, err := eng.Analyze(ctx, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Score <= 0.2 {
		t.Errorf("expected positive score > 0.2, got %f", result.Score)
	}
	if result.Label != "positive" {
		t.Errorf("expected label 'positive', got %q", result.Label)
	}
	if len(result.KeyPhrases) == 0 {
		t.Errorf("expected at least one key phrase, got none")
	}
	if result.Confidence <= 0 {
		t.Errorf("expected positive confidence, got %f", result.Confidence)
	}
}

func TestAnalyze_Negative(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	text := "This price is too much and expensive. We are unhappy with this bad offer."
	result, err := eng.Analyze(ctx, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Score >= -0.2 {
		t.Errorf("expected negative score < -0.2, got %f", result.Score)
	}
	if result.Label != "negative" {
		t.Errorf("expected label 'negative', got %q", result.Label)
	}
	if len(result.KeyPhrases) == 0 {
		t.Errorf("expected at least one key phrase, got none")
	}
}

func TestAnalyze_Neutral(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	text := "We have received your proposal and will review it shortly."
	result, err := eng.Analyze(ctx, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Label != "neutral" {
		t.Errorf("expected label 'neutral', got %q", result.Label)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0 for neutral text, got %f", result.Score)
	}
}

func TestAnalyze_Mixed(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	text := "We appreciate the offer but the pricing is too much. The value is good though."
	result, err := eng.Analyze(ctx, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.KeyPhrases) == 0 {
		t.Errorf("expected at least one key phrase, got none")
	}
	// Should contain both positive and negative keywords
	hasPositive := false
	hasNegative := false
	for _, kp := range result.KeyPhrases {
		switch kp {
		case "appreciate", "value", "good":
			hasPositive = true
		case "too much":
			hasNegative = true
		}
	}
	if !hasPositive {
		t.Errorf("expected at least one positive keyword, got %v", result.KeyPhrases)
	}
	if !hasNegative {
		t.Errorf("expected at least one negative keyword, got %v", result.KeyPhrases)
	}
}

func TestAnalyze_EmptyText_ReturnsError(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.Analyze(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestAnalyze_ExceedsMaxLength_ReturnsError(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	text := string(make([]byte, 10001))
	_, err := eng.Analyze(ctx, text)
	if err == nil {
		t.Fatal("expected error for text exceeding max length, got nil")
	}
}
