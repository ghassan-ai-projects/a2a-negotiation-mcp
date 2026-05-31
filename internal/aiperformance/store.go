package aiperformance

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages AI agent performance logs in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates an AIPerformanceStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate aiperformance: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS ai_performance_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		latency_ms INTEGER NOT NULL DEFAULT 0,
		tokens_used INTEGER NOT NULL DEFAULT 0,
		success INTEGER NOT NULL DEFAULT 1,
		negotiation_type TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// LogCall logs a new AI agent performance call and returns the record with generated ID and timestamp.
func (s *Store) LogCall(ctx context.Context, model, toolName string, latencyMs, tokensUsed int, success bool, negotiationType string) (*AIPerformanceLog, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	successInt := 0
	if success {
		successInt = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_performance_log (model, tool_name, latency_ms, tokens_used, success, negotiation_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, model, toolName, latencyMs, tokensUsed, successInt, negotiationType, now)
	if err != nil {
		return nil, fmt.Errorf("log call: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("log call last insert id: %w", err)
	}
	return &AIPerformanceLog{
		ID:              int(id),
		Model:           model,
		ToolName:        toolName,
		LatencyMs:       latencyMs,
		TokensUsed:      tokensUsed,
		Success:         success,
		NegotiationType: negotiationType,
		CreatedAt:       now,
	}, nil
}

// GetSummary returns aggregated performance metrics grouped by model.
func (s *Store) GetSummary(ctx context.Context) ([]ProviderSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			model,
			count(*) AS total_calls,
			CAST(sum(success) AS REAL) / count(*) AS success_rate,
			avg(latency_ms) AS avg_latency_ms,
			sum(tokens_used) AS total_tokens
		FROM ai_performance_log
		GROUP BY model
		ORDER BY model
	`)
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	defer rows.Close()

	var summaries []ProviderSummary
	for rows.Next() {
		var s ProviderSummary
		if err := rows.Scan(&s.Model, &s.TotalCalls, &s.SuccessRate, &s.AvgLatencyMs, &s.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if summaries == nil {
		summaries = []ProviderSummary{}
	}
	return summaries, nil
}

// GetCalls returns the most recent performance calls for a given model, limited to limit rows.
func (s *Store) GetCalls(ctx context.Context, model string, limit int) ([]AIPerformanceLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, model, tool_name, latency_ms, tokens_used, success, negotiation_type, created_at
		FROM ai_performance_log
		WHERE model = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, model, limit)
	if err != nil {
		return nil, fmt.Errorf("get calls: %w", err)
	}
	defer rows.Close()

	var logs []AIPerformanceLog
	for rows.Next() {
		var l AIPerformanceLog
		var successInt int
		if err := rows.Scan(&l.ID, &l.Model, &l.ToolName, &l.LatencyMs, &l.TokensUsed, &successInt, &l.NegotiationType, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan call: %w", err)
		}
		l.Success = successInt == 1
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []AIPerformanceLog{}
	}
	return logs, nil
}

// DB returns the underlying database connection. Used for sharing the connection in tests.
func (s *Store) DB() *sql.DB {
	return s.db
}
