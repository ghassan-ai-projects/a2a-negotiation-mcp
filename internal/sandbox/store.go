package sandbox

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages sandbox execution history in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a sandbox Store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate sandbox: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sandbox_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tool_name TEXT NOT NULL,
		params TEXT NOT NULL,
		result TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'executed',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// RecordExecution saves a sandbox execution and returns it with the generated ID and timestamp.
func (s *Store) RecordExecution(ctx context.Context, toolName, params, result string) (*SandboxExecution, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO sandbox_history (tool_name, params, result, status, created_at)
		VALUES (?, ?, ?, 'executed', ?)
	`, toolName, params, result, now)
	if err != nil {
		return nil, fmt.Errorf("record execution: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("record execution last insert id: %w", err)
	}
	return &SandboxExecution{
		ID:        int(id),
		ToolName:  toolName,
		Params:    params,
		Result:    result,
		Status:    "executed",
		CreatedAt: now,
	}, nil
}

// GetHistory returns recent sandbox executions, ordered by created_at DESC, limited to 20.
func (s *Store) GetHistory(ctx context.Context) ([]SandboxExecution, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tool_name, params, result, status, created_at
		FROM sandbox_history
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer rows.Close()

	var executions []SandboxExecution
	for rows.Next() {
		var e SandboxExecution
		if err := rows.Scan(&e.ID, &e.ToolName, &e.Params, &e.Result, &e.Status, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan execution: %w", err)
		}
		executions = append(executions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if executions == nil {
		executions = []SandboxExecution{}
	}
	return executions, nil
}

// ResetHistory deletes all sandbox history.
func (s *Store) ResetHistory(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sandbox_history`)
	if err != nil {
		return fmt.Errorf("reset history: %w", err)
	}
	return nil
}

// DB returns the underlying *sql.DB for sharing.
func (s *Store) DB() *sql.DB {
	return s.db
}
