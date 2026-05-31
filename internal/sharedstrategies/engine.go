package sharedstrategies

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Engine provides shared strategy business logic.
type Engine struct {
	store *Store
}

// NewEngine creates a shared strategy engine.
func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

// Store exposes the underlying store for cross-package access.
func (e *Engine) Store() *Store { return e.store }

// ShareStrategy creates a new shared strategy.
func (e *Engine) ShareStrategy(ctx context.Context, name, notes, strategyType string) (*SharedStrategy, error) {
	st := &SharedStrategy{
		ID:           uuid.New().String(),
		Name:         name,
		Notes:        notes,
		StrategyType: strategyType,
		UsageCount:   0,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if strategyType == "" {
		st.StrategyType = "balanced"
	}
	if err := e.store.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("share strategy: %w", err)
	}
	return st, nil
}

// List returns all shared strategies.
func (e *Engine) List(ctx context.Context) ([]SharedStrategy, error) {
	return e.store.List(ctx)
}

// ImportStrategy increments the usage count for a strategy and returns it.
func (e *Engine) ImportStrategy(ctx context.Context, id string) (*SharedStrategy, error) {
	st, err := e.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("import strategy: %w", err)
	}
	if st == nil {
		return nil, fmt.Errorf("strategy %s not found", id)
	}
	if err := e.store.IncrementUsage(ctx, id); err != nil {
		return nil, fmt.Errorf("import strategy increment: %w", err)
	}
	st.UsageCount++
	return st, nil
}
