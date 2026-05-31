package auditlog

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store persists audit log entries.
type Store struct {
	db *sql.DB
}

// NewStore creates an audit log store sharing the given DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("auditlog migrate: %w", err)
	}
	return s, nil
}

// DB exposes the underlying DB for sharing.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			user_id TEXT DEFAULT 'system',
			details TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)
	`)
	return err
}

// Log inserts a new audit entry.
func (s *Store) Log(ctx context.Context, action, userID, details string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (action, user_id, details) VALUES (?, ?, ?)`,
		action, userID, details,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Query returns audit entries matching optional filters.
func (s *Store) Query(ctx context.Context, action, userID string, limit int, since string) ([]AuditEntry, error) {
	query := `SELECT id, action, user_id, details, created_at FROM audit_log WHERE 1=1`
	args := []any{}

	if action != "" {
		query += ` AND action = ?`
		args = append(args, action)
	}
	if userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	if since != "" {
		query += ` AND created_at >= ?`
		args = append(args, since)
	}

	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var createdAt string
		if err := rows.Scan(&e.ID, &e.Action, &e.UserID, &e.Details, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		if e.CreatedAt.IsZero() {
			e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// Summary returns aggregated audit stats.
func (s *Store) Summary(ctx context.Context) (*AuditSummary, error) {
	summary := &AuditSummary{
		ByAction: make(map[string]int64),
		ByDay:    make(map[string]int64),
	}

	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&summary.TotalActions); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT action, COUNT(*) FROM audit_log GROUP BY action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var count int64
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		summary.ByAction[action] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows2, err := s.db.QueryContext(ctx, `SELECT DATE(created_at), COUNT(*) FROM audit_log GROUP BY DATE(created_at) ORDER BY DATE(created_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var day string
		var count int64
		if err := rows2.Scan(&day, &count); err != nil {
			return nil, err
		}
		summary.ByDay[day] = count
	}
	return summary, rows2.Err()
}
