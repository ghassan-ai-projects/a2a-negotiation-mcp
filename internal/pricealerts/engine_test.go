package pricealerts

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func inMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEngine_EnableAlert(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	getLatest := func(ctx context.Context, vendor, sku string) (float64, error) {
		return 100, nil
	}

	eng := NewEngine(s, getLatest, testLogger())
	ctx := context.Background()

	rule, err := eng.EnableAlert(ctx, "Slack", "Pro", 10, "webhook")
	if err != nil {
		t.Fatalf("EnableAlert: %v", err)
	}
	if rule.Vendor != "Slack" {
		t.Errorf("vendor = %q, want Slack", rule.Vendor)
	}
	if rule.SKU != "Pro" {
		t.Errorf("sku = %q, want Pro", rule.SKU)
	}
	if rule.ThresholdPct != 10 {
		t.Errorf("threshold = %f, want 10", rule.ThresholdPct)
	}
	if !rule.Enabled {
		t.Error("rule not enabled")
	}

	baseline, err := s.GetBaseline(ctx, "Slack", "Pro")
	if err != nil {
		t.Fatalf("GetBaseline: %v", err)
	}
	if baseline != 100 {
		t.Errorf("baseline = %f, want 100", baseline)
	}
}

func TestEngine_CheckAlerts_PriceDrop(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Background()

	getLatest := func(ctx context.Context, vendor, sku string) (float64, error) {
		return 100, nil
	}
	eng := NewEngine(s, getLatest, testLogger())

	if _, err := eng.EnableAlert(ctx, "Slack", "Pro", 10, "webhook"); err != nil {
		t.Fatalf("EnableAlert: %v", err)
	}

	// Price dropped to 80 (20% drop, threshold is 10%)
	eng2 := NewEngine(s, func(ctx context.Context, vendor, sku string) (float64, error) {
		return 80, nil
	}, testLogger())

	results, err := eng2.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].ThresholdMet {
		t.Error("threshold should be met")
	}
	if results[0].DropPct != 20 {
		t.Errorf("drop_pct = %f, want 20", results[0].DropPct)
	}
}

func TestEngine_CheckAlerts_StablePrice(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Background()

	getLatest := func(ctx context.Context, vendor, sku string) (float64, error) {
		return 100, nil
	}
	eng := NewEngine(s, getLatest, testLogger())

	if _, err := eng.EnableAlert(ctx, "Slack", "Pro", 10, "webhook"); err != nil {
		t.Fatalf("EnableAlert: %v", err)
	}

	results, err := eng.CheckAlerts(ctx)
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ThresholdMet {
		t.Error("threshold should NOT be met for stable price")
	}
	if results[0].DropPct != 0 {
		t.Errorf("drop_pct = %f, want 0", results[0].DropPct)
	}
}

func TestEngine_DisableAlert(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Background()

	getLatest := func(ctx context.Context, vendor, sku string) (float64, error) {
		return 100, nil
	}
	eng := NewEngine(s, getLatest, testLogger())

	if _, err := eng.EnableAlert(ctx, "Slack", "Pro", 10, "webhook"); err != nil {
		t.Fatalf("EnableAlert: %v", err)
	}

	if err := eng.DisableAlert(ctx, "Slack", "Pro"); err != nil {
		t.Fatalf("DisableAlert: %v", err)
	}

	rule, err := s.GetRule(ctx, "Slack", "Pro")
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if rule != nil {
		t.Error("rule should be nil after disable")
	}
}

func TestEngine_CheckAlerts_NoRules(t *testing.T) {
	db := inMemoryDB(t)
	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	eng := NewEngine(s, func(ctx context.Context, vendor, sku string) (float64, error) {
		return 0, nil
	}, testLogger())

	results, err := eng.CheckAlerts(context.Background())
	if err != nil {
		t.Fatalf("CheckAlerts: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
