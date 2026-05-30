package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Engine computes vendor health scores and negotiation leverage.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a new health engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// CalculateScore computes health score from signals.
// Base score: 50 (neutral).
// Each signal adds/subtracts its weight.
// Layoff = -15, Lawsuit = -20, Funding = +10, Growth = +15, IPO = +20, Acquisition = +5.
// Clamped to 1-100 range.
func (e *Engine) CalculateScore(ctx context.Context, vendor string) (*VendorHealth, error) {
	signals, err := e.store.GetSignals(ctx, vendor)
	if err != nil {
		return nil, fmt.Errorf("get signals: %w", err)
	}

	score := 50 // neutral baseline
	for _, sig := range signals {
		score += sig.Weight
	}

	// Clamp to 1-100
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}

	category := categorize(score)

	vh := &VendorHealth{
		Vendor:      vendor,
		Score:       score,
		Category:    category,
		Signals:     signals,
		LastUpdated: time.Now().UTC(),
	}

	if err := e.store.UpsertHealth(ctx, vh); err != nil {
		return nil, fmt.Errorf("upsert health: %w", err)
	}

	return vh, nil
}

// GetLeverage returns negotiation advice based on vendor health.
func (e *Engine) GetLeverage(ctx context.Context, vendor string) (*NegotiationLeverage, error) {
	vh, err := e.store.GetHealth(ctx, vendor)
	if err != nil {
		return nil, fmt.Errorf("get health: %w", err)
	}
	if vh == nil {
		// No data yet — compute a fresh score
		vh, err = e.CalculateScore(ctx, vendor)
		if err != nil {
			return nil, fmt.Errorf("calculate score: %w", err)
		}
	}

	leverage, suggestion := deriveLeverage(vh)

	return &NegotiationLeverage{
		Vendor:     vendor,
		Health:     *vh,
		Leverage:   leverage,
		Suggestion: suggestion,
	}, nil
}

// RecordSignal adds a signal and recalculates the vendor's health score.
func (e *Engine) RecordSignal(ctx context.Context, vendor, signalType, source, detail string, weight int) error {
	if weight == 0 {
		// Use default weight for the signal type if weight is unspecified
		if w, ok := SignalTypeWeights[signalType]; ok {
			weight = w
		}
	}

	if err := e.store.AddSignal(ctx, vendor, signalType, source, detail, weight); err != nil {
		return fmt.Errorf("add signal: %w", err)
	}

	// Recalculate and persist
	if _, err := e.CalculateScore(ctx, vendor); err != nil {
		return fmt.Errorf("recalculate score: %w", err)
	}

	return nil
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}

func categorize(score int) string {
	switch {
	case score < 30:
		return "struggling"
	case score > 60:
		return "growing"
	default:
		return "stable"
	}
}

func deriveLeverage(vh *VendorHealth) (leverage, suggestion string) {
	switch {
	case vh.Score < 30:
		leverage = "high"
		suggestion = "Push hard — "
		if hasSignalOfType(vh.Signals, "layoff") {
			suggestion += "they just had layoffs"
		} else if hasSignalOfType(vh.Signals, "lawsuit") {
			suggestion += "they are facing legal challenges"
		} else {
			suggestion += "vendor is struggling"
		}
	case vh.Score <= 60:
		leverage = "medium"
		suggestion = "Standard approach — vendor is stable, negotiate with moderate targets"
	default:
		leverage = "low"
		suggestion = "Vendor is growing — "
		if hasSignalOfType(vh.Signals, "ipo") {
			suggestion += "recent IPO gives them strong position"
		} else if hasSignalOfType(vh.Signals, "growth") {
			suggestion += "strong growth reduces discount likelihood"
		} else {
			suggestion += "expect less room for discounts"
		}
	}
	return
}

func hasSignalOfType(signals []Signal, signalType string) bool {
	for _, sig := range signals {
		if sig.Type == signalType {
			return true
		}
	}
	return false
}
