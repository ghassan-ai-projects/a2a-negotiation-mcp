package contractclauses

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

func TestAddAndGetClause(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	saved, err := s.AddClause(ctx, "confidentiality", "Non-Disclosure Agreement", "The parties agree to keep confidential...", "Standard NDA clause")
	if err != nil {
		t.Fatalf("AddClause: %v", err)
	}

	if saved.Title != "Non-Disclosure Agreement" {
		t.Errorf("expected title 'Non-Disclosure Agreement', got %s", saved.Title)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if saved.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	got, err := s.GetClause(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetClause: %v", err)
	}
	if got.Title != saved.Title {
		t.Errorf("expected title %s, got %s", saved.Title, got.Title)
	}
	if got.Category != "confidentiality" {
		t.Errorf("expected category confidentiality, got %s", got.Category)
	}
}

func TestGetClause_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetClause(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent clause")
	}
}

func TestListClauses_All(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	s.AddClause(ctx, "payment", "Net 30 Terms", "Payment shall be due within 30 days...", "")
	s.AddClause(ctx, "termination", "Termination for Cause", "Either party may terminate...", "With cause termination")

	clauses, err := s.ListClauses(ctx, "")
	if err != nil {
		t.Fatalf("ListClauses: %v", err)
	}

	if len(clauses) != 2 {
		t.Errorf("expected 2 clauses, got %d", len(clauses))
	}
	// ORDER BY title
	if clauses[0].Title > clauses[1].Title {
		t.Errorf("expected clauses ordered by title, got %s before %s", clauses[0].Title, clauses[1].Title)
	}
}

func TestListClauses_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	clauses, err := s.ListClauses(ctx, "")
	if err != nil {
		t.Fatalf("ListClauses: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected empty list, got %d clauses", len(clauses))
	}
}

func TestListClauses_FilteredByCategory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	s.AddClause(ctx, "payment", "Net 30 Terms", "Payment shall be due within 30 days...", "")
	s.AddClause(ctx, "termination", "Termination for Cause", "Either party may terminate...", "")
	s.AddClause(ctx, "payment", "Early Payment Discount", "A 2% discount applies...", "")

	clauses, err := s.ListClauses(ctx, "payment")
	if err != nil {
		t.Fatalf("ListClauses: %v", err)
	}

	if len(clauses) != 2 {
		t.Errorf("expected 2 payment clauses, got %d", len(clauses))
	}
}

func TestSearchClauses(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	s.AddClause(ctx, "payment", "Net 30 Terms", "Payment shall be due within 30 days of invoice", "Standard payment term")
	s.AddClause(ctx, "confidentiality", "Non-Disclosure Agreement", "The parties agree to keep confidential information secret", "Standard NDA")
	s.AddClause(ctx, "termination", "Termination for Cause", "Either party may terminate this agreement upon material breach", "")

	// Search by title
	clauses, err := s.SearchClauses(ctx, "Net 30")
	if err != nil {
		t.Fatalf("SearchClauses: %v", err)
	}
	if len(clauses) != 1 {
		t.Errorf("expected 1 clause for 'Net 30', got %d", len(clauses))
	}

	// Search by content
	clauses, err = s.SearchClauses(ctx, "confidential")
	if err != nil {
		t.Fatalf("SearchClauses: %v", err)
	}
	if len(clauses) != 1 {
		t.Errorf("expected 1 clause for 'confidential', got %d", len(clauses))
	}

	// Search with no matches
	clauses, err = s.SearchClauses(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("SearchClauses: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for 'nonexistent', got %d", len(clauses))
	}

	// Search across title and content — "agreement" appears in clause 2 title and clause 3 content
	clauses, err = s.SearchClauses(ctx, "agreement")
	if err != nil {
		t.Fatalf("SearchClauses: %v", err)
	}
	if len(clauses) != 2 {
		t.Errorf("expected 2 clauses for 'agreement', got %d", len(clauses))
	}
}
