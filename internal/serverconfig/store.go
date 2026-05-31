package serverconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Store manages server configuration entries in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a ServerConfigStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate serverconfig: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS server_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// GetConfig returns all server configuration entries.
func (s *Store) GetConfig(ctx context.Context) ([]ConfigEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value, updated_at FROM server_config ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	defer rows.Close()

	var entries []ConfigEntry
	for rows.Next() {
		var e ConfigEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan config entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []ConfigEntry{}
	}
	return entries, nil
}

// SetConfig sets a server configuration entry, inserting or replacing it.
func (s *Store) SetConfig(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO server_config (key, value, updated_at)
		VALUES (?, ?, ?)
	`, key, value, now)
	if err != nil {
		return fmt.Errorf("set config %q: %w", key, err)
	}
	return nil
}

// ExportConfig returns all server configuration entries as a JSON string.
func (s *Store) ExportConfig(ctx context.Context) (string, error) {
	entries, err := s.GetConfig(ctx)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("export config marshal: %w", err)
	}
	return string(b), nil
}

// ImportConfig imports server configuration entries from a JSON string.
// It returns the number of entries imported.
func (s *Store) ImportConfig(ctx context.Context, jsonData string) (int, error) {
	var entries []ConfigEntry
	if err := json.Unmarshal([]byte(jsonData), &entries); err != nil {
		return 0, fmt.Errorf("import config unmarshal: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, e := range entries {
		_, err := s.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO server_config (key, value, updated_at)
			VALUES (?, ?, ?)
		`, e.Key, e.Value, now)
		if err != nil {
			return 0, fmt.Errorf("import config entry %q: %w", e.Key, err)
		}
	}
	return len(entries), nil
}
