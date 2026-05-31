package ratelimitdashboard

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	// DefaultDailyLimit is the default maximum requests per day.
	DefaultDailyLimit = 100
)

// Engine provides rate limit dashboard functionality.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a new rate limit dashboard engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// GetStatus returns the current rate limit usage status.
func (e *Engine) GetStatus(ctx context.Context) (*RateLimitStatus, error) {
	now := time.Now().UTC()

	// Count requests this minute
	startOfMinute := now.Truncate(time.Minute)
	requestsThisMinute, err := e.store.CountSince(ctx, startOfMinute)
	if err != nil {
		return nil, fmt.Errorf("count this minute: %w", err)
	}

	// Count requests this hour
	startOfHour := now.Truncate(time.Hour)
	requestsThisHour, err := e.store.CountSince(ctx, startOfHour)
	if err != nil {
		return nil, fmt.Errorf("count this hour: %w", err)
	}

	// Count requests today
	requestsToday, err := e.store.CountToday(ctx)
	if err != nil {
		return nil, fmt.Errorf("count today: %w", err)
	}

	remainingBudget := DefaultDailyLimit - requestsToday
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	// Determine status color
	status := ColorGreen
	if requestsToday >= 80 {
		status = ColorRed
	} else if requestsToday >= 50 {
		status = ColorYellow
	}

	return &RateLimitStatus{
		RateLimitConfig:    PerSecond{PerSecond: 0}, // unlimited by default
		RequestsThisMinute: requestsThisMinute,
		RequestsThisHour:   requestsThisHour,
		RequestsToday:      requestsToday,
		RemainingBudget:    remainingBudget,
		Status:             status,
	}, nil
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
