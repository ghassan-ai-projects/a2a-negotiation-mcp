package prompts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store manages prompt templates in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a PromptStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate prompts: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS prompt_templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SavePrompt saves a new prompt template and returns it with the generated ID and timestamp.
func (s *Store) SavePrompt(ctx context.Context, name, content, tags string) (*PromptTemplate, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO prompt_templates (name, content, tags, created_at)
		VALUES (?, ?, ?, ?)
	`, name, content, tags, now)
	if err != nil {
		return nil, fmt.Errorf("save prompt: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("save prompt last insert id: %w", err)
	}
	return &PromptTemplate{
		ID:        int(id),
		Name:      name,
		Content:   content,
		Tags:      tags,
		CreatedAt: now,
	}, nil
}

// ListPrompts returns all prompt templates, optionally filtered by tag, ordered by created_at DESC.
func (s *Store) ListPrompts(ctx context.Context, tag string) ([]PromptTemplate, error) {
	query := `SELECT id, name, content, tags, created_at FROM prompt_templates`
	args := []any{}
	if tag != "" {
		query += ` WHERE tags LIKE '%' || ? || '%'`
		args = append(args, tag)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()

	var prompts []PromptTemplate
	for rows.Next() {
		var p PromptTemplate
		if err := rows.Scan(&p.ID, &p.Name, &p.Content, &p.Tags, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt: %w", err)
		}
		prompts = append(prompts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if prompts == nil {
		prompts = []PromptTemplate{}
	}
	return prompts, nil
}

// GetPrompt retrieves a single prompt template by ID.
func (s *Store) GetPrompt(ctx context.Context, id int) (*PromptTemplate, error) {
	var p PromptTemplate
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, content, tags, created_at
		FROM prompt_templates WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Content, &p.Tags, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get prompt: %w", err)
	}
	return &p, nil
}

// RenderPrompt gets a prompt by ID and replaces {{key}} placeholders with
// values from the variables map, returning the rendered content.
func (s *Store) RenderPrompt(ctx context.Context, id int, variables map[string]string) (string, error) {
	p, err := s.GetPrompt(ctx, id)
	if err != nil {
		return "", err
	}

	rendered := p.Content
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}
	return rendered, nil
}
