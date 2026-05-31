package strategycomparison

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
)

// Engine provides strategy comparison simulations.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a strategycomparison Engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Compare simulates outcomes for multiple negotiation strategies.
func (e *Engine) Compare(ctx context.Context, req StrategyComparisonRequest) (StrategyComparisonResult, error) {
	if req.Vendor == "" {
		return StrategyComparisonResult{}, fmt.Errorf("vendor is required")
	}

	if len(req.Strategies) == 0 {
		req.Strategies = []string{"aggressive", "balanced", "conservative"}
	}

	// Look up list price from pricing store
	var listPrice float64
	err := e.db.QueryRowContext(ctx, `
		SELECT p.list_price
		FROM pricing_snapshot p
		JOIN vendors v ON v.id = p.vendor_id
		WHERE v.name = ? AND (p.sku = ? OR ? = '')
		ORDER BY p.list_price ASC
		LIMIT 1
	`, req.Vendor, req.SKU, req.SKU).Scan(&listPrice)
	if err == sql.ErrNoRows {
		// Default if not found in pricing
		listPrice = req.Budget
		if listPrice <= 0 {
			listPrice = 100.0
		}
	} else if err != nil {
		return StrategyComparisonResult{}, fmt.Errorf("query list price: %w", err)
	}

	var results []StrategyResult
	for _, strategy := range req.Strategies {
		result, err := simulateStrategy(strategy, listPrice)
		if err != nil {
			return StrategyComparisonResult{}, err
		}
		results = append(results, result)
	}

	// Determine best strategy (highest expected savings)
	bestStrategy := ""
	bestSavings := -1.0
	for _, r := range results {
		if r.ExpectedSavings > bestSavings {
			bestSavings = r.ExpectedSavings
			bestStrategy = r.Strategy
		}
	}

	if results == nil {
		results = []StrategyResult{}
	}

	return StrategyComparisonResult{
		Vendor:       req.Vendor,
		SKU:          req.SKU,
		Budget:       req.Budget,
		Results:      results,
		BestStrategy: bestStrategy,
	}, nil
}

func simulateStrategy(strategy string, listPrice float64) (StrategyResult, error) {
	var discountPct float64
	var riskLevel string
	var outcome string

	switch strategy {
	case "aggressive":
		discountPct = 0.30
		riskLevel = "high"
		outcome = "Push for 30% discount — may risk deal if vendor is inflexible"
	case "balanced":
		discountPct = 0.20
		riskLevel = "medium"
		outcome = "Target 20% discount — reasonable ask with good win probability"
	case "conservative":
		discountPct = 0.10
		riskLevel = "low"
		outcome = "Aim for 10% discount — low risk, high acceptance rate"
	default:
		return StrategyResult{}, fmt.Errorf("unknown strategy: %s", strategy)
	}

	savings := listPrice * discountPct

	return StrategyResult{
		Strategy:        strategy,
		LikelyOutcome:   outcome,
		ExpectedSavings: math.Round(savings*100) / 100,
		RiskLevel:       riskLevel,
	}, nil
}
