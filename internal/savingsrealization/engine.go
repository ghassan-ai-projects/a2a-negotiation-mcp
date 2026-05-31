package savingsrealization

import (
	"context"
	"log/slog"
	"math"
	"sort"
)

// Engine manages savings realization tracking and reporting.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a savingsrealization Engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		logger: logger,
	}
}

// Record records a savings realization for a deal.
func (e *Engine) Record(ctx context.Context, sessionID, vendor string, projectedAmount, actualAmount float64, period string) (*SavingsRealization, error) {
	if period == "" {
		period = "monthly"
	}
	return e.store.Record(ctx, sessionID, vendor, projectedAmount, actualAmount, period)
}

// GetReport returns an aggregated realization report for the given period.
func (e *Engine) GetReport(ctx context.Context, period string) (RealizationReport, error) {
	if period == "" {
		period = "90d"
	}

	records, err := e.store.GetReport(ctx, period)
	if err != nil {
		return RealizationReport{}, err
	}

	if len(records) == 0 {
		return RealizationReport{
			ByVendor:      []VendorRealization{},
			TopShortfalls: []VendorRealization{},
		}, nil
	}

	var totalProjected, totalRealized float64
	vendorMap := make(map[string]*VendorRealization)

	for _, r := range records {
		totalProjected += r.ProjectedAmount
		totalRealized += r.ActualAmount

		if vr, ok := vendorMap[r.Vendor]; ok {
			vr.Projected += r.ProjectedAmount
			vr.Actual += r.ActualAmount
		} else {
			vendorMap[r.Vendor] = &VendorRealization{
				Vendor:    r.Vendor,
				Projected: r.ProjectedAmount,
				Actual:    r.ActualAmount,
			}
		}
	}

	totalProjected = math.Round(totalProjected*100) / 100
	totalRealized = math.Round(totalRealized*100) / 100

	var realizationRate float64
	if totalProjected > 0 {
		realizationRate = math.Round((totalRealized/totalProjected)*10000) / 100
	}

	byVendor := make([]VendorRealization, 0, len(vendorMap))
	var shortfalls []VendorRealization

	for _, vr := range vendorMap {
		vr.Projected = math.Round(vr.Projected*100) / 100
		vr.Actual = math.Round(vr.Actual*100) / 100
		if vr.Projected > 0 {
			vr.Rate = math.Round((vr.Actual/vr.Projected)*10000) / 100
		} else {
			vr.Rate = 0
		}
		byVendor = append(byVendor, *vr)

		if vr.Rate < 100 {
			shortfalls = append(shortfalls, *vr)
		}
	}

	sort.Slice(shortfalls, func(i, j int) bool {
		return shortfalls[i].Rate < shortfalls[j].Rate
	})

	// Top 5 shortfalls
	if len(shortfalls) > 5 {
		shortfalls = shortfalls[:5]
	}
	if shortfalls == nil {
		shortfalls = []VendorRealization{}
	}

	return RealizationReport{
		TotalProjected:  totalProjected,
		TotalRealized:   totalRealized,
		RealizationRate: realizationRate,
		ByVendor:        byVendor,
		TopShortfalls:   shortfalls,
	}, nil
}

// GetVendorReport returns a per-vendor breakdown of savings realization.
func (e *Engine) GetVendorReport(ctx context.Context, vendor string) ([]SavingsRealization, error) {
	records, err := e.store.ListByVendor(ctx, vendor)
	if err != nil {
		return nil, err
	}
	if records == nil {
		records = []SavingsRealization{}
	}
	return records, nil
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
