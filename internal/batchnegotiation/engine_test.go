package batchnegotiation

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func TestBatchWithSingleItem(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(store, hstore, logger)

	result, err := eng.Run(context.Background(), BatchRequest{
		Items: []BatchItem{
			{Vendor: "Slack", SKU: "Pro", Strategy: "balanced", Budget: 8.75},
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.BatchID == "" {
		t.Error("expected non-empty batch_id")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Status != "completed" {
		t.Errorf("expected status completed, got %s", result.Results[0].Status)
	}
	if result.Results[0].SessionID == "" {
		t.Error("expected non-empty session_id")
	}
	if result.TotalSavings <= 0 {
		t.Errorf("expected positive total_savings, got %f", result.TotalSavings)
	}
}

func TestBatchWithMultipleItems(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(store, hstore, logger)

	result, err := eng.Run(context.Background(), BatchRequest{
		Items: []BatchItem{
			{Vendor: "Slack", SKU: "Pro", Strategy: "aggressive", Budget: 8.75},
			{Vendor: "GitHub", SKU: "Team", Strategy: "balanced", Budget: 4.00},
			{Vendor: "Salesforce", SKU: "Enterprise", Strategy: "conservative", Budget: 165.00},
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}
	for i, r := range result.Results {
		if r.Status != "completed" {
			t.Errorf("result %d: expected status completed, got %s", i, r.Status)
		}
		if r.SessionID == "" {
			t.Errorf("result %d: expected non-empty session_id", i)
		}
	}
	if result.TotalSavings <= 0 {
		t.Errorf("expected positive total_savings, got %f", result.TotalSavings)
	}
}

func TestBatchEmptyItemsReturnsError(t *testing.T) {
	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	defer pstore.Close()

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(store, hstore, logger)

	_, err = eng.Run(context.Background(), BatchRequest{Items: []BatchItem{}})
	if err == nil {
		t.Fatal("expected error for empty items, got nil")
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
func (discard) Close() error                { return nil }
