package trends

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"
)

func setupTrendsStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func seedTrendData(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	// Rising trend: Slack standard
	for i := 11; i >= 0; i-- {
		date := now.AddDate(0, -i, 0)
		price := 7.0 + float64(12-i)*0.2 // 7.0 -> 9.4 over 12 months
		store.Save(ctx, &PriceSnapshot{
			Vendor:    "Slack",
			SKU:       "standard",
			Price:     price,
			ListPrice: 10.0,
			Date:      date,
			CreatedAt: now,
		})
	}

	// Falling trend: GitHub team
	for i := 11; i >= 0; i-- {
		date := now.AddDate(0, -i, 0)
		price := 4.0 - float64(12-i)*0.15 // 4.0 -> 2.2 over 12 months
		store.Save(ctx, &PriceSnapshot{
			Vendor:    "GitHub",
			SKU:       "team",
			Price:     price,
			ListPrice: 5.0,
			Date:      date,
			CreatedAt: now,
		})
	}

	// Stable: DigitalOcean basic
	for i := 11; i >= 0; i-- {
		date := now.AddDate(0, -i, 0)
		store.Save(ctx, &PriceSnapshot{
			Vendor:    "DigitalOcean",
			SKU:       "basic",
			Price:     5.0,
			ListPrice: 6.0,
			Date:      date,
			CreatedAt: now,
		})
	}
}

func TestAnalyze_RisingTrend(t *testing.T) {
	store := setupTrendsStore(t)
	seedTrendData(t, store)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	analysis, err := eng.Analyze(ctx, "Slack", "standard", "1y")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if analysis.Vendor != "Slack" {
		t.Errorf("expected Slack, got %s", analysis.Vendor)
	}
	if analysis.SKU != "standard" {
		t.Errorf("expected standard, got %s", analysis.SKU)
	}
	if analysis.Direction != "up" {
		t.Errorf("expected direction 'up', got %s", analysis.Direction)
	}
	if analysis.Slope <= 0 {
		t.Errorf("expected positive slope, got %f", analysis.Slope)
	}
	if analysis.DataPoints < 12 {
		t.Errorf("expected at least 12 data points, got %d", analysis.DataPoints)
	}
	if len(analysis.Snapshots) < 12 {
		t.Errorf("expected at least 12 snapshots, got %d", len(analysis.Snapshots))
	}
}

func TestAnalyze_FallingTrend(t *testing.T) {
	store := setupTrendsStore(t)
	seedTrendData(t, store)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	analysis, err := eng.Analyze(ctx, "GitHub", "team", "1y")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if analysis.Direction != "down" {
		t.Errorf("expected direction 'down', got %s", analysis.Direction)
	}
	if analysis.Slope >= 0 {
		t.Errorf("expected negative slope, got %f", analysis.Slope)
	}
}

func TestAnalyze_StableTrend(t *testing.T) {
	store := setupTrendsStore(t)
	seedTrendData(t, store)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	analysis, err := eng.Analyze(ctx, "DigitalOcean", "basic", "1y")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if analysis.Direction != "stable" {
		t.Errorf("expected direction 'stable', got %s", analysis.Direction)
	}
	// Should have low volatility since price is constant
	if analysis.Volatility > 0.001 {
		t.Errorf("expected near-zero volatility, got %f", analysis.Volatility)
	}
	if analysis.PriceChange6M != 0 {
		t.Errorf("expected 0%% price change, got %f", analysis.PriceChange6M)
	}
}

