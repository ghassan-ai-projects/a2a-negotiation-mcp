package contracttemplates

import (
	"context"
	"database/sql"
	"fmt"
)

// Store manages contract templates in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate contracttemplates: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS contract_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'general',
			content TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	return err
}

// Create inserts a new contract template.
func (s *Store) Create(ctx context.Context, t *ContractTemplate) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO contract_templates (id, name, category, content, created_at) VALUES (?, ?, ?, ?, COALESCE(NULLIF(?, ''), datetime('now')))`,
		t.ID, t.Name, t.Category, t.Content, t.CreatedAt,
	)
	return err
}

// List returns all contract templates, optionally filtered by category.
func (s *Store) List(ctx context.Context, category string) ([]ContractTemplate, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, name, category, content, created_at FROM contract_templates WHERE category = ? ORDER BY name`,
			category,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, name, category, content, created_at FROM contract_templates ORDER BY name`,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ContractTemplate
	for rows.Next() {
		var t ContractTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Content, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// GetByCategory returns templates for a specific category.
func (s *Store) GetByCategory(ctx context.Context, category string) ([]ContractTemplate, error) {
	return s.List(ctx, category)
}

// Get returns a single template by ID.
func (s *Store) Get(ctx context.Context, id string) (*ContractTemplate, error) {
	var t ContractTemplate
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, category, content, created_at FROM contract_templates WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.Category, &t.Content, &t.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// Delete removes a contract template by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM contract_templates WHERE id = ?`, id)
	return err
}
