package strategymarket

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

func TestPublishStrategy(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	strat, err := s.PublishStrategy(ctx, "Competitive Pricing", "A strategy for competitive pricing negotiations", `{"max_discount":0.15}`, "pricing")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}

	if strat.Name != "Competitive Pricing" {
		t.Errorf("expected name 'Competitive Pricing', got %s", strat.Name)
	}
	if strat.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if strat.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
	if strat.Rating != 0 {
		t.Errorf("expected initial rating 0, got %f", strat.Rating)
	}
	if strat.RatingCount != 0 {
		t.Errorf("expected initial rating_count 0, got %d", strat.RatingCount)
	}
}

func TestBrowseStrategies_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	strategies, err := s.BrowseStrategies(ctx, "", "")
	if err != nil {
		t.Fatalf("BrowseStrategies: %v", err)
	}
	if strategies == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(strategies) != 0 {
		t.Errorf("expected 0 strategies, got %d", len(strategies))
	}
}

func TestBrowseStrategies_FilteredByCategory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	s1, err := s.PublishStrategy(ctx, "Strategy A", "Desc A", "{}", "pricing")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}
	s2, err := s.PublishStrategy(ctx, "Strategy B", "Desc B", "{}", "legal")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}
	s3, err := s.PublishStrategy(ctx, "Strategy C", "Desc C", "{}", "pricing")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}

	pricingStrategies, err := s.BrowseStrategies(ctx, "pricing", "")
	if err != nil {
		t.Fatalf("BrowseStrategies(pricing): %v", err)
	}
	if len(pricingStrategies) != 2 {
		t.Errorf("expected 2 pricing strategies, got %d", len(pricingStrategies))
	}

	// Verify both pricing strategies are present
	names := make(map[string]bool)
	for _, st := range pricingStrategies {
		names[st.Name] = true
	}
	if !names["Strategy A"] {
		t.Error("expected 'Strategy A' in results")
	}
	if !names["Strategy C"] {
		t.Error("expected 'Strategy C' in results")
	}

	// Verify non-pricing strategy is not in results
	for _, st := range pricingStrategies {
		if st.ID == s2.ID {
			t.Error("unexpected legal Strategy B in pricing results")
		}
	}

	// Verify IDs match
	if pricingStrategies[0].ID != s3.ID && pricingStrategies[1].ID != s3.ID {
		t.Error("expected s3 (Strategy C) to appear in pricing results")
	}
	if pricingStrategies[0].ID != s1.ID && pricingStrategies[1].ID != s1.ID {
		t.Error("expected s1 (Strategy A) to appear in pricing results")
	}
}

func TestBrowseStrategies_SortedByRating(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	s1, err := s.PublishStrategy(ctx, "Low Rated", "Low", "{}", "test")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}
	s2, err := s.PublishStrategy(ctx, "High Rated", "High", "{}", "test")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}

	// Rate s2 higher
	_, err = s.RateStrategy(ctx, s2.ID, 5.0)
	if err != nil {
		t.Fatalf("RateStrategy: %v", err)
	}
	// Rate s1 lower
	_, err = s.RateStrategy(ctx, s1.ID, 1.0)
	if err != nil {
		t.Fatalf("RateStrategy: %v", err)
	}

	strategies, err := s.BrowseStrategies(ctx, "test", "rating")
	if err != nil {
		t.Fatalf("BrowseStrategies(test, rating): %v", err)
	}
	if len(strategies) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(strategies))
	}

	// Highest rated should come first
	if strategies[0].Name != "High Rated" {
		t.Errorf("expected first to be 'High Rated', got %s", strategies[0].Name)
	}
	if strategies[0].Rating <= strategies[1].Rating {
		t.Error("expected first strategy to have higher rating than second")
	}
}

func TestImportStrategy(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.PublishStrategy(ctx, "My Strategy", "A great strategy", `{"key":"value"}`, "general")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}

	imported, err := s.ImportStrategy(ctx, saved.ID)
	if err != nil {
		t.Fatalf("ImportStrategy: %v", err)
	}

	if imported.Name != saved.Name {
		t.Errorf("expected name %s, got %s", saved.Name, imported.Name)
	}
	if imported.Description != saved.Description {
		t.Errorf("expected description %s, got %s", saved.Description, imported.Description)
	}
	if imported.Config != saved.Config {
		t.Errorf("expected config %s, got %s", saved.Config, imported.Config)
	}
	if imported.Category != saved.Category {
		t.Errorf("expected category %s, got %s", saved.Category, imported.Category)
	}
}

func TestImportStrategy_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.ImportStrategy(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent strategy")
	}
}

func TestRateStrategy(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.PublishStrategy(ctx, "Rateable", "Test strategy for rating", "{}", "test")
	if err != nil {
		t.Fatalf("PublishStrategy: %v", err)
	}

	// First rating: 4.0
	rated, err := s.RateStrategy(ctx, saved.ID, 4.0)
	if err != nil {
		t.Fatalf("RateStrategy (first): %v", err)
	}
	if rated.Rating != 4.0 {
		t.Errorf("expected rating 4.0, got %f", rated.Rating)
	}
	if rated.RatingCount != 1 {
		t.Errorf("expected rating_count 1, got %d", rated.RatingCount)
	}

	// Second rating: 2.0 → average = (4*1 + 2)/2 = 3.0
	rated, err = s.RateStrategy(ctx, saved.ID, 2.0)
	if err != nil {
		t.Fatalf("RateStrategy (second): %v", err)
	}
	if rated.Rating != 3.0 {
		t.Errorf("expected rating 3.0, got %f", rated.Rating)
	}
	if rated.RatingCount != 2 {
		t.Errorf("expected rating_count 2, got %d", rated.RatingCount)
	}
}

func TestRateStrategy_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.RateStrategy(ctx, 999, 5.0)
	if err == nil {
		t.Fatal("expected error for non-existent strategy")
	}
}
