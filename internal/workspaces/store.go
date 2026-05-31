package workspaces

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store persists workspace data.
type Store struct {
	db *sql.DB
}

// NewStore creates a Workspace store sharing the given DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("workspaces migrate: %w", err)
	}
	return s, nil
}

// DB exposes the underlying DB for sharing.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	return err
}

// Create inserts a new workspace.
func (s *Store) Create(ctx context.Context, id, name, description string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workspaces (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, name, description, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return err
}

// List returns all workspaces ordered by name.
func (s *Store) List(ctx context.Context) ([]Workspace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM workspaces ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Workspace
	for rows.Next() {
		var w Workspace
		var createdAt, updatedAt string
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		w.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		w.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		result = append(result, w)
	}
	return result, rows.Err()
}

// Get retrieves a single workspace by ID.
func (s *Store) Get(ctx context.Context, id string) (*Workspace, error) {
	var w Workspace
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM workspaces WHERE id = ?`, id,
	).Scan(&w.ID, &w.Name, &w.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	w.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &w, nil
}

// Delete removes a workspace by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id)
	return err
}
