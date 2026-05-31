package dataimport

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupDataImportTest(t *testing.T) *Engine {
	t.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pstore.Close() })

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	return NewEngine(pstore, hstore)
}

func TestImportDealsJSON(t *testing.T) {
	eng := setupDataImportTest(t)
	ctx := context.Background()

	dealsJSON := `[
		{"vendor": "Slack", "sku": "Pro", "list_price": 8.75, "final_price": 7.00, "discount_percentage": 20, "seats": 50, "term_months": 12, "strategy": "balanced"},
		{"vendor": "GitHub", "sku": "Team", "list_price": 4.00, "final_price": 3.20, "discount_percentage": 20, "seats": 100, "term_months": 12, "strategy": "aggressive"}
	]`

	result, err := eng.Import(ctx, ImportRequest{
		Type: ImportTypeDeals,
		Data: dealsJSON,
		Mode: ImportModeImport,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if result.ValidCount != 2 {
		t.Errorf("expected 2 valid records, got %d", result.ValidCount)
	}
	if result.ImportedCount != 2 {
		t.Errorf("expected 2 imported, got %d", result.ImportedCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Summary == "" {
		t.Errorf("expected non-empty summary")
	}
}

func TestImportValidateOnly(t *testing.T) {
	eng := setupDataImportTest(t)
	ctx := context.Background()

	dealsJSON := `[
		{"vendor": "Slack", "sku": "Pro", "list_price": 8.75, "final_price": 7.00, "discount_percentage": 20, "seats": 50, "term_months": 12, "strategy": "balanced"}
	]`

	result, err := eng.Validate(ctx, ImportRequest{
		Type: ImportTypeDeals,
		Data: dealsJSON,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.ValidCount != 1 {
		t.Errorf("expected 1 valid record, got %d", result.ValidCount)
	}
	if result.ImportedCount != 0 {
		t.Errorf("expected 0 imported in validate mode, got %d", result.ImportedCount)
	}
}

func TestImportInvalidData(t *testing.T) {
	eng := setupDataImportTest(t)
	ctx := context.Background()

	dealsJSON := `[
		{"vendor": "", "sku": "Pro", "list_price": 8.75, "final_price": 7.00},
		{"vendor": "Slack", "sku": "Pro", "list_price": 0, "final_price": 7.00},
		{"vendor": "Slack", "sku": "Pro", "list_price": 8.75, "final_price": 0}
	]`

	result, err := eng.Validate(ctx, ImportRequest{
		Type: ImportTypeDeals,
		Data: dealsJSON,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if result.ValidCount != 0 {
		t.Errorf("expected 0 valid records, got %d", result.ValidCount)
	}
	if len(result.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}
