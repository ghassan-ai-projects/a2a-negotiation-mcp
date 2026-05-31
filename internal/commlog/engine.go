package commlog

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// Engine wraps store operations for vendor communication logging.
type Engine struct {
	store  *Store
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates an Engine with the given store, DB, and logger.
func NewEngine(store *Store, db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{store: store, db: db, logger: logger}
}

// Store returns the underlying store for direct access.
func (e *Engine) Store() *Store {
	return e.store
}

// Log records a new vendor communication entry.
func (e *Engine) Log(ctx context.Context, vendor, commType, summary, detail string) (*CommEntry, error) {
	if vendor == "" {
		return nil, fmt.Errorf("vendor is required")
	}
	if commType == "" {
		return nil, fmt.Errorf("comm_type is required")
	}
	if summary == "" {
		return nil, fmt.Errorf("summary is required")
	}

	entry, err := e.store.Log(ctx, vendor, commType, summary, detail)
	if err != nil {
		return nil, fmt.Errorf("commlog log: %w", err)
	}

	e.logger.Info("communication logged", "vendor", vendor, "comm_type", commType, "id", entry.ID)
	return entry, nil
}

// History returns recent communication entries for a vendor.
func (e *Engine) History(ctx context.Context, vendor string, limit int) (*CommLogResult, error) {
	if limit <= 0 {
		limit = 20
	}

	entries, total, err := e.store.ListByVendor(ctx, vendor, limit)
	if err != nil {
		return nil, fmt.Errorf("commlog history: %w", err)
	}

	if entries == nil {
		entries = []CommEntry{}
	}

	return &CommLogResult{
		Entries:    entries,
		TotalCount: total,
	}, nil
}
