package vendorknowledge

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

func TestIngestAndGetDocument(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	doc, err := s.IngestDocument(ctx, "Acme Corp", "Annual report 2026 content", "report")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	if doc.Vendor != "Acme Corp" {
		t.Errorf("expected vendor 'Acme Corp', got %s", doc.Vendor)
	}
	if doc.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if doc.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
	if doc.DocType != "report" {
		t.Errorf("expected doc_type 'report', got %s", doc.DocType)
	}

	// Search back to verify it was stored
	results, err := s.SearchDocs(ctx, "Acme Corp", "report")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "Annual report 2026 content" {
		t.Errorf("expected content 'Annual report 2026 content', got %s", results[0].Content)
	}
}

func TestSearchDocs_ByVendorAndQuery(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.IngestDocument(ctx, "Vendor A", "Pricing terms for Q1", "contract")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	_, err = s.IngestDocument(ctx, "Vendor A", "Security compliance checklist", "compliance")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	_, err = s.IngestDocument(ctx, "Vendor B", "Pricing terms for Q2", "contract")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// Search Vendor A for "Pricing"
	results, err := s.SearchDocs(ctx, "Vendor A", "Pricing")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for Vendor A 'Pricing', got %d", len(results))
	}
	if len(results) > 0 && results[0].Content != "Pricing terms for Q1" {
		t.Errorf("expected 'Pricing terms for Q1', got %s", results[0].Content)
	}

	// Search Vendor B for "Pricing"
	results, err = s.SearchDocs(ctx, "Vendor B", "Pricing")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for Vendor B 'Pricing', got %d", len(results))
	}

	// Search Vendor A for "Security" (only matches Vendor A)
	results, err = s.SearchDocs(ctx, "Vendor A", "Security")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for Vendor A 'Security', got %d", len(results))
	}
}

func TestKnowledgeReport_DocTypeBreakdown(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Ingest documents for Vendor X with different doc types
	_, err := s.IngestDocument(ctx, "Vendor X", "Contract content", "contract")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	_, err = s.IngestDocument(ctx, "Vendor X", "Compliance doc", "compliance")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	_, err = s.IngestDocument(ctx, "Vendor X", "Another contract", "contract")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	// A different vendor to ensure it's not counted
	_, err = s.IngestDocument(ctx, "Vendor Y", "Report", "report")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	report, err := s.GetKnowledgeReport(ctx, "Vendor X")
	if err != nil {
		t.Fatalf("GetKnowledgeReport: %v", err)
	}

	totalDocs, ok := report["total_docs"].(int)
	if !ok {
		t.Fatal("expected total_docs to be int")
	}
	if totalDocs != 3 {
		t.Errorf("expected total_docs 3, got %d", totalDocs)
	}

	breakdown, ok := report["doc_type_breakdown"].(map[string]int)
	if !ok {
		t.Fatal("expected doc_type_breakdown to be map[string]int")
	}
	if breakdown["contract"] != 2 {
		t.Errorf("expected contract count 2, got %d", breakdown["contract"])
	}
	if breakdown["compliance"] != 1 {
		t.Errorf("expected compliance count 1, got %d", breakdown["compliance"])
	}

	mostRecent, ok := report["most_recent"].(*string)
	if !ok || mostRecent == nil {
		t.Fatal("expected most_recent to be non-nil *string")
	}
}

func TestSearchDocs_NoResults(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	docs, err := s.SearchDocs(ctx, "Nonexistent Vendor", "anything")
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if docs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}
