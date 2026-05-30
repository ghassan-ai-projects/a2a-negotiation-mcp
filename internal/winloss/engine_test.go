package winloss

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	_ "modernc.org/sqlite"
)

func setupWinLossTest(t *testing.T) (*Engine, *history.Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hStore, err := history.NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := NewEngine(hStore, logger)

	return engine, hStore
}

func seedSessions(t *testing.T, store *history.Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	sessions := []*history.SessionRecord{
		{ID: "s1", Vendor: "Slack", Strategy: "aggressive", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
		{ID: "s2", Vendor: "Slack", Strategy: "aggressive", Status: "completed", Outcome: "lost", CreatedAt: now, UpdatedAt: now},
		{ID: "s3", Vendor: "Slack", Strategy: "balanced", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
		{ID: "s4", Vendor: "GitHub", Strategy: "conservative", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
		{ID: "s5", Vendor: "GitHub", Strategy: "conservative", Status: "completed", Outcome: "lost", CreatedAt: now, UpdatedAt: now},
		{ID: "s6", Vendor: "GitHub", Strategy: "balanced", Status: "completed", Outcome: "won", CreatedAt: now, UpdatedAt: now},
		{ID: "s7", Vendor: "Slack", Strategy: "balanced", Status: "active", Outcome: "in_progress", CreatedAt: now, UpdatedAt: now},
	}

	for _, s := range sessions {
		if err := store.SaveSession(ctx, s); err != nil {
			t.Fatalf("SaveSession: %v", err)
		}
	}

	// Also add some deal outcomes
	deals := []*history.DealOutcome{
		{Vendor: "Slack", SKU: "standard", ListPrice: 100, FinalPrice: 80, DiscountPct: 0.2, Strategy: "aggressive", SessionID: "s8", CreatedAt: now},
	}
	for _, d := range deals {
		if err := store.SaveDealOutcome(ctx, d); err != nil {
			t.Fatalf("SaveDealOutcome: %v", err)
		}
	}
}

// ─── Tests ───

func TestAnalyze_MixedOutcomes(t *testing.T) {
	engine, store := setupWinLossTest(t)
	seedSessions(t, store)
	ctx := context.Background()

	report, err := engine.Analyze(ctx, "", "", "all")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if report.TotalDeals == 0 {
		t.Fatal("expected non-zero total deals")
	}
	if report.Won == 0 {
		t.Errorf("expected at least 1 won, got %d", report.Won)
	}
	if report.Lost == 0 {
		t.Errorf("expected at least 1 lost, got %d", report.Lost)
	}
	if report.WinRate <= 0 {
		t.Errorf("expected positive win rate, got %f", report.WinRate)
	}
}

func TestAnalyze_ByVendor(t *testing.T) {
	engine, store := setupWinLossTest(t)
	seedSessions(t, store)
	ctx := context.Background()

	report, err := engine.Analyze(ctx, "", "", "all")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.ByVendor) == 0 {
		t.Fatal("expected vendor breakdowns")
	}

	found := false
	for _, vb := range report.ByVendor {
		if vb.Vendor == "Slack" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Slack in vendor breakdown")
	}
}

func TestAnalyze_ByStrategy(t *testing.T) {
	engine, store := setupWinLossTest(t)
	seedSessions(t, store)
	ctx := context.Background()

	report, err := engine.Analyze(ctx, "", "", "all")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(report.ByStrategy) == 0 {
		t.Fatal("expected strategy breakdowns")
	}
}

func TestAnalyze_EmptyData(t *testing.T) {
	engine, _ := setupWinLossTest(t)
	ctx := context.Background()

	report, err := engine.Analyze(ctx, "", "", "all")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if report.TotalDeals != 0 {
		t.Errorf("expected 0 total deals, got %d", report.TotalDeals)
	}
	if report.Won != 0 {
		t.Errorf("expected 0 won, got %d", report.Won)
	}
	if report.WinRate != 0 {
		t.Errorf("expected 0 win rate, got %f", report.WinRate)
	}
}

func TestAnalyze_InvalidPeriod(t *testing.T) {
	engine, _ := setupWinLossTest(t)
	ctx := context.Background()

	_, err := engine.Analyze(ctx, "", "", "invalid")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}

func TestAnalyze_FilterByVendor(t *testing.T) {
	engine, store := setupWinLossTest(t)
	seedSessions(t, store)
	ctx := context.Background()

	report, err := engine.Analyze(ctx, "GitHub", "", "all")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if report.TotalDeals == 0 {
		t.Error("expected deals for GitHub")
	}
}
