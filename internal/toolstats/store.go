package toolstats

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store provides tool usage statistics persistence.
type Store struct {
	db *sql.DB
}

// NewStore creates a new toolstats store and ensures the schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("toolstats migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tool_usage_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tool_name TEXT NOT NULL,
			called_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_tool_usage_tool_name ON tool_usage_stats(tool_name);
	`)
	return err
}

// LogCall records a tool invocation.
func (s *Store) LogCall(ctx context.Context, toolName string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tool_usage_stats (tool_name, called_at) VALUES (?, datetime('now'))`,
		toolName,
	)
	return err
}

// GetTopTools returns the most frequently called tools since the given time.
func (s *Store) GetTopTools(ctx context.Context, since time.Time, limit int) ([]ToolUsage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tool_name, COUNT(*) AS cnt
		FROM tool_usage_stats
		WHERE called_at >= ?
		GROUP BY tool_name
		ORDER BY cnt DESC
		LIMIT ?
	`, since.Format(time.RFC3339), limit)
	if err != nil {
		return nil, fmt.Errorf("get top tools: %w", err)
	}
	defer rows.Close()

	var result []ToolUsage
	for rows.Next() {
		var tu ToolUsage
		if err := rows.Scan(&tu.ToolName, &tu.CallCount); err != nil {
			return nil, fmt.Errorf("scan tool usage: %w", err)
		}
		result = append(result, tu)
	}
	return result, rows.Err()
}

// CountByTool returns the call count for each tool since the given time.
func (s *Store) CountByTool(ctx context.Context, since time.Time) ([]ToolUsage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tool_name, COUNT(*) AS cnt
		FROM tool_usage_stats
		WHERE called_at >= ?
		GROUP BY tool_name
		ORDER BY cnt DESC
	`, since.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("count by tool: %w", err)
	}
	defer rows.Close()

	var result []ToolUsage
	for rows.Next() {
		var tu ToolUsage
		if err := rows.Scan(&tu.ToolName, &tu.CallCount); err != nil {
			return nil, fmt.Errorf("scan tool usage: %w", err)
		}
		result = append(result, tu)
	}
	return result, rows.Err()
}

// TotalCalls returns the total number of tool calls since the given time.
func (s *Store) TotalCalls(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tool_usage_stats WHERE called_at >= ?
	`, since.Format(time.RFC3339)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("total calls: %w", err)
	}
	return count, nil
}
