package tco

import (
	"context"
	"fmt"
	"math"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// Engine calculates total cost of ownership for SaaS products.
type Engine struct {
	pricingStore *pricing.Store
}

// NewEngine creates a TCO engine.
func NewEngine(pricingStore *pricing.Store) *Engine {
	return &Engine{pricingStore: pricingStore}
}

// Calculate computes the TCO for a given vendor/SKU.
func (e *Engine) Calculate(ctx context.Context, input TCOInput) (*TCOOutput, error) {
	if input.Seats <= 0 {
		input.Seats = 50
	}
	if input.TermMonths <= 0 {
		input.TermMonths = 12
	}

	// Get pricing from store
	price, err := e.pricingStore.GetPricingByVendorSKU(ctx, input.Vendor, input.SKU)
	if err != nil {
		return nil, fmt.Errorf("get pricing: %w", err)
	}

	listPrice := price.ListPrice
	seats := float64(input.Seats)

	// Core calculations
	annualSubscription := listPrice * seats * 12
	perUnitCost := listPrice
	total1YTCO := annualSubscription + input.ImplementationCosts + input.TrainingCosts + input.SupportCosts
	total3YTCO := total1YTCO*3 - input.ImplementationCosts // non-recurring
	costPerUserPerMonth := total1YTCO / seats / 12

	// Market average CUPM (cost per user per month)
	// Using typical_pct as the discount rate off list price
	typicalDiscount := price.TypicalPct / 100.0
	marketAvgCUPM := listPrice * (1 - typicalDiscount)

	// Savings vs market
	var savingsVsMarketPct float64
	if marketAvgCUPM > 0 {
		savingsVsMarketPct = (marketAvgCUPM - costPerUserPerMonth) / marketAvgCUPM * 100
		savingsVsMarketPct = math.Round(savingsVsMarketPct*100) / 100
	}

	// Hidden costs flagged
	var hiddenCosts []string
	if input.ImplementationCosts > 0 {
		hiddenCosts = append(hiddenCosts, "Implementation costs")
	}
	if input.TrainingCosts > 0 {
		hiddenCosts = append(hiddenCosts, "Training costs")
	}
	if input.SupportCosts > 0 {
		hiddenCosts = append(hiddenCosts, "Support costs")
	}
	if hiddenCosts == nil {
		hiddenCosts = []string{}
	}

	return &TCOOutput{
		Vendor:              input.Vendor,
		SKU:                 input.SKU,
		Seats:               input.Seats,
		TermMonths:          input.TermMonths,
		PerUnitCost:         math.Round(perUnitCost*100) / 100,
		AnnualSubscription:  math.Round(annualSubscription*100) / 100,
		Total1YTCO:          math.Round(total1YTCO*100) / 100,
		Total3YTCO:          math.Round(total3YTCO*100) / 100,
		CostPerUserPerMonth: math.Round(costPerUserPerMonth*100) / 100,
		MarketAvgCUPM:       math.Round(marketAvgCUPM*100) / 100,
		SavingsVsMarketPct:  savingsVsMarketPct,
		HiddenCostsFlagged:  hiddenCosts,
	}, nil
}
