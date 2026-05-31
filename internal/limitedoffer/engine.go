package limitedoffer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Engine analyzes time-limited vendor offers.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a new limited offer engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Analyze evaluates a time-limited offer and returns a structured result.
func (e *Engine) Analyze(ctx context.Context, input *OfferInput) (*OfferResult, error) {
	// Calculate savings
	var savings float64
	if input.CurrentSpend > 0 {
		savings = input.CurrentSpend - input.OfferPrice
	} else {
		savings = input.CurrentPrice - input.OfferPrice
	}

	// Calculate days remaining
	now := time.Now().UTC()
	expires := input.ExpiresAt
	daysRemaining := expires.Sub(now).Hours() / 24
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Determine urgency
	urgency := "normal"
	if daysRemaining < 3 {
		urgency = "critical"
	} else if daysRemaining < 7 {
		urgency = "warning"
	}

	// Determine recommendation
	recommendation := "review"
	if savings > 0 {
		recommendation = "accept"
	} else if savings < 0 {
		recommendation = "pass"
	}

	// Query pricing store for list_price to calculate vs_best_price_pct
	vsBestPricePct := 0.0
	if input.Vendor != "" && input.SKU != "" {
		listPrice, err := e.queryListPrice(ctx, input.Vendor, input.SKU)
		if err != nil {
			e.logger.Warn("failed to query list price", "vendor", input.Vendor, "sku", input.SKU, "error", err)
		} else if listPrice > 0 {
			vsBestPricePct = math.Round((listPrice-input.OfferPrice)/listPrice*100*100) / 100
			if vsBestPricePct < 0 {
				vsBestPricePct = 0
			}
		}
	}

	return &OfferResult{
		Savings:        math.Round(savings*100) / 100,
		DaysRemaining:  math.Round(daysRemaining*100) / 100,
		Urgency:        urgency,
		Recommendation: recommendation,
		VsBestPricePct: vsBestPricePct,
	}, nil
}

// queryListPrice retrieves the list price for a vendor/SKU from the pricing store.
func (e *Engine) queryListPrice(ctx context.Context, vendor, sku string) (float64, error) {
	query := `
		SELECT ps.list_price
		FROM pricing_snapshot ps
		JOIN vendors v ON v.id = ps.vendor_id
		WHERE v.name = ? AND ps.sku = ?
		LIMIT 1
	`
	var listPrice float64
	err := e.db.QueryRowContext(ctx, query, vendor, sku).Scan(&listPrice)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no pricing found for vendor=%s sku=%s", vendor, sku)
	}
	if err != nil {
		return 0, fmt.Errorf("query list price: %w", err)
	}
	return listPrice, nil
}
