package autotrigger

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages auto-negotiation triggers in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates an autotrigger Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate autotrigger: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS negotiation_triggers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		condition TEXT NOT NULL,
		action TEXT NOT NULL,
		vendor TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS trigger_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trigger_id INTEGER NOT NULL,
		fired_at TEXT NOT NULL,
		outcome TEXT NOT NULL DEFAULT ''
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetTrigger saves a new trigger and returns it with the generated ID and timestamp.
func (s *Store) SetTrigger(ctx context.Context, condition, action, vendor string) (*Trigger, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO negotiation_triggers (condition, action, vendor, enabled, created_at)
		VALUES (?, ?, ?, 1, ?)
	`, condition, action, vendor, now)
	if err != nil {
		return nil, fmt.Errorf("set trigger: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("set trigger last insert id: %w", err)
	}
	return &Trigger{
		ID:        int(id),
		Condition: condition,
		Action:    action,
		Vendor:    vendor,
		Enabled:   true,
		CreatedAt: now,
	}, nil
}

// ListTriggers returns all triggers ordered by created_at DESC.
func (s *Store) ListTriggers(ctx context.Context) ([]Trigger, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, condition, action, vendor, enabled, created_at
		FROM negotiation_triggers
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	defer rows.Close()

	var triggers []Trigger
	for rows.Next() {
		var t Trigger
		var enabled int
		if err := rows.Scan(&t.ID, &t.Condition, &t.Action, &t.Vendor, &enabled, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan trigger: %w", err)
		}
		t.Enabled = enabled != 0
		triggers = append(triggers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if triggers == nil {
		triggers = []Trigger{}
	}
	return triggers, nil
}

// GetTriggerLog returns the most recent 20 trigger log entries ordered by fired_at DESC.
func (s *Store) GetTriggerLog(ctx context.Context) ([]TriggerLogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trigger_id, fired_at, outcome
		FROM trigger_log
		ORDER BY fired_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("get trigger log: %w", err)
	}
	defer rows.Close()

	var entries []TriggerLogEntry
	for rows.Next() {
		var e TriggerLogEntry
		if err := rows.Scan(&e.ID, &e.TriggerID, &e.FiredAt, &e.Outcome); err != nil {
			return nil, fmt.Errorf("scan trigger log: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []TriggerLogEntry{}
	}
	return entries, nil
}
