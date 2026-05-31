package workspaces

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Engine provides workspace business logic.
type Engine struct {
	store  *Store
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a workspaces engine.
func NewEngine(store *Store, db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{store: store, db: db, logger: logger}
}

// Create creates a new workspace.
func (e *Engine) Create(ctx context.Context, name, description string) (*Workspace, error) {
	id := uuid.New().String()
	now := time.Now().UTC()
	if err := e.store.Create(ctx, id, name, description, now); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return &Workspace{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// List returns all workspaces.
func (e *Engine) List(ctx context.Context) ([]Workspace, error) {
	return e.store.List(ctx)
}

// Summary returns aggregated data for a workspace.
func (e *Engine) Summary(ctx context.Context, id string) (*WorkspaceSummary, error) {
	ws, err := e.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	if ws == nil {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}

	summary := &WorkspaceSummary{
		ID:   ws.ID,
		Name: ws.Name,
	}

	if err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT vendor), COUNT(*), COALESCE(SUM(list_price - final_price), 0)
		FROM deal_outcomes
	`).Scan(&summary.VendorCount, &summary.DealCount, &summary.TotalSavings); err != nil {
		e.logger.Warn("workspace summary: deal query failed", "error", err)
	}

	summary.MemberCount = 1
	return summary, nil
}
