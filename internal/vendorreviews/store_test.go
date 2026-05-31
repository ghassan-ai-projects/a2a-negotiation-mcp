package vendorreviews

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

func TestAddReview(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	review, err := store.AddReview(ctx, "Slack", 4, "Great negotiation, got 15% discount")
	if err != nil {
		t.Fatalf("AddReview: %v", err)
	}

	if review.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if review.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", review.Vendor)
	}
	if review.Rating != 4 {
		t.Errorf("expected rating 4, got %d", review.Rating)
	}
	if review.Comment != "Great negotiation, got 15% discount" {
		t.Errorf("unexpected comment: %s", review.Comment)
	}
	if review.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestGetVendorReviews(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.AddReview(ctx, "Slack", 5, "Excellent deal")
	if err != nil {
		t.Fatalf("AddReview: %v", err)
	}
	_, err = store.AddReview(ctx, "Slack", 3, "Average experience")
	if err != nil {
		t.Fatalf("AddReview: %v", err)
	}

	reviews, err := store.GetVendorReviews(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetVendorReviews: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(reviews))
	}

	// Both ratings should be present (order may vary with identical timestamps)
	ratings := map[int]bool{}
	for _, r := range reviews {
		ratings[r.Rating] = true
	}
	if !ratings[5] {
		t.Error("expected rating 5 among reviews")
	}
	if !ratings[3] {
		t.Error("expected rating 3 among reviews")
	}
}

func TestGetVendorReviews_Empty(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	reviews, err := store.GetVendorReviews(ctx, "NonExistentVendor")
	if err != nil {
		t.Fatalf("GetVendorReviews: %v", err)
	}
	if len(reviews) != 0 {
		t.Errorf("expected 0 reviews, got %d", len(reviews))
	}
}

func TestGetTopVendors(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// Seed vendors into the pricing DB so the top vendors query can reference them
	vendors := []struct {
		name     string
		category string
	}{
		{"Slack", "Communication"},
		{"GitHub", "Developer Tools"},
	}

	for _, v := range vendors {
		_, err := store.db.ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
			v.name, v.category)
		if err != nil {
			t.Fatalf("seed vendor %s: %v", v.name, err)
		}
	}

	_, err := store.AddReview(ctx, "Slack", 5, "Great")
	if err != nil {
		t.Fatalf("AddReview Slack: %v", err)
	}
	_, err = store.AddReview(ctx, "Slack", 4, "Good")
	if err != nil {
		t.Fatalf("AddReview Slack: %v", err)
	}
	_, err = store.AddReview(ctx, "GitHub", 3, "Okay")
	if err != nil {
		t.Fatalf("AddReview GitHub: %v", err)
	}

	results, err := store.GetTopVendors(ctx, "")
	if err != nil {
		t.Fatalf("GetTopVendors: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one top vendor")
	}
	// Slack avg = (5+4)/2 = 4.5, GitHub avg = 3.0
	if results[0]["vendor"] != "Slack" {
		t.Errorf("expected top vendor Slack, got %v", results[0]["vendor"])
	}
}

func TestGetTopVendors_Empty(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	results, err := store.GetTopVendors(ctx, "")
	if err != nil {
		t.Fatalf("GetTopVendors: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
