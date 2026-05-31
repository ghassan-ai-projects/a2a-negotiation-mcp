package notes

import (
	"context"
	"database/sql"
	"fmt"
)

// Store persists negotiation notes.
type Store struct {
	db *sql.DB
}

// NewStore creates a new notes store sharing the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("notes migrate: %w", err)
	}
	return s, nil
}

// DB exposes the underlying DB for sharing.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS negotiation_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_negotiation_notes_session_id ON negotiation_notes(session_id);
	`)
	return err
}

// Add inserts a new note.
func (s *Store) Add(ctx context.Context, note *NegotiationNote) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO negotiation_notes (session_id, content, created_at) VALUES (?, ?, ?)`,
		note.SessionID, note.Content, note.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	note.ID = id
	return nil
}

// ListBySession returns all notes for a given session.
func (s *Store) ListBySession(ctx context.Context, sessionID string) ([]NegotiationNote, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, content, created_at FROM negotiation_notes WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NegotiationNote
	for rows.Next() {
		var n NegotiationNote
		if err := rows.Scan(&n.ID, &n.SessionID, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// Delete removes a note by ID.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM negotiation_notes WHERE id = ?`, id)
	return err
}
