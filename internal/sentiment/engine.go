package sentiment

import (
	"context"
	"fmt"
	"strings"
)

// Engine performs keyword-based sentiment analysis on text.
// It is stateless and safe for concurrent use.
type Engine struct{}

// NewEngine creates a new sentiment analysis engine.
func NewEngine() *Engine {
	return &Engine{}
}

var positiveKeywords = []string{
	"great", "excellent", "happy", "appreciate", "agree",
	"fair", "value", "fantastic", "pleased", "good",
}

var negativeKeywords = []string{
	"expensive", "too much", "unhappy", "disagree", "unfair",
	"reject", "problem", "terrible", "bad", "disappointed",
}

// Analyze scores the sentiment of the given text using keyword matching.
// Returns an error if text is empty or exceeds 10,000 characters.
func (e *Engine) Analyze(ctx context.Context, text string) (*SentimentResult, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("text must not be empty")
	}
	if len(text) > 10000 {
		return nil, fmt.Errorf("text exceeds maximum length of 10000 characters: got %d", len(text))
	}

	lower := strings.ToLower(text)

	var score float64
	var keyPhrases []string

	for _, kw := range positiveKeywords {
		if strings.Contains(lower, kw) {
			score += 0.15
			keyPhrases = append(keyPhrases, kw)
		}
	}

	for _, kw := range negativeKeywords {
		if strings.Contains(lower, kw) {
			score -= 0.15
			keyPhrases = append(keyPhrases, kw)
		}
	}

	// Clamp score to [-1.0, 1.0]
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	totalKeywords := len(keyPhrases)
	confidence := float64(totalKeywords) * 0.15
	if confidence > 1.0 {
		confidence = 1.0
	}

	var label string
	switch {
	case score > 0.2:
		label = "positive"
	case score < -0.2:
		label = "negative"
	default:
		label = "neutral"
	}

	return &SentimentResult{
		Score:      score,
		Confidence: confidence,
		Label:      label,
		KeyPhrases: keyPhrases,
	}, nil
}
