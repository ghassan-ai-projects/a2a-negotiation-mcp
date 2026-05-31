package webhooklog

import (
	"context"
	"database/sql"
	"fmt"
)

// Store manages webhook event logs in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a WebhookLogStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate webhooklog: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS webhook_event_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// ListEvents returns webhook events, optionally filtered by status, ordered by created_at DESC.
func (s *Store) ListEvents(ctx context.Context, status string, limit int) ([]WebhookEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, event_type, payload, status, attempts, created_at FROM webhook_event_log`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []WebhookEvent
	for rows.Next() {
		var e WebhookEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.Status, &e.Attempts, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if events == nil {
		events = []WebhookEvent{}
	}
	return events, nil
}

// GetEvent retrieves a single webhook event by ID.
func (s *Store) GetEvent(ctx context.Context, id int) (*WebhookEvent, error) {
	var e WebhookEvent
	err := s.db.QueryRowContext(ctx, `
		SELECT id, event_type, payload, status, attempts, created_at
		FROM webhook_event_log WHERE id = ?
	`, id).Scan(&e.ID, &e.EventType, &e.Payload, &e.Status, &e.Attempts, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return &e, nil
}

// ReplayEvent increments the attempt count and updates the status to 'replayed'.
func (s *Store) ReplayEvent(ctx context.Context, id int) (*WebhookEvent, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE webhook_event_log
		SET attempts = attempts + 1, status = 'replayed'
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("replay event: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("replay event rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("event not found: id=%d", id)
	}
	return s.GetEvent(ctx, id)
}

// GetStats returns aggregated webhook event statistics.
func (s *Store) GetStats(ctx context.Context) (*WebhookStats, error) {
	var stats WebhookStats
	stats.StatusBreakdown = make(map[string]int)

	row := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_events,
			COALESCE(AVG(CASE WHEN status = 'success' THEN 1.0 ELSE 0.0 END), 0.0) AS success_rate,
			COALESCE(AVG(CAST(attempts AS REAL)), 0.0) AS avg_attempts
		FROM webhook_event_log
	`)
	if err := row.Scan(&stats.TotalEvents, &stats.SuccessRate, &stats.AvgAttempts); err != nil {
		return nil, fmt.Errorf("get stats aggregate: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) AS cnt
		FROM webhook_event_log
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("get stats breakdown: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan stats breakdown: %w", err)
		}
		stats.StatusBreakdown[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &stats, nil
}
