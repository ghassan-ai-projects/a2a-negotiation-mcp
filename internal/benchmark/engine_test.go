package benchmark

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// mockHistoryStore implements HistoryStore for testing.
type mockHistoryStore struct {
	histories map[string]*history.HistorySummary
	deals     map[string][]history.DealOutcome
}

func (m *mockHistoryStore) GetHistory(ctx context.Context, vendor string, period string) (*history.HistorySummary, error) {
	key := vendor + ":" + period
	if h, ok := m.histories[key]; ok {
		return h, nil
	}
	return &history.HistorySummary{}, nil
}

func (m *mockHistoryStore) GetSimilarDeals(ctx context.Context, vendor string, limit int) ([]history.DealOutcome, error) {
	if d, ok := m.deals[vendor]; ok {
		if len(d) > limit {
			return d[:limit], nil
		}
		return d, nil
	}
	return nil, nil
}

func setupBenchmarkTest(t *testing.T) (*Engine, *mockHistoryStore) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mock := &mockHistoryStore{
		histories: make(map[string]*history.HistorySummary),
		deals:     make(map[string][]history.DealOutcome),
	}

	// Seed global history with deals
	now := time.Now()
	globalDeals := []history.DealOutcome{
		{Vendor: "VendorA", ListPrice: 1000, FinalPrice: 800, DiscountPct: 0.20, CreatedAt: now},
		{Vendor: "VendorA", ListPrice: 2000, FinalPrice: 1500, DiscountPct: 0.25, CreatedAt: now},
		{Vendor: "VendorB", ListPrice: 500, FinalPrice: 450, DiscountPct: 0.10, CreatedAt: now},
		{Vendor: "VendorC", ListPrice: 3000, FinalPrice: 2100, DiscountPct: 0.30, CreatedAt: now},
	}

	mock.histories[":90d"] = &history.HistorySummary{
		TotalDeals:     4,
		AvgDiscountPct: 21.25,
		TotalSavings:   650,
		Deals:          globalDeals,
	}

	// User-specific history (VendorA only)
	userDeals := []history.DealOutcome{
		{Vendor: "VendorA", ListPrice: 1000, FinalPrice: 750, DiscountPct: 0.25, CreatedAt: now},
		{Vendor: "VendorA", ListPrice: 2000, FinalPrice: 1400, DiscountPct: 0.30, CreatedAt: now},
	}

	mock.histories["VendorA:90d"] = &history.HistorySummary{
		TotalDeals:     2,
		AvgDiscountPct: 27.5,
		TotalSavings:   850,
		Deals:          userDeals,
	}

	engine := NewEngine(mock, logger)
	return engine, mock
}

func TestGenerateReport_WithData(t *testing.T) {
	eng, _ := setupBenchmarkTest(t)
	ctx := context.Background()

	report, err := eng.GenerateReport(ctx, "user-1", "VendorA", "", "90d")
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", report.UserID)
	}
	if report.Period != "90d" {
		t.Errorf("expected period 90d, got %s", report.Period)
	}
	if report.DealCount != 2 {
		t.Errorf("expected 2 deals, got %d", report.DealCount)
	}
	// User savings = (1000-750) + (2000-1400) = 250 + 600 = 850
	if report.TotalSavings != 850 {
		t.Errorf("expected savings 850, got %f", report.TotalSavings)
	}
	// User avg discount = (0.25 + 0.30) / 2 = 0.275
	if report.AvgDiscountPct != 27.5 {
		t.Errorf("expected avg discount 27.5, got %f", report.AvgDiscountPct)
	}
	// User percentile should be above median (>50) since 0.275 > 0.2125
	if report.Percentile <= 50 {
		t.Errorf("expected percentile >50, got %f", report.Percentile)
	}

	// Should have VendorA in by_vendor
	if len(report.ByVendor) != 1 {
		t.Errorf("expected 1 vendor in report, got %d", len(report.ByVendor))
	} else if report.ByVendor[0].Vendor != "VendorA" {
		t.Errorf("expected VendorA, got %s", report.ByVendor[0].Vendor)
	}
}

func TestGenerateReport_Empty(t *testing.T) {
	eng, mock := setupBenchmarkTest(t)
	ctx := context.Background()

	// Add empty user history
	mock.histories["NoVendor:90d"] = &history.HistorySummary{}

	report, err := eng.GenerateReport(ctx, "user-2", "NoVendor", "", "90d")
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.DealCount != 0 {
		t.Errorf("expected 0 deals for empty, got %d", report.DealCount)
	}
}

func TestGenerateReport_NoGlobalData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mock := &mockHistoryStore{
		histories: make(map[string]*history.HistorySummary),
	}
	eng := NewEngine(mock, logger)
	ctx := context.Background()

	// User has deals but no global data
	mock.histories["VendorX:30d"] = &history.HistorySummary{
		TotalDeals:     1,
		AvgDiscountPct: 15.0,
		TotalSavings:   100,
		Deals: []history.DealOutcome{
			{Vendor: "VendorX", ListPrice: 1000, FinalPrice: 900, DiscountPct: 0.10, CreatedAt: time.Now()},
		},
	}

	report, err := eng.GenerateReport(ctx, "user-3", "VendorX", "", "30d")
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	// Without global data, percentile defaults to 50
	if report.Percentile != 50 {
		t.Errorf("expected percentile 50 for no global data, got %f", report.Percentile)
	}
}

func TestGenerateReport_InvalidPeriod(t *testing.T) {
	eng, _ := setupBenchmarkTest(t)
	ctx := context.Background()

	_, err := eng.GenerateReport(ctx, "user-1", "", "", "invalid")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}

func TestGenerateReport_DefaultPeriod(t *testing.T) {
	eng, mock := setupBenchmarkTest(t)
	ctx := context.Background()

	// Add history with empty period
	mock.histories["VendorA:90d"] = &history.HistorySummary{
		TotalDeals:     2,
		AvgDiscountPct: 27.5,
		TotalSavings:   850,
		Deals: []history.DealOutcome{
			{Vendor: "VendorA", ListPrice: 1000, FinalPrice: 750, DiscountPct: 0.25, CreatedAt: time.Now()},
		},
	}

	report, err := eng.GenerateReport(ctx, "user-1", "VendorA", "", "") // empty period defaults to 90d
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.Period != "90d" {
		t.Errorf("expected default period 90d, got %s", report.Period)
	}
}
