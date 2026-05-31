package communitybench

import (
	"context"
	"math"
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

func TestUploadAndGetBenchmarks(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	entry, err := s.UploadBenchmark(ctx, "VendorA", "cloud", 15.5, 50000)
	if err != nil {
		t.Fatalf("UploadBenchmark: %v", err)
	}

	if entry.Vendor != "VendorA" {
		t.Errorf("expected vendor VendorA, got %s", entry.Vendor)
	}
	if entry.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if entry.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
	if entry.DiscountPct != 15.5 {
		t.Errorf("expected discount 15.5, got %f", entry.DiscountPct)
	}
	if entry.DealValue != 50000 {
		t.Errorf("expected deal value 50000, got %f", entry.DealValue)
	}
}

func TestGetBenchmarks_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	benchmarks, err := s.GetBenchmarks(ctx, "")
	if err != nil {
		t.Fatalf("GetBenchmarks: %v", err)
	}
	if benchmarks == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(benchmarks) != 0 {
		t.Errorf("expected 0 benchmarks, got %d", len(benchmarks))
	}
}

func TestGetBenchmarks_FilteredByCategory(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Add data for two categories
	_, _ = s.UploadBenchmark(ctx, "VendorA", "cloud", 15.0, 50000)
	_, _ = s.UploadBenchmark(ctx, "VendorB", "cloud", 20.0, 60000)
	_, _ = s.UploadBenchmark(ctx, "VendorC", "ai", 10.0, 100000)
	_, _ = s.UploadBenchmark(ctx, "VendorD", "ai", 12.0, 120000)

	// Filter by cloud
	cloudBm, err := s.GetBenchmarks(ctx, "cloud")
	if err != nil {
		t.Fatalf("GetBenchmarks(cloud): %v", err)
	}
	if len(cloudBm) != 1 {
		t.Fatalf("expected 1 cloud benchmark group, got %d", len(cloudBm))
	}
	if cloudBm[0].Category != "cloud" {
		t.Errorf("expected category cloud, got %s", cloudBm[0].Category)
	}
	if cloudBm[0].SampleCount != 2 {
		t.Errorf("expected 2 samples for cloud, got %d", cloudBm[0].SampleCount)
	}
	if math.Abs(cloudBm[0].AvgDiscount-17.5) > 0.01 {
		t.Errorf("expected avg discount ~17.5, got %f", cloudBm[0].AvgDiscount)
	}
	if math.Abs(cloudBm[0].MedianDeal-55000) > 0.01 {
		t.Errorf("expected median deal 55000, got %f", cloudBm[0].MedianDeal)
	}
}

func TestGetBenchmarks_All(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, _ = s.UploadBenchmark(ctx, "VendorA", "cloud", 15.0, 50000)
	_, _ = s.UploadBenchmark(ctx, "VendorB", "ai", 20.0, 60000)

	benchmarks, err := s.GetBenchmarks(ctx, "")
	if err != nil {
		t.Fatalf("GetBenchmarks: %v", err)
	}
	if len(benchmarks) != 2 {
		t.Errorf("expected 2 benchmark groups, got %d", len(benchmarks))
	}
}

func TestCompareToBenchmark(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Add sample data
	_, _ = s.UploadBenchmark(ctx, "VendorA", "cloud", 15.0, 50000)
	_, _ = s.UploadBenchmark(ctx, "VendorB", "cloud", 25.0, 70000)

	// Compare a deal that is slightly above average discount and below median value
	result, err := s.CompareToBenchmark(ctx, 20.0, 60000, "cloud")
	if err != nil {
		t.Fatalf("CompareToBenchmark: %v", err)
	}

	if result["category"] != "cloud" {
		t.Errorf("expected category cloud, got %v", result["category"])
	}
	if result["my_discount_pct"] != 20.0 {
		t.Errorf("expected my_discount_pct 20.0, got %v", result["my_discount_pct"])
	}
	sampleCount, ok := result["sample_count"].(int)
	if !ok || sampleCount != 2 {
		t.Errorf("expected sample_count 2, got %v", result["sample_count"])
	}
	avgDisc, ok := result["avg_discount"].(float64)
	if !ok || math.Abs(avgDisc-20.0) > 0.01 {
		t.Errorf("expected avg_discount ~20.0, got %v", result["avg_discount"])
	}
	medianDeal, ok := result["median_deal"].(float64)
	if !ok || math.Abs(medianDeal-60000) > 0.01 {
		t.Errorf("expected median_deal ~60000, got %v", result["median_deal"])
	}
}

func TestCompareToBenchmark_NoData(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	result, err := s.CompareToBenchmark(ctx, 15.0, 50000, "unknown")
	if err != nil {
		t.Fatalf("CompareToBenchmark: %v", err)
	}

	if result["sample_count"] != 0 {
		t.Errorf("expected sample_count 0 for unknown category, got %v", result["sample_count"])
	}
	if result["message"] != "No benchmark data available for this category." {
		t.Errorf("unexpected message: %v", result["message"])
	}
}

func TestUploadMultipleEntries(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	titles := []string{"VendorA", "VendorB", "VendorC"}
	for _, vendor := range titles {
		_, err := s.UploadBenchmark(ctx, vendor, "general", 10.0, 10000)
		if err != nil {
			t.Fatalf("UploadBenchmark(%s): %v", vendor, err)
		}
	}

	benchmarks, err := s.GetBenchmarks(ctx, "")
	if err != nil {
		t.Fatalf("GetBenchmarks: %v", err)
	}
	if len(benchmarks) != 1 {
		t.Fatalf("expected 1 benchmark group, got %d", len(benchmarks))
	}
	if benchmarks[0].SampleCount != 3 {
		t.Errorf("expected 3 samples, got %d", benchmarks[0].SampleCount)
	}
}
