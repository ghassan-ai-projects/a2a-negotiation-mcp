package auditlog

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Engine provides audit log business logic.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates an audit log engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// LogAction records an action in the audit log.
func (e *Engine) LogAction(ctx context.Context, action, userID, details string) (*AuditEntry, error) {
	id, err := e.store.Log(ctx, action, userID, details)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}
	return &AuditEntry{
		ID:        id,
		Action:    action,
		UserID:    userID,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Search returns audit entries matching optional filters.
func (e *Engine) Search(ctx context.Context, action, userID string, limit int, since string) ([]AuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return e.store.Query(ctx, action, userID, limit, since)
}

// Summary returns aggregated audit stats.
func (e *Engine) Summary(ctx context.Context) (*AuditSummary, error) {
	return e.store.Summary(ctx)
}
