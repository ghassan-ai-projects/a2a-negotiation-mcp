package approvals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Engine provides approval workflow business logic.
type Engine struct {
	store *Store
}

// NewEngine creates an approvals engine.
func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

// Store exposes the underlying store for cross-package access.
func (e *Engine) Store() *Store { return e.store }

// RequestApproval creates a new pending approval request.
func (e *Engine) RequestApproval(ctx context.Context, sessionID, reason string, threshold float64) (*Approval, error) {
	a := &Approval{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Reason:    reason,
		Threshold: threshold,
		Status:    ApprovalPending,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := e.store.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("request approval: %w", err)
	}
	return a, nil
}

// Approve sets an approval's status to approved.
func (e *Engine) Approve(ctx context.Context, id string) (*Approval, error) {
	a, err := e.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("approve: %w", err)
	}
	if a == nil {
		return nil, fmt.Errorf("approval %s not found", id)
	}
	if a.Status != ApprovalPending {
		return nil, fmt.Errorf("approval %s already %s", id, a.Status)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := e.store.UpdateStatus(ctx, id, ApprovalApproved, now); err != nil {
		return nil, fmt.Errorf("approve update: %w", err)
	}
	a.Status = ApprovalApproved
	a.ResolvedAt = &now
	return a, nil
}

// Reject sets an approval's status to rejected.
func (e *Engine) Reject(ctx context.Context, id string) (*Approval, error) {
	a, err := e.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reject: %w", err)
	}
	if a == nil {
		return nil, fmt.Errorf("approval %s not found", id)
	}
	if a.Status != ApprovalPending {
		return nil, fmt.Errorf("approval %s already %s", id, a.Status)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := e.store.UpdateStatus(ctx, id, ApprovalRejected, now); err != nil {
		return nil, fmt.Errorf("reject update: %w", err)
	}
	a.Status = ApprovalRejected
	a.ResolvedAt = &now
	return a, nil
}

// Pending returns all pending approvals.
func (e *Engine) Pending(ctx context.Context) ([]Approval, error) {
	return e.store.ListPending(ctx)
}
