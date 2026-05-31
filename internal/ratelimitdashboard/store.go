package ratelimitdashboard

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages API usage log persistence backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate ratelimitdashboard: %w", err)
	}
	return s, nil
}

// NewInMemoryStore creates an in-memory Store for tests.
func NewInMemoryStore() (*Store, error) {
	db, err := sql.Open("sqlite", ":memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate ratelimitdashboard: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS api_usage_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key_id TEXT NOT NULL,
		endpoint TEXT NOT NULL,
		timestamp TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_api_usage_timestamp ON api_usage_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_api_usage_api_key ON api_usage_log(api_key_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// LogRequest records an API usage event.
func (s *Store) LogRequest(ctx context.Context, apiKeyID, endpoint string) (*APIUsageEntry, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO api_usage_log (api_key_id, endpoint, timestamp)
		VALUES (?, ?, ?)
	`, apiKeyID, endpoint, now)
	if err != nil {
		return nil, fmt.Errorf("log api request: %w", err)
	}
	id, _ := result.LastInsertId()
	return &APIUsageEntry{
		ID:        id,
		APIKeyID:  apiKeyID,
		Endpoint:  endpoint,
		Timestamp: now,
	}, nil
}

// CountSince returns the number of API requests logged since the given time.
func (s *Store) CountSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM api_usage_log WHERE timestamp >= ?
	`, since.Format(time.RFC3339)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count since: %w", err)
	}
	return count, nil
}

// CountToday returns the number of API requests logged today (UTC).
func (s *Store) CountToday(ctx context.Context) (int, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	return s.CountSince(ctx, startOfDay)
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
