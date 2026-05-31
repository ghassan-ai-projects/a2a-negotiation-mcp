package ratelimitmgr

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages rate limit configuration and hit tracking in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate ratelimitmgr: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS rate_limit_config (
		tool_name TEXT PRIMARY KEY,
		max_calls INTEGER NOT NULL DEFAULT 100,
		window_seconds INTEGER NOT NULL DEFAULT 60,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS rate_limit_hits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tool_name TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT ''
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// GetConfig returns all rate limit configurations.
func (s *Store) GetConfig(ctx context.Context) ([]RateLimitConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tool_name, max_calls, window_seconds, updated_at
		FROM rate_limit_config
		ORDER BY tool_name
	`)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	defer rows.Close()

	var configs []RateLimitConfig
	for rows.Next() {
		var c RateLimitConfig
		if err := rows.Scan(&c.ToolName, &c.MaxCalls, &c.WindowSeconds, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if configs == nil {
		configs = []RateLimitConfig{}
	}
	return configs, nil
}

// SetConfig creates or updates a rate limit configuration for a tool.
func (s *Store) SetConfig(ctx context.Context, toolName string, maxCalls, windowSeconds int) (*RateLimitConfig, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO rate_limit_config (tool_name, max_calls, window_seconds, updated_at)
		VALUES (?, ?, ?, ?)
	`, toolName, maxCalls, windowSeconds, now)
	if err != nil {
		return nil, fmt.Errorf("set config: %w", err)
	}
	return &RateLimitConfig{
		ToolName:      toolName,
		MaxCalls:      maxCalls,
		WindowSeconds: windowSeconds,
		UpdatedAt:     now,
	}, nil
}

// GetHits returns rate limit hits, optionally filtered to today only.
// Results are ordered by timestamp DESC.
func (s *Store) GetHits(ctx context.Context, period string) ([]RateLimitHit, error) {
	query := `SELECT id, tool_name, timestamp, source FROM rate_limit_hits`
	args := []any{}

	if period == "today" {
		today := time.Now().UTC().Format("2006-01-02")
		query += ` WHERE timestamp >= ?`
		args = append(args, today)
	}
	query += ` ORDER BY timestamp DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get hits: %w", err)
	}
	defer rows.Close()

	var hits []RateLimitHit
	for rows.Next() {
		var h RateLimitHit
		if err := rows.Scan(&h.ID, &h.ToolName, &h.Timestamp, &h.Source); err != nil {
			return nil, fmt.Errorf("scan hit: %w", err)
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if hits == nil {
		hits = []RateLimitHit{}
	}
	return hits, nil
}
