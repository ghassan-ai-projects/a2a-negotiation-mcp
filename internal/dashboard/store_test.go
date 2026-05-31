package dashboard

import (
	"context"
	"encoding/json"
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

func TestCreateWidget(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.CreateWidget(ctx, "price_chart", "Price Trends", `{"vendor":"Slack","sku":"Pro"}`)
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}

	if saved.Title != "Price Trends" {
		t.Errorf("expected title 'Price Trends', got %s", saved.Title)
	}
	if saved.WidgetType != "price_chart" {
		t.Errorf("expected widget_type 'price_chart', got %s", saved.WidgetType)
	}
	if saved.Config != `{"vendor":"Slack","sku":"Pro"}` {
		t.Errorf("expected config '{\"vendor\":\"Slack\",\"sku\":\"Pro\"}', got %s", saved.Config)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
}

func TestListWidgets_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	widgets, err := s.ListWidgets(ctx)
	if err != nil {
		t.Fatalf("ListWidgets: %v", err)
	}
	if widgets == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(widgets) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(widgets))
	}
}

func TestListWidgets_Multiple(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	w1, err := s.CreateWidget(ctx, "price_chart", "Chart A", "{}")
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}
	w2, err := s.CreateWidget(ctx, "metric", "Metric B", `{"key":"value"}`)
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}

	widgets, err := s.ListWidgets(ctx)
	if err != nil {
		t.Fatalf("ListWidgets: %v", err)
	}
	if len(widgets) != 2 {
		t.Errorf("expected 2 widgets, got %d", len(widgets))
	}

	// Both widgets should be present regardless of order
	ids := make(map[int]bool)
	for _, w := range widgets {
		ids[w.ID] = true
	}
	if !ids[w1.ID] {
		t.Error("expected widget w1 in list")
	}
	if !ids[w2.ID] {
		t.Error("expected widget w2 in list")
	}
}

func TestRenderDashboard_WithIDs(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	w1, err := s.CreateWidget(ctx, "price_chart", "A", "{}")
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}
	w2, err := s.CreateWidget(ctx, "metric", "B", "{}")
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}
	w3, err := s.CreateWidget(ctx, "table", "C", "{}")
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}

	dash, err := s.RenderDashboard(ctx, []int{w1.ID, w3.ID})
	if err != nil {
		t.Fatalf("RenderDashboard: %v", err)
	}
	if dash.Count != 2 {
		t.Errorf("expected count 2, got %d", dash.Count)
	}
	if len(dash.Widgets) != 2 {
		t.Errorf("expected 2 widgets, got %d", len(dash.Widgets))
	}

	ids := make(map[int]bool)
	for _, w := range dash.Widgets {
		ids[w.ID] = true
	}
	if !ids[w1.ID] {
		t.Error("expected widget 1 in dashboard")
	}
	if ids[w2.ID] {
		t.Error("did not expect widget 2 in dashboard")
	}
	if !ids[w3.ID] {
		t.Error("expected widget 3 in dashboard")
	}
}

func TestRenderDashboard_EmptyIDs(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	dash, err := s.RenderDashboard(ctx, []int{})
	if err != nil {
		t.Fatalf("RenderDashboard: %v", err)
	}
	if dash.Count != 0 {
		t.Errorf("expected count 0, got %d", dash.Count)
	}
	if len(dash.Widgets) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(dash.Widgets))
	}
}

func TestExportDashboard(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.CreateWidget(ctx, "price_chart", "Chart A", `{"vendor":"Slack"}`)
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}
	_, err = s.CreateWidget(ctx, "metric", "Metric B", `{"threshold":0.8}`)
	if err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}

	exported, err := s.ExportDashboard(ctx, "json")
	if err != nil {
		t.Fatalf("ExportDashboard: %v", err)
	}

	var parsed []Widget
	if err := json.Unmarshal([]byte(exported), &parsed); err != nil {
		t.Fatalf("ExportDashboard output is not valid JSON: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 widgets in export, got %d", len(parsed))
	}

	// Verify pretty-printed (contains newlines and indentation)
	if exported[0] != '[' {
		t.Errorf("expected JSON array, got %c", exported[0])
	}
	// Pretty-printed JSON has newlines after brackets
	if exported[1] != '\n' {
		t.Error("expected pretty-printed JSON with newlines for format='json'")
	}
}

func TestExportDashboard_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	exported, err := s.ExportDashboard(ctx, "")
	if err != nil {
		t.Fatalf("ExportDashboard: %v", err)
	}
	if exported != "[]" {
		t.Errorf("expected empty JSON array '[]', got %s", exported)
	}
}