func TestAnalyze_InsufficientData(t *testing.T) {
	store := setupTrendsStore(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	// Only 1 data point
	store.Save(ctx, &PriceSnapshot{
		Vendor: "NewVendor",
		SKU:    "basic",
		Price:  10.0,
		Date:   time.Now().UTC(),
	})

	analysis, err := eng.Analyze(ctx, "NewVendor", "basic", "1y")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if analysis.Direction != "insufficient_data" {
		t.Errorf("expected 'insufficient_data', got %s", analysis.Direction)
	}
	if analysis.DataPoints != 1 {
		t.Errorf("expected 1 data point, got %d", analysis.DataPoints)
	}
}

func TestAnalyze_EmptyData(t *testing.T) {
	store := setupTrendsStore(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	analysis, err := eng.Analyze(ctx, "NoDataVendor", "", "1y")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if analysis.Direction != "insufficient_data" {
		t.Errorf("expected 'insufficient_data', got %s", analysis.Direction)
	}
	if analysis.DataPoints != 0 {
		t.Errorf("expected 0 data points, got %d", analysis.DataPoints)
	}
}

func TestAnalyze_InvalidPeriod(t *testing.T) {
	store := setupTrendsStore(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	ctx := context.Background()

	_, err := eng.Analyze(ctx, "Vendor", "", "xyz")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}

func TestStore_SaveAndQuery(t *testing.T) {
	store := setupTrendsStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ps := &PriceSnapshot{
		Vendor:    "TestVendor",
		SKU:       "test-sku",
		Price:     99.50,
		ListPrice: 120.00,
		Date:      now,
		CreatedAt: now,
	}

	err := store.Save(ctx, ps)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if ps.ID == 0 {
		t.Error("expected non-zero ID")
	}

	// Query back
	results, err := store.Query(ctx, "TestVendor", "test-sku", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Price != 99.50 {
		t.Errorf("expected price 99.50, got %f", results[0].Price)
	}
}

func TestStore_GetLatest(t *testing.T) {
	store := setupTrendsStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Save 2 snapshots at different dates
	store.Save(ctx, &PriceSnapshot{
		Vendor: "V", SKU: "S", Price: 10, Date: now.AddDate(0, -1, 0), CreatedAt: now,
	})
	store.Save(ctx, &PriceSnapshot{
		Vendor: "V", SKU: "S", Price: 20, Date: now, CreatedAt: now,
	})

	latest, err := store.GetLatest(ctx, "V", "S")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil result")
	}
	if latest.Price != 20 {
		t.Errorf("expected latest price 20, got %f", latest.Price)
	}
}

func TestStore_GetLatest_NotFound(t *testing.T) {
	store := setupTrendsStore(t)
	ctx := context.Background()

	latest, err := store.GetLatest(ctx, "NonExistent", "")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest != nil {
		t.Error("expected nil for non-existent vendor")
	}
}

func TestStore_GetStats(t *testing.T) {
	store := setupTrendsStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	prices := []float64{10, 20, 30, 40, 50}
	for _, p := range prices {
		store.Save(ctx, &PriceSnapshot{
			Vendor: "StatsVendor", SKU: "stats-sku", Price: p,
			Date: now, CreatedAt: now,
		})
	}

	min, max, avg, stddev, count, err := store.GetStats(ctx, "StatsVendor", "stats-sku")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
	if min != 10 {
		t.Errorf("expected min 10, got %f", min)
	}
	if max != 50 {
		t.Errorf("expected max 50, got %f", max)
	}
	if avg != 30 {
		t.Errorf("expected avg 30, got %f", avg)
	}
	_ = stddev
}

func TestBulkInsert(t *testing.T) {
	store := setupTrendsStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	snapshots := []PriceSnapshot{
		{Vendor: "BulkVendor", SKU: "A", Price: 10, Date: now, CreatedAt: now},
		{Vendor: "BulkVendor", SKU: "A", Price: 20, Date: now.AddDate(0, 0, -1), CreatedAt: now},
	}

	if err := store.BulkInsert(ctx, snapshots); err != nil {
		t.Fatalf("BulkInsert: %v", err)
	}

	results, err := store.Query(ctx, "BulkVendor", "A", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results after bulk insert, got %d", len(results))
	}
}

func TestLinearRegression(t *testing.T) {
	// y = 2x + 1
	x := []float64{0, 1, 2, 3, 4}
	y := []float64{1, 3, 5, 7, 9}

	slope, intercept := linearRegression(x, y)

	if slope != 2.0 {
		t.Errorf("expected slope 2.0, got %f", slope)
	}
	if intercept != 1.0 {
		t.Errorf("expected intercept 1.0, got %f", intercept)
	}
}

func TestLinearRegression_Flat(t *testing.T) {
	x := []float64{0, 1, 2, 3, 4}
	y := []float64{5, 5, 5, 5, 5}

	slope, intercept := linearRegression(x, y)

	if slope != 0 {
		t.Errorf("expected slope 0, got %f", slope)
	}
	if intercept != 5.0 {
		t.Errorf("expected intercept 5.0, got %f", intercept)
	}
}

func TestCheckSeasonal(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Create snapshots with higher Q4 prices
	snapshots := []PriceSnapshot{
		{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Price: 100},  // Q1
		{Date: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Price: 105},  // Q1
		{Date: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), Price: 120}, // Q4
		{Date: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), Price: 125}, // Q4
	}
	_ = now

	result := checkSeasonal(snapshots)
	// Q4 avg = 122.5, Q1 avg = 102.5, diff = 20/102.5 = 19.5% > 10%
	if !result {
		t.Error("expected seasonal=true for Q4 > Q1 by >10%")
	}
}

func TestCheckSeasonal_NoSeasonality(t *testing.T) {
	snapshots := []PriceSnapshot{
		{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Price: 100},  // Q1
		{Date: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Price: 102},  // Q1
		{Date: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), Price: 103}, // Q4
		{Date: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), Price: 101}, // Q4
	}

	result := checkSeasonal(snapshots)
	if result {
		t.Error("expected seasonal=false for <10% difference")
	}
}
