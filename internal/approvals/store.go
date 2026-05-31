package approvals

import (
	"context"
	"database/sql"
	"fmt"
)

// Store persists approval requests.
type Store struct {
	db *sql.DB
}

// NewStore creates a new approvals store sharing the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("approvals migrate: %w", err)
	}
	return s, nil
}

// DB exposes the underlying DB for sharing.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS approvals (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			reason TEXT,
			threshold REAL DEFAULT 0,
			status TEXT DEFAULT 'pending',
			created_at TEXT,
			resolved_at TEXT
		)
	`)
	return err
}

// Create inserts a new approval request.
func (s *Store) Create(ctx context.Context, a *Approval) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO approvals (id, session_id, reason, threshold, status, created_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.SessionID, a.Reason, a.Threshold, a.Status, a.CreatedAt, a.ResolvedAt,
	)
	return err
}

// Get returns a single approval by ID.
func (s *Store) Get(ctx context.Context, id string) (*Approval, error) {
	var a Approval
	err := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, reason, threshold, status, created_at, resolved_at FROM approvals WHERE id = ?`, id,
	).Scan(&a.ID, &a.SessionID, &a.Reason, &a.Threshold, &a.Status, &a.CreatedAt, &a.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateStatus changes the status and sets resolved_at for an approval.
func (s *Store) UpdateStatus(ctx context.Context, id string, status ApprovalStatus, resolvedAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE approvals SET status = ?, resolved_at = ? WHERE id = ?`,
		status, resolvedAt, id,
	)
	return err
}

// ListPending returns all approvals with status 'pending'.
func (s *Store) ListPending(ctx context.Context) ([]Approval, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, reason, threshold, status, created_at, resolved_at FROM approvals WHERE status = 'pending' ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Approval
	for rows.Next() {
		var a Approval
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Reason, &a.Threshold, &a.Status, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
