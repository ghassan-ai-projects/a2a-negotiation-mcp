package export

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	_ "modernc.org/sqlite"
)

func setupExportTest(t *testing.T) (*Engine, *history.Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	exportStore, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewExportStore: %v", err)
	}

	hStore, err := history.NewStore(db)
	if err != nil {
		t.Fatalf("NewHistoryStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := NewEngine(exportStore, hStore, logger)

	return engine, hStore
}

func seedDeals(t *testing.T, store *history.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	deals := []*history.DealOutcome{
		{Vendor: "Slack", SKU: "standard", ListPrice: 100, FinalPrice: 80, DiscountPct: 0.2, Seats: 50, TermMonths: 12, Strategy: "aggressive", CreatedAt: now},
		{Vendor: "GitHub", SKU: "team", ListPrice: 200, FinalPrice: 150, DiscountPct: 0.25, Seats: 20, TermMonths: 24, Strategy: "balanced", CreatedAt: now},
		{Vendor: "Slack", SKU: "pro", ListPrice: 300, FinalPrice: 240, DiscountPct: 0.2, Seats: 10, TermMonths: 12, Strategy: "aggressive", CreatedAt: now},
	}

	for _, d := range deals {
		if err := store.SaveDealOutcome(ctx, d); err != nil {
			t.Fatalf("SaveDealOutcome: %v", err)
		}
	}
}

func seedSessions(t *testing.T, store *history.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	sessions := []*history.SessionRecord{
		{ID: "s1", Vendor: "Slack", Strategy: "aggressive", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
		{ID: "s2", Vendor: "GitHub", Strategy: "balanced", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
	}

	for _, s := range sessions {
		if err := store.SaveSession(ctx, s); err != nil {
			t.Fatalf("SaveSession: %v", err)
		}
	}
}

// ─── Tests ───

func TestExport_DealsCSV(t *testing.T) {
	engine, store := setupExportTest(t)
	seedDeals(t, store)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "csv", Type: "deals"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount != 3 {
		t.Errorf("expected 3 deal records, got %d", result.RecordCount)
	}
	if result.Format != "csv" {
		t.Errorf("expected csv format, got %s", result.Format)
	}
	if !strings.HasSuffix(result.Filename, ".csv") {
		t.Errorf("expected .csv filename, got %s", result.Filename)
	}
	if !strings.Contains(result.Data, "Slack") {
		t.Error("expected Slack in CSV data")
	}
	if !strings.HasPrefix(result.Data, "vendor") {
		t.Error("expected CSV header row")
	}
}

func TestExport_DealsJSON(t *testing.T) {
	engine, store := setupExportTest(t)
	seedDeals(t, store)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "json", Type: "deals"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount != 3 {
		t.Errorf("expected 3 deal records, got %d", result.RecordCount)
	}
	if !strings.HasSuffix(result.Filename, ".json") {
		t.Errorf("expected .json filename, got %s", result.Filename)
	}

	// Verify valid JSON
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(result.Data), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 3 {
		t.Errorf("expected 3 JSON objects, got %d", len(parsed))
	}
}

func TestExport_SessionsCSV(t *testing.T) {
	engine, store := setupExportTest(t)
	seedSessions(t, store)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "csv", Type: "sessions"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount != 2 {
		t.Errorf("expected 2 session records, got %d", result.RecordCount)
	}
	if !strings.Contains(result.Data, "Slack") {
		t.Error("expected Slack in session data")
	}
}

func TestExport_Analytics(t *testing.T) {
	engine, store := setupExportTest(t)
	seedDeals(t, store)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "csv", Type: "analytics"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount != 1 {
		t.Errorf("expected 1 analytics record, got %d", result.RecordCount)
	}
	if !strings.Contains(result.Data, "total_deals") {
		t.Error("expected total_deals in analytics")
	}
}

func TestExport_AllJSON(t *testing.T) {
	engine, store := setupExportTest(t)
	seedDeals(t, store)
	seedSessions(t, store)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "json", Type: "all"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount < 3 {
		t.Errorf("expected at least 3 total records, got %d", result.RecordCount)
	}

	// Verify valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result.Data), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := parsed["deals"]; !ok {
		t.Error("expected deals key in all export")
	}
	if _, ok := parsed["sessions"]; !ok {
		t.Error("expected sessions key in all export")
	}
}

func TestExport_EmptyData(t *testing.T) {
	engine, _ := setupExportTest(t)
	ctx := context.Background()

	result, err := engine.Export(ctx, ExportRequest{Format: "csv", Type: "deals"})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.RecordCount != 0 {
		t.Errorf("expected 0 records, got %d", result.RecordCount)
	}
}

func TestExport_InvalidFormat(t *testing.T) {
	engine, _ := setupExportTest(t)
	ctx := context.Background()

	_, err := engine.Export(ctx, ExportRequest{Format: "xml", Type: "deals"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestExport_InvalidType(t *testing.T) {
	engine, _ := setupExportTest(t)
	ctx := context.Background()

	_, err := engine.Export(ctx, ExportRequest{Format: "csv", Type: "invalid"})
	if err == nil {
		t.Error("expected error for invalid export type")
	}
}
