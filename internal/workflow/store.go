package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages automation workflows in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a WorkflowStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate workflow: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS automation_workflows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		steps_json TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS workflow_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL,
		run_at TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		result TEXT NOT NULL DEFAULT ''
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateWorkflow creates a new automation workflow and returns it with the generated ID and timestamp.
func (s *Store) CreateWorkflow(ctx context.Context, name, stepsJSON string) (*Workflow, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO automation_workflows (name, steps_json, created_at)
		VALUES (?, ?, ?)
	`, name, stepsJSON, now)
	if err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create workflow last insert id: %w", err)
	}
	return &Workflow{
		ID:        int(id),
		Name:      name,
		StepsJSON: stepsJSON,
		CreatedAt: now,
	}, nil
}

// ListWorkflows returns all automation workflows ordered by created_at DESC.
func (s *Store) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, steps_json, created_at
		FROM automation_workflows
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []Workflow
	for rows.Next() {
		var w Workflow
		if err := rows.Scan(&w.ID, &w.Name, &w.StepsJSON, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		workflows = append(workflows, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if workflows == nil {
		workflows = []Workflow{}
	}
	return workflows, nil
}

// RunWorkflow simulates running a workflow by inserting an execution log entry and returning a result.
func (s *Store) RunWorkflow(ctx context.Context, id int, params string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result := fmt.Sprintf("Workflow %d executed with params: %s", id, params)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workflow_log (workflow_id, run_at, status, result)
		VALUES (?, ?, 'executed', ?)
	`, id, now, result)
	if err != nil {
		return "", fmt.Errorf("run workflow: %w", err)
	}

	return result, nil
}

// GetWorkflowLog returns the execution log for a given workflow, ordered by run_at DESC.
func (s *Store) GetWorkflowLog(ctx context.Context, workflowID int) ([]WorkflowLogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workflow_id, run_at, status, result
		FROM workflow_log
		WHERE workflow_id = ?
		ORDER BY run_at DESC
	`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("get workflow log: %w", err)
	}
	defer rows.Close()

	var entries []WorkflowLogEntry
	for rows.Next() {
		var e WorkflowLogEntry
		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.RunAt, &e.Status, &e.Result); err != nil {
			return nil, fmt.Errorf("scan workflow log: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []WorkflowLogEntry{}
	}
	return entries, nil
}
