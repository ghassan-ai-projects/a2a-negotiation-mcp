package batchcsv

import (
	"context"
	"testing"
)

func TestProcessCSV_Valid(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,budget,target_price,notes\nAcme Corp,competitive,10000,9500,first deal\nBeta Inc,collaborative,50000,48000,\nGamma Ltd,aggressive,7500,7000,urgent"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 3 {
		t.Errorf("expected CreatedCount 3, got %d", result.CreatedCount)
	}
	if result.RowCount != 3 {
		t.Errorf("expected RowCount 3, got %d", result.RowCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestProcessCSV_WithErrors(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,budget,target_price,notes\nAcme Corp,competitive,10000,9500,valid\n,aggressive,5000,4500,missing vendor\nBeta Inc,invalid_strat,50000,48000,invalid strategy\nGamma Ltd,collaborative,0,7000,zero budget\nDelta Ltd,collaborative,-100,7000,negative budget"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 1 {
		t.Errorf("expected CreatedCount 1, got %d", result.CreatedCount)
	}
	if result.RowCount != 5 {
		t.Errorf("expected RowCount 5, got %d", result.RowCount)
	}
	if len(result.Errors) != 4 {
		t.Errorf("expected 4 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestProcessCSV_Empty(t *testing.T) {
	eng := NewEngine()

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 0 {
		t.Errorf("expected CreatedCount 0, got %d", result.CreatedCount)
	}
	if len(result.Errors) != 1 || result.Errors[0] != "csv content is empty" {
		t.Errorf("expected empty csv error, got %v", result.Errors)
	}
}

func TestProcessCSV_BlankRows(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,budget,target_price,notes\n\nAcme Corp,competitive,10000,9500,first deal\n\n\nBeta Inc,collaborative,50000,48000,\n"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 2 {
		t.Errorf("expected CreatedCount 2, got %d", result.CreatedCount)
	}
	if result.RowCount != 2 {
		t.Errorf("expected RowCount 2, got %d", result.RowCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestProcessCSV_HeaderOnly(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,budget,target_price,notes"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 0 {
		t.Errorf("expected CreatedCount 0, got %d", result.CreatedCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error for no data rows, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestProcessCSV_InvalidHeader(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,price,notes\nAcme Corp,competitive,10000,first deal"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 0 {
		t.Errorf("expected CreatedCount 0, got %d", result.CreatedCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error for invalid header, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestProcessCSV_TooFewColumns(t *testing.T) {
	eng := NewEngine()
	csv := "vendor,strategy,budget,target_price,notes\nAcme Corp,competitive"

	ctx := context.Background()
	result, err := eng.ProcessCSV(ctx, csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CreatedCount != 0 {
		t.Errorf("expected CreatedCount 0, got %d", result.CreatedCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
}
