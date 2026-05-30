package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// HistoryStore is the interface the benchmark engine needs from the history store.
type HistoryStore interface {
	GetHistory(ctx context.Context, vendor string, period string) (*history.HistorySummary, error)
	GetSimilarDeals(ctx context.Context, vendor string, limit int) ([]history.DealOutcome, error)
}

// Engine generates benchmark reports comparing user savings against all users.
type Engine struct {
	historyStore HistoryStore
	logger       *slog.Logger
}

// NewEngine creates a benchmark engine.
func NewEngine(historyStore HistoryStore, logger *slog.Logger) *Engine {
	return &Engine{historyStore: historyStore, logger: logger}
}

// GenerateReport creates a benchmark report for a user filtered by vendor/category and period.
// It reads deal outcomes from the history store, calculates the user's metrics,
// compares against global data for percentile, and groups by vendor.
func (e *Engine) GenerateReport(ctx context.Context, userID string, vendor, category, period string) (*BenchmarkReport, error) {
	// Validate period
	switch period {
	case "30d", "90d", "1y", "all", "":
		if period == "" {
			period = "90d"
		}
	default:
		return nil, fmt.Errorf("invalid period %q: use 30d, 90d, 1y, or all", period)
	}

	// Get user's history for the filter
	histVendor := vendor
	if category != "" {
		// Category is not directly in history, so we get all and filter broader
		// For now, vendor and category are mutually exclusive filters over history
		histVendor = ""
	}
	userSummary, err := e.historyStore.GetHistory(ctx, histVendor, period)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	if userSummary == nil || len(userSummary.Deals) == 0 {
		return &BenchmarkReport{
			UserID: userID,
			Period: period,
		}, nil
	}

	// Get global history (no vendor filter) for percentile calculation
	globalSummary, err := e.historyStore.GetHistory(ctx, "", period)
	if err != nil {
		return nil, fmt.Errorf("get global history: %w", err)
	}

	// Calculate percentile: compare user's avg discount against all other users
	percentile := e.calcPercentile(userSummary, globalSummary)

	// Group by vendor
	vendorMap := make(map[string]*VendorSavings)
	for _, deal := range userSummary.Deals {
		vs, ok := vendorMap[deal.Vendor]
		if !ok {
			vs = &VendorSavings{Vendor: deal.Vendor}
			vendorMap[deal.Vendor] = vs
		}
		vs.Savings += deal.ListPrice - deal.FinalPrice
		vs.AvgDiscountPct += deal.DiscountPct
		vs.DealCount++
	}

	byVendor := make([]VendorSavings, 0, len(vendorMap))
	for _, vs := range vendorMap {
		if vs.DealCount > 0 {
			vs.AvgDiscountPct = math.Round(vs.AvgDiscountPct/float64(vs.DealCount)*100) / 100
		}
		vs.Savings = math.Round(vs.Savings*100) / 100
		byVendor = append(byVendor, *vs)
	}
	sort.Slice(byVendor, func(i, j int) bool {
		return byVendor[i].Savings > byVendor[j].Savings
	})

	report := &BenchmarkReport{
		UserID:         userID,
		Period:         period,
		TotalSavings:   math.Round(userSummary.TotalSavings*100) / 100,
		AvgDiscountPct: math.Round(userSummary.AvgDiscountPct*100) / 100,
		DealCount:      userSummary.TotalDeals,
		Percentile:     math.Round(percentile*100) / 100,
		ByVendor:       byVendor,
	}

	return report, nil
}

// calcPercentile computes the user's percentile rank based on avg discount.
// Returns 0-100 where higher = better.
func (e *Engine) calcPercentile(user *history.HistorySummary, global *history.HistorySummary) float64 {
	if global == nil || global.TotalDeals == 0 {
		return 50.0 // No global data = median
	}
	if user.TotalDeals == 0 {
		return 0
	}

	// We estimate percentile using a simplified approach:
	// compare the user's avg discount against what we know globally
	// If user has no savings, they're at the bottom
	if user.AvgDiscountPct <= 0 {
		return 0
	}

	// Rough estimate: if user is above global average, they're above median
	// Scale from 50-100 or 0-50 based on how they compare
	if user.AvgDiscountPct > global.AvgDiscountPct {
		// Above average: 50-100 range
		ratio := user.AvgDiscountPct / math.Max(global.AvgDiscountPct, 0.01)
		if ratio > 2 {
			return 95.0
		}
		return 50.0 + math.Min(ratio-1, 1.0)*45.0
	}

	// Below average: 0-50 range
	ratio := user.AvgDiscountPct / math.Max(global.AvgDiscountPct, 0.01)
	return math.Max(0, ratio*50.0)
}
