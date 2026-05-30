package roi

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Engine computes ROI analytics for negotiated deals.
type Engine struct {
	store *Store
}

// NewEngine creates a new ROI engine.
func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

// Calculate computes the full ROI calculation from deal parameters.
//
// Formulas:
//
//	AnnualSavings      = CurrentSpend - NegotiatedPrice
//	ROIPct             = (AnnualSavings - AnnualOverhead - ImplementationCosts/3) / (ImplementationCosts + AnnualOverhead) * 100
//	PaybackMonths      = (ImplementationCosts + AnnualOverhead) / (AnnualSavings / 12)
//	Savings1Y          = AnnualSavings - AnnualOverhead - ImplementationCosts
//	Savings3Y          = (AnnualSavings - AnnualOverhead) * 3 - ImplementationCosts
//	Savings5Y          = (AnnualSavings - AnnualOverhead) * 5 - ImplementationCosts
//	NPV                = sum_{y=1..5} (AnnualSavings - AnnualOverhead) / (1 + 0.08)^y - ImplementationCosts
func (e *Engine) Calculate(ctx context.Context, currentSpend, negotiatedPrice, implementationCosts, annualOverhead float64) (*ROICalculation, error) {
	if currentSpend <= 0 || negotiatedPrice <= 0 {
		return nil, fmt.Errorf("current_spend and negotiated_price must be positive")
	}
	if negotiatedPrice > currentSpend {
		return nil, fmt.Errorf("negotiated_price cannot exceed current_spend")
	}

	annualSavings := currentSpend - negotiatedPrice
	netAnnualSavings := annualSavings - annualOverhead

	totalInvestment := implementationCosts + annualOverhead

	// ROIPct
	var roiPct float64
	if totalInvestment > 0 {
		roiPct = (netAnnualSavings - implementationCosts/3) / totalInvestment * 100
	} else {
		roiPct = 0
	}

	// PaybackMonths
	var paybackMonths float64
	if annualSavings > 0 {
		paybackMonths = totalInvestment / (annualSavings / 12)
	} else {
		paybackMonths = 0
	}

	// Savings by year
	savings1Y := netAnnualSavings - implementationCosts
	savings3Y := netAnnualSavings*3 - implementationCosts
	savings5Y := netAnnualSavings*5 - implementationCosts

	// NPV: discounted at 8% over 5 years
	discountRate := 0.08
	npv := -implementationCosts
	for y := 1; y <= 5; y++ {
		npv += netAnnualSavings / math.Pow(1+discountRate, float64(y))
	}

	calc := &ROICalculation{
		CurrentSpend:        currentSpend,
		NegotiatedPrice:     negotiatedPrice,
		ImplementationCosts: implementationCosts,
		AnnualOverhead:      annualOverhead,
		AnnualSavings:       annualSavings,
		ROIPct:              math.Round(roiPct*100) / 100,
		PaybackMonths:       math.Round(paybackMonths*100) / 100,
		Savings1Y:           math.Round(savings1Y*100) / 100,
		Savings3Y:           math.Round(savings3Y*100) / 100,
		Savings5Y:           math.Round(savings5Y*100) / 100,
		NPV:                 math.Round(npv*100) / 100,
		CreatedAt:           time.Now().UTC(),
	}

	return calc, nil
}

// Save persists a calculation (with optional user ID) and returns it with the ID populated.
func (e *Engine) Save(ctx context.Context, calc *ROICalculation, userID string) (*ROICalculation, error) {
	calc.UserID = userID
	if calc.CreatedAt.IsZero() {
		calc.CreatedAt = time.Now().UTC()
	}
	if err := e.store.Save(ctx, calc); err != nil {
		return nil, fmt.Errorf("save roi: %w", err)
	}
	return calc, nil
}

// ListByUser returns all calculations for a user.
func (e *Engine) ListByUser(ctx context.Context, userID string) ([]ROICalculation, error) {
	return e.store.ListByUser(ctx, userID)
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
