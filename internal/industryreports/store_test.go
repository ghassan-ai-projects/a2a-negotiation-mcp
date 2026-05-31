package industryreports

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

func TestSaveAndGetReport(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.SaveReport(ctx, "Cloud Market Report 2026", "cloud", "Content about cloud market", "Gartner")
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	if saved.Title != "Cloud Market Report 2026" {
		t.Errorf("expected title 'Cloud Market Report 2026', got %s", saved.Title)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	got, err := s.GetReport(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if got.Title != saved.Title {
		t.Errorf("expected title %s, got %s", saved.Title, got.Title)
	}
	if got.Source != "Gartner" {
		t.Errorf("expected source Gartner, got %s", got.Source)
	}
}

func TestListReports_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	reports, err := s.ListReports(ctx, "")
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if reports == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(reports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(reports))
	}
}

func TestListReports_FilteredByCategory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved1, err := s.SaveReport(ctx, "AI Report", "ai", "AI content", "Source A")
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}
	saved2, err := s.SaveReport(ctx, "Cloud Report", "cloud", "Cloud content", "Source B")
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}
	saved3, err := s.SaveReport(ctx, "AI Trends", "ai", "More AI content", "Source C")
	if err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	aiReports, err := s.ListReports(ctx, "ai")
	if err != nil {
		t.Fatalf("ListReports(ai): %v", err)
	}
	if len(aiReports) != 2 {
		t.Errorf("expected 2 ai reports, got %d", len(aiReports))
	}

	// Verify both AI reports are present (order may vary within same second)
	titles := make(map[string]bool)
	for _, r := range aiReports {
		titles[r.Title] = true
	}
	if !titles["AI Report"] {
		t.Error("expected 'AI Report' in results")
	}
	if !titles["AI Trends"] {
		t.Error("expected 'AI Trends' in results")
	}

	// Verify non-AI report is not in the results
	for _, r := range aiReports {
		if r.ID == saved2.ID {
			t.Error("unexpected Cloud Report in ai results")
		}
	}

	// Verify IDs match
	if aiReports[0].ID != saved3.ID && aiReports[1].ID != saved3.ID {
		t.Error("expected saved3 (AI Trends) to appear in ai results")
	}
	if aiReports[0].ID != saved1.ID && aiReports[1].ID != saved1.ID {
		t.Error("expected saved1 (AI Report) to appear in ai results")
	}
}

func TestGetReport_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetReport(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent report")
	}
}

func TestListReportsMultiple(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	titles := []string{"Report A", "Report B", "Report C"}
	for _, title := range titles {
		_, err := s.SaveReport(ctx, title, "general", "Content for "+title, "Source")
		if err != nil {
			t.Fatalf("SaveReport(%s): %v", title, err)
		}
	}

	reports, err := s.ListReports(ctx, "")
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(reports) != 3 {
		t.Errorf("expected 3 reports, got %d", len(reports))
	}
}
