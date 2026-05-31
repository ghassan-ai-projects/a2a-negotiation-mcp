package contractclauses

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages contract clauses in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a ContractClausesStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate contractclauses: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contract_clauses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// ListClauses returns contract clauses, optionally filtered by category, ordered by title.
func (s *Store) ListClauses(ctx context.Context, category string) ([]Clause, error) {
	query := `SELECT id, category, title, content, description, created_at FROM contract_clauses`
	args := []any{}
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY title`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list clauses: %w", err)
	}
	defer rows.Close()

	var clauses []Clause
	for rows.Next() {
		var c Clause
		if err := rows.Scan(&c.ID, &c.Category, &c.Title, &c.Content, &c.Description, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan clause: %w", err)
		}
		clauses = append(clauses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if clauses == nil {
		clauses = []Clause{}
	}
	return clauses, nil
}

// GetClause retrieves a single contract clause by ID.
func (s *Store) GetClause(ctx context.Context, id int) (*Clause, error) {
	var c Clause
	err := s.db.QueryRowContext(ctx, `
		SELECT id, category, title, content, description, created_at
		FROM contract_clauses WHERE id = ?
	`, id).Scan(&c.ID, &c.Category, &c.Title, &c.Content, &c.Description, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("clause not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get clause: %w", err)
	}
	return &c, nil
}

// SearchClauses searches contract clauses by query across title, content, and description.
func (s *Store) SearchClauses(ctx context.Context, query string) ([]Clause, error) {
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, category, title, content, description, created_at
		FROM contract_clauses
		WHERE title LIKE ? OR content LIKE ? OR description LIKE ?
		ORDER BY title
	`, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("search clauses: %w", err)
	}
	defer rows.Close()

	var clauses []Clause
	for rows.Next() {
		var c Clause
		if err := rows.Scan(&c.ID, &c.Category, &c.Title, &c.Content, &c.Description, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan clause: %w", err)
		}
		clauses = append(clauses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if clauses == nil {
		clauses = []Clause{}
	}
	return clauses, nil
}

// AddClause adds a new contract clause and returns it with the generated ID and timestamp.
func (s *Store) AddClause(ctx context.Context, category, title, content, description string) (*Clause, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO contract_clauses (category, title, content, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, category, title, content, description, now)
	if err != nil {
		return nil, fmt.Errorf("add clause: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("add clause last insert id: %w", err)
	}
	return &Clause{
		ID:          int(id),
		Category:    category,
		Title:       title,
		Content:     content,
		Description: description,
		CreatedAt:   now,
	}, nil
}
