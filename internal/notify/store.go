package notify

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages notification preferences and logs backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a notify Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate notify: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notification_preferences (
		user_id TEXT NOT NULL,
		channel TEXT NOT NULL DEFAULT 'webhook',
		enabled_types TEXT NOT NULL DEFAULT '[]',
		digest_frequency TEXT NOT NULL DEFAULT 'never',
		webhook_url TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (user_id, channel)
	);

	CREATE TABLE IF NOT EXISTS notification_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		channel TEXT NOT NULL,
		message TEXT NOT NULL,
		priority TEXT NOT NULL DEFAULT 'normal',
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetPreferences upserts notification preferences for a user and channel.
func (s *Store) SetPreferences(ctx context.Context, prefs *NotificationPreferences) error {
	enabledJSON := "[]"
	if len(prefs.EnabledTypes) > 0 {
		enabledJSON = `["` + prefs.EnabledTypes[0] + `"]`
		for _, t := range prefs.EnabledTypes[1:] {
			enabledJSON += `,"` + t + `"`
		}
		enabledJSON = `["` + prefs.EnabledTypes[0] + `"`
		for _, t := range prefs.EnabledTypes[1:] {
			enabledJSON += `,"` + t + `"`
		}
		enabledJSON += `]`
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_preferences (user_id, channel, enabled_types, digest_frequency, webhook_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, channel) DO UPDATE SET
			enabled_types=excluded.enabled_types,
			digest_frequency=excluded.digest_frequency,
			webhook_url=excluded.webhook_url,
			updated_at=excluded.updated_at
	`, prefs.UserID, prefs.Channel, enabledJSON, prefs.DigestFreq, prefs.WebhookURL, now, now)
	if err != nil {
		return fmt.Errorf("set preferences: %w", err)
	}
	return nil
}

// GetPreferences retrieves notification preferences for a user and channel.
func (s *Store) GetPreferences(ctx context.Context, userID, channel string) (*NotificationPreferences, error) {
	var prefs NotificationPreferences
	var enabledJSON, createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, channel, enabled_types, digest_frequency, webhook_url, created_at, updated_at
		FROM notification_preferences WHERE user_id = ? AND channel = ?
	`, userID, channel).Scan(&prefs.UserID, &prefs.Channel, &enabledJSON, &prefs.DigestFreq, &prefs.WebhookURL, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	_ = createdAt
	_ = updatedAt

	// Parse enabled_types JSON
	var types []string
	if enabledJSON != "" && enabledJSON != "[]" {
		// Simple parse: strip brackets and split by comma
		inner := enabledJSON[1 : len(enabledJSON)-1]
		if inner != "" {
			parts := splitJSONArray(inner)
			types = parts
		}
	}
	prefs.EnabledTypes = types
	return &prefs, nil
}

// splitJSONArray splits a JSON array string (without brackets) into individual quoted strings.
func splitJSONArray(inner string) []string {
	var result []string
	i := 0
	for i < len(inner) {
		// Skip whitespace and commas
		for i < len(inner) && (inner[i] == ' ' || inner[i] == ',') {
			i++
		}
		if i >= len(inner) {
			break
		}
		// Expect opening quote
		if inner[i] == '"' {
			i++ // skip opening quote
			start := i
			for i < len(inner) && inner[i] != '"' {
				i++
			}
			if i > start {
				result = append(result, inner[start:i])
			}
			if i < len(inner) {
				i++ // skip closing quote
			}
		} else {
			i++
		}
	}
	return result
}

// LogNotification inserts a record into the notification_log.
func (s *Store) LogNotification(ctx context.Context, n *Notification) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_log (user_id, type, channel, message, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, n.UserID, n.Type, n.Channel, n.Message, n.Priority, n.Status, n.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("log notification: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
