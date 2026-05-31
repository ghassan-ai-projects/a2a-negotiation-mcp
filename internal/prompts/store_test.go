package prompts

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

func TestSaveAndGetPrompt(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.SavePrompt(ctx, "Price Negotiation", "You are negotiating {{vendor}} pricing.", "negotiation,price")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	if saved.Name != "Price Negotiation" {
		t.Errorf("expected name 'Price Negotiation', got %s", saved.Name)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	got, err := s.GetPrompt(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if got.Name != saved.Name {
		t.Errorf("expected name %s, got %s", saved.Name, got.Name)
	}
	if got.Tags != "negotiation,price" {
		t.Errorf("expected tags 'negotiation,price', got %s", got.Tags)
	}
}

func TestListPrompts_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	prompts, err := s.ListPrompts(ctx, "")
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if prompts == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(prompts) != 0 {
		t.Errorf("expected 0 prompts, got %d", len(prompts))
	}
}

func TestListPrompts_FilteredByTag(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.SavePrompt(ctx, "Contract Review", "Review {{contract}} terms.", "legal,contract")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}
	_, err = s.SavePrompt(ctx, "Price Negotiation", "Negotiate {{vendor}} price.", "negotiation,price")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}
	_, err = s.SavePrompt(ctx, "Legal Compliance", "Check {{regulation}} compliance.", "legal,compliance")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	legalPrompts, err := s.ListPrompts(ctx, "legal")
	if err != nil {
		t.Fatalf("ListPrompts(legal): %v", err)
	}
	if len(legalPrompts) != 2 {
		t.Errorf("expected 2 legal prompts, got %d", len(legalPrompts))
	}

	// Verify both legal prompts are present
	titles := make(map[string]bool)
	for _, p := range legalPrompts {
		titles[p.Name] = true
	}
	if !titles["Contract Review"] {
		t.Error("expected 'Contract Review' in results")
	}
	if !titles["Legal Compliance"] {
		t.Error("expected 'Legal Compliance' in results")
	}

	// Verify non-legal prompt is not in the results
	for _, p := range legalPrompts {
		if p.Name == "Price Negotiation" {
			t.Error("unexpected Price Negotiation in legal results")
		}
	}
}

func TestGetPrompt_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetPrompt(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent prompt")
	}
}

func TestRenderPrompt_ReplacesVariables(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.SavePrompt(ctx, "Vendor Prompt", "You are negotiating with {{vendor}} for {{sku}}. Budget: {{budget}}.", "negotiation")
	if err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	variables := map[string]string{
		"vendor": "Slack",
		"sku":    "Pro",
		"budget": "5000",
	}

	rendered, err := s.RenderPrompt(ctx, saved.ID, variables)
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}

	expected := "You are negotiating with Slack for Pro. Budget: 5000."
	if rendered != expected {
		t.Errorf("expected %q, got %q", expected, rendered)
	}
}
