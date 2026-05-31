package notes

import (
	"context"
	"fmt"
	"time"
)

// Engine provides notes business logic.
type Engine struct {
	store *Store
}

// NewEngine creates a notes engine.
func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

// Store exposes the underlying store for cross-package access.
func (e *Engine) Store() *Store { return e.store }

// AddNote adds a note to a negotiation session.
func (e *Engine) AddNote(ctx context.Context, sessionID, content string) (*NegotiationNote, error) {
	note := &NegotiationNote{
		SessionID: sessionID,
		Content:   content,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := e.store.Add(ctx, note); err != nil {
		return nil, fmt.Errorf("add note: %w", err)
	}
	return note, nil
}

// ListNotes returns all notes for a session.
func (e *Engine) ListNotes(ctx context.Context, sessionID string) ([]NegotiationNote, error) {
	return e.store.ListBySession(ctx, sessionID)
}

// DeleteNote removes a note by ID.
func (e *Engine) DeleteNote(ctx context.Context, id int64) error {
	return e.store.Delete(ctx, id)
}
