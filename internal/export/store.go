package export

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages data export persistence backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates an export Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate export: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS data_exports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL DEFAULT '',
		format TEXT NOT NULL,
		export_type TEXT NOT NULL,
		record_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SaveExport persists an export metadata record and returns the assigned ID.
func (s *Store) SaveExport(ctx context.Context, userID, format, exportType string, recordCount int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO data_exports (user_id, format, export_type, record_count, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, format, exportType, recordCount, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("save export: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
