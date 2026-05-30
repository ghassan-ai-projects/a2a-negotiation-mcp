package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides webhook subscription data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate webhooks: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS webhook_subscriptions (
		id TEXT PRIMARY KEY,
		url TEXT,
		events TEXT,
		secret TEXT,
		status TEXT DEFAULT 'active',
		created_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_webhook_status ON webhook_subscriptions(status);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Create inserts a new webhook subscription.
func (s *Store) Create(ctx context.Context, sub *Subscription) error {
	sub.ID = uuid.New().String()
	sub.Status = "active"
	sub.CreatedAt = time.Now().UTC()

	eventsJSON, err := json.Marshal(sub.Events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO webhook_subscriptions (id, url, events, secret, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.URL, string(eventsJSON), sub.Secret, sub.Status, sub.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}
	return nil
}

// Get retrieves a subscription by ID.
func (s *Store) Get(ctx context.Context, id string) (*Subscription, error) {
	var sub Subscription
	var eventsJSON, createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, events, secret, status, created_at
		FROM webhook_subscriptions WHERE id = ?
	`, id).Scan(&sub.ID, &sub.URL, &eventsJSON, &sub.Secret, &sub.Status, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get webhook: %w", err)
	}

	if err := json.Unmarshal([]byte(eventsJSON), &sub.Events); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}
	sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &sub, nil
}

// List returns all subscriptions, optionally filtered by status.
func (s *Store) List(ctx context.Context, status string) ([]Subscription, error) {
	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, url, events, secret, status, created_at
			FROM webhook_subscriptions WHERE status = ? ORDER BY created_at DESC
		`, status)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, url, events, secret, status, created_at
			FROM webhook_subscriptions ORDER BY created_at DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var eventsJSON, createdAt string
		if err := rows.Scan(&sub.ID, &sub.URL, &eventsJSON, &sub.Secret, &sub.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		if err := json.Unmarshal([]byte(eventsJSON), &sub.Events); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// GetByEvent returns all active subscriptions that match a given event type.
func (s *Store) GetByEvent(ctx context.Context, eventType string) ([]Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, events, secret, status, created_at
		FROM webhook_subscriptions WHERE status = 'active' ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("get webhooks by event: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		var eventsJSON, createdAt string
		if err := rows.Scan(&sub.ID, &sub.URL, &eventsJSON, &sub.Secret, &sub.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		if err := json.Unmarshal([]byte(eventsJSON), &sub.Events); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		for _, e := range sub.Events {
			if e == eventType || e == "*" {
				subs = append(subs, sub)
				break
			}
		}
	}
	return subs, rows.Err()
}

// Disable sets the subscription status to "disabled" (soft delete).
func (s *Store) Disable(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE webhook_subscriptions SET status = 'disabled' WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("disable webhook: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("webhook not found: %s", id)
	}
	return nil
}
