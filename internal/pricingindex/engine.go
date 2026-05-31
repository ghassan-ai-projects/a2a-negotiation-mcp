package pricingindex

import (
	"context"
	"fmt"
	"math"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

type rawRow struct {
	vendor, category               string
	listPrice, minObs, maxObs, typicalPct float64
}

// Engine computes competitive pricing indices from the pricing store.
type Engine struct {
	pricingStore *pricing.Store
}

// NewEngine creates a pricing index engine.
func NewEngine(pricingStore *pricing.Store) *Engine {
	return &Engine{pricingStore: pricingStore}
}

// Index builds a PricingIndex filtered by optional category and vendor.
func (e *Engine) Index(ctx context.Context, category, vendor string) (*PricingIndex, error) {
	db := e.pricingStore.DB()
	period := "current"

	query := `
		SELECT v.name, v.category, ps.list_price, ps.min_observed, ps.max_observed, ps.typical_pct
		FROM pricing_snapshot ps
		JOIN vendors v ON v.id = ps.vendor_id
		WHERE 1=1`
	var args []any

	if category != "" {
		query += " AND v.category = ?"
		args = append(args, category)
	}
	if vendor != "" {
		query += " AND v.name = ?"
		args = append(args, vendor)
	}
	query += " ORDER BY v.name, ps.sku"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pricing index: %w", err)
	}
	defer rows.Close()

	var raw []rawRow
	for rows.Next() {
		var r rawRow
		if err := rows.Scan(&r.vendor, &r.category, &r.listPrice, &r.minObs, &r.maxObs, &r.typicalPct); err != nil {
			return nil, err
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("no pricing data found")
	}

	return buildIndex(raw, category, period), nil
}

func buildIndex(raw []rawRow, category, period string) *PricingIndex {
	vendorMap := make(map[string]*vendorAccum)
	for _, r := range raw {
		va, ok := vendorMap[r.vendor]
		if !ok {
			va = &vendorAccum{category: r.category}
			vendorMap[r.vendor] = va
		}
		va.prices = append(va.prices, r.listPrice)
		va.minObs = append(va.minObs, r.minObs)
		va.maxObs = append(va.maxObs, r.maxObs)
		va.count++
	}

	minPrice := math.MaxFloat64
	maxPrice := 0.0
	var allPrices []float64
	effectiveCategory := category

	var vendors []VendorIndex
	for name, va := range vendorMap {
		avg := mean(va.prices)
		allPrices = append(allPrices, va.prices...)

		if avg < minPrice {
			minPrice = avg
		}
		if avg > maxPrice {
			maxPrice = avg
		}

		if va.category != "" && effectiveCategory == "" {
			effectiveCategory = va.category
		}

		trend := "stable"
		if len(va.prices) > 1 {
			first := va.prices[0]
			last := va.prices[len(va.prices)-1]
			if last > first*1.01 {
				trend = "up"
			} else if last < first*0.99 {
				trend = "down"
			}
		}

		vendors = append(vendors, VendorIndex{
			Vendor:     name,
			AvgPrice:   avg,
			Category:   va.category,
			PriceTrend: trend,
		})
	}

	if minPrice == math.MaxFloat64 {
		minPrice = 0
	}

	avgPrice := mean(allPrices)

	momPct := 0.0
	if len(vendors) > 1 && avgPrice > 0 {
		momPct = ((maxPrice - minPrice) / avgPrice) * 100
	}

	volatilityIdx := 0.0
	if len(allPrices) > 1 && avgPrice > 0 {
		variance := 0.0
		for _, p := range allPrices {
			diff := p - avgPrice
			variance += diff * diff
		}
		variance /= float64(len(allPrices))
		stddev := math.Sqrt(variance)
		volatilityIdx = (stddev / avgPrice) * 100
	}

	return &PricingIndex{
		Category:      effectiveCategory,
		Period:        period,
		AvgPrice:      avgPrice,
		PriceRange:    PriceRange{Min: minPrice, Max: maxPrice},
		Vendors:       vendors,
		MoMChangePct:  math.Round(momPct*100) / 100,
		VolatilityIdx: math.Round(volatilityIdx*100) / 100,
	}
}

type vendorAccum struct {
	category string
	prices   []float64
	minObs   []float64
	maxObs   []float64
	count    int
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
