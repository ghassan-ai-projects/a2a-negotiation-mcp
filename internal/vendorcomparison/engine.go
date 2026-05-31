package vendorcomparison

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sort"
)

// Engine provides multi-vendor comparison analysis.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a vendorcomparison Engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Compare runs a multi-vendor comparison for a given category.
func (e *Engine) Compare(ctx context.Context, req ComparisonRequest) (ComparisonResult, error) {
	if req.Category == "" {
		return ComparisonResult{}, fmt.Errorf("category is required")
	}
	if req.Seats <= 0 {
		req.Seats = 50
	}

	rows, err := e.db.QueryContext(ctx, `
		SELECT v.name, p.sku, v.category, p.list_price, p.min_observed, p.max_observed, p.typical_pct
		FROM pricing_snapshot p
		JOIN vendors v ON v.id = p.vendor_id
		WHERE v.category = ?
		ORDER BY p.list_price ASC
	`, req.Category)
	if err != nil {
		return ComparisonResult{}, fmt.Errorf("query vendor comparison: %w", err)
	}
	defer rows.Close()

	var comparisons []VendorComparison
	totalPrice := 0.0

	for rows.Next() {
		var vc VendorComparison
		var minObs, maxObs, typicalPct float64
		if err := rows.Scan(&vc.Vendor, &vc.SKU, &vc.Category, &vc.ListPrice, &minObs, &maxObs, &typicalPct); err != nil {
			return ComparisonResult{}, fmt.Errorf("scan comparison row: %w", err)
		}

		typicalDiscount := typicalPct / 100.0
		vc.TypicalPrice = vc.ListPrice * (1 - typicalDiscount)

		// Compute annual cost: list_price * seats * 12
		vc.AnnualCost = vc.ListPrice * float64(req.Seats) * 12

		// Savings potential: list_price - typical_price
		vc.SavingsPotential = vc.ListPrice - vc.TypicalPrice

		// Score: higher for lower price-to-value ratio (simple heuristic)
		if vc.ListPrice > 0 {
			vc.Score = int(math.Round((1 - (vc.TypicalPrice / vc.ListPrice)) * 100))
		}

		comparisons = append(comparisons, vc)
		totalPrice += vc.ListPrice
	}
	if err := rows.Err(); err != nil {
		return ComparisonResult{}, fmt.Errorf("rows iteration: %w", err)
	}

	if len(comparisons) == 0 {
		return ComparisonResult{
			Category:    req.Category,
			Comparisons: []VendorComparison{},
		}, nil
	}

	// Compute average price
	avgPrice := math.Round(totalPrice/float64(len(comparisons))*100) / 100

	// Determine top pick: lowest annual cost
	topPick := ""
	lowestCost := math.MaxFloat64
	for _, c := range comparisons {
		if c.AnnualCost < lowestCost {
			lowestCost = c.AnnualCost
			topPick = c.Vendor
		}
	}

	// Ensure nil-safe
	if comparisons == nil {
		comparisons = []VendorComparison{}
	}

	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].AnnualCost < comparisons[j].AnnualCost
	})

	return ComparisonResult{
		Category:    req.Category,
		Comparisons: comparisons,
		TopPick:     topPick,
		AvgPrice:    avgPrice,
	}, nil
}
