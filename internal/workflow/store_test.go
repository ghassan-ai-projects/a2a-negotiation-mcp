package workflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestCreateWorkflow(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	steps := `[{"condition":"price < 100","action":"approve","params":""}]`
	saved, err := s.CreateWorkflow(ctx, "Auto Approve Low Price", steps)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	if saved.Name != "Auto Approve Low Price" {
		t.Errorf("expected name 'Auto Approve Low Price', got %s", saved.Name)
	}
	if saved.StepsJSON != steps {
		t.Errorf("expected steps_json %s, got %s", steps, saved.StepsJSON)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
}

func TestListWorkflows(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Initially empty
	workflows, err := s.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if workflows == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(workflows))
	}

	// Create two workflows
	_, err = s.CreateWorkflow(ctx, "Workflow A", `[{"action":"email"}]`)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	_, err = s.CreateWorkflow(ctx, "Workflow B", `[{"action":"slack"}]`)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	workflows, err = s.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if len(workflows) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(workflows))
	}
	// Verify both workflows are present
	names := make(map[string]bool)
	for _, w := range workflows {
		names[w.Name] = true
	}
	if !names["Workflow A"] {
		t.Error("expected 'Workflow A' in results")
	}
	if !names["Workflow B"] {
		t.Error("expected 'Workflow B' in results")
	}
}

func TestRunWorkflowAndGetLog(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateWorkflow(ctx, "Test Run", `[{"action":"test"}]`)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	// Run workflow
	result, err := s.RunWorkflow(ctx, saved.ID, `{"key":"value"}`)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	if result == "" {
		t.Errorf("expected non-empty result")
	}

	// Check log
	logs, err := s.GetWorkflowLog(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetWorkflowLog: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Status != "executed" {
		t.Errorf("expected status 'executed', got %s", logs[0].Status)
	}
	if logs[0].WorkflowID != saved.ID {
		t.Errorf("expected workflow_id %d, got %d", saved.ID, logs[0].WorkflowID)
	}
}

func TestGetWorkflowLog_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateWorkflow(ctx, "Empty Log", `[]`)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	logs, err := s.GetWorkflowLog(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetWorkflowLog: %v", err)
	}
	if logs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 log entries, got %d", len(logs))
	}
}

func TestRunWorkflowMultipleTimes(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateWorkflow(ctx, "Multi Run", `[{"action":"multi"}]`)
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := s.RunWorkflow(ctx, saved.ID, `{"run": `+fmt.Sprintf("%d", i)+`}`)
		if err != nil {
			t.Fatalf("RunWorkflow (run %d): %v", i, err)
		}
	}

	logs, err := s.GetWorkflowLog(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetWorkflowLog: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(logs))
	}
}
