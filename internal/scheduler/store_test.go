package scheduler

import (
	"context"
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

func TestCreateAndListSchedule(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateSchedule(ctx, "Acme Corp", "aggressive", "0 */6 * * *")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	if saved.Vendor != "Acme Corp" {
		t.Errorf("expected vendor 'Acme Corp', got %s", saved.Vendor)
	}
	if saved.Strategy != "aggressive" {
		t.Errorf("expected strategy 'aggressive', got %s", saved.Strategy)
	}
	if saved.CronExpr != "0 */6 * * *" {
		t.Errorf("expected cron_expr '0 */6 * * *', got %s", saved.CronExpr)
	}
	if !saved.Enabled {
		t.Error("expected enabled to be true")
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	schedules, err := s.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}
	if schedules[0].ID != saved.ID {
		t.Errorf("expected ID %d, got %d", saved.ID, schedules[0].ID)
	}
}

func TestListSchedules_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	schedules, err := s.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if schedules == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules, got %d", len(schedules))
	}
}

func TestDeleteSchedule(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateSchedule(ctx, "Beta Inc", "balanced", "0 0 * * *")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	schedules, err := s.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule before delete, got %d", len(schedules))
	}

	if err := s.DeleteSchedule(ctx, saved.ID); err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}

	schedules, err = s.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("ListSchedules after delete: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(schedules))
	}
}

func TestGetScheduleResults(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateSchedule(ctx, "Gamma Ltd", "conservative", "0 12 * * 1")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	// No results yet — should return empty slice
	results, err := s.GetScheduleResults(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetScheduleResults: %v", err)
	}
	if results == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	// Insert a result directly for testing
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO schedule_results (schedule_id, run_at, status, summary)
		VALUES (?, '2026-05-31T10:00:00Z', 'completed', 'Negotiation completed successfully')
	`, saved.ID)
	if err != nil {
		t.Fatalf("insert result: %v", err)
	}

	results, err = s.GetScheduleResults(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetScheduleResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "completed" {
		t.Errorf("expected status 'completed', got %s", results[0].Status)
	}
	if results[0].Summary != "Negotiation completed successfully" {
		t.Errorf("expected summary 'Negotiation completed successfully', got %s", results[0].Summary)
	}
}
