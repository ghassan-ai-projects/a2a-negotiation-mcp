package pricealerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// LatestPriceFn fetches the latest price for a vendor/SKU.
type LatestPriceFn func(ctx context.Context, vendor, sku string) (float64, error)

// Engine implements price drop alert logic.
type Engine struct {
	store       *Store
	getLatest   LatestPriceFn
	logger      *slog.Logger
}

// NewEngine creates a price alert engine.
func NewEngine(store *Store, getLatest LatestPriceFn, logger *slog.Logger) *Engine {
	return &Engine{
		store:       store,
		getLatest:   getLatest,
		logger:      logger,
	}
}

// Store returns the underlying store for MCP handlers.
func (e *Engine) Store() *Store {
	return e.store
}

// EnableAlert saves a price alert rule and records the current baseline price.
func (e *Engine) EnableAlert(ctx context.Context, vendor, sku string, thresholdPct float64, channel string) (*PriceAlertRule, error) {
	if thresholdPct <= 0 {
		thresholdPct = 10
	}
	if channel == "" {
		channel = "webhook"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rule := &PriceAlertRule{
		Vendor:       vendor,
		SKU:          sku,
		ThresholdPct: thresholdPct,
		Channel:      channel,
		Enabled:      true,
		CreatedAt:    now,
	}

	if err := e.store.SetRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("enable alert: %w", err)
	}

	price, err := e.getLatest(ctx, vendor, sku)
	if err != nil {
		e.logger.Warn("get current price for baseline", "vendor", vendor, "sku", sku, "error", err)
	} else if price > 0 {
		if err := e.store.SetBaseline(ctx, vendor, sku, price); err != nil {
			e.logger.Warn("set baseline", "vendor", vendor, "sku", sku, "error", err)
		}
	}

	return rule, nil
}

// CheckAlerts evaluates all enabled price alert rules against current prices.
func (e *Engine) CheckAlerts(ctx context.Context) ([]PriceAlertResult, error) {
	rules, err := e.store.ListRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("check alerts: list rules: %w", err)
	}

	var results []PriceAlertResult
	for _, rule := range rules {
		res, err := e.evaluateRule(ctx, rule)
		if err != nil {
			e.logger.Warn("evaluate rule", "vendor", rule.Vendor, "sku", rule.SKU, "error", err)
			continue
		}
		results = append(results, *res)

		if err := e.store.UpdateLastChecked(ctx, rule.Vendor, rule.SKU); err != nil {
			e.logger.Warn("update last_checked", "vendor", rule.Vendor, "sku", rule.SKU, "error", err)
		}
	}

	return results, nil
}

// DisableAlert removes a price alert rule.
func (e *Engine) DisableAlert(ctx context.Context, vendor, sku string) error {
	return e.store.DeleteRule(ctx, vendor, sku)
}

func (e *Engine) evaluateRule(ctx context.Context, rule PriceAlertRule) (*PriceAlertResult, error) {
	baseline, err := e.store.GetBaseline(ctx, rule.Vendor, rule.SKU)
	if err != nil {
		return nil, fmt.Errorf("get baseline: %w", err)
	}

	current, err := e.getLatest(ctx, rule.Vendor, rule.SKU)
	if err != nil {
		return nil, fmt.Errorf("get current price: %w", err)
	}

	res := &PriceAlertResult{
		Vendor:        rule.Vendor,
		SKU:           rule.SKU,
		PreviousPrice: baseline,
		CurrentPrice:  current,
		ThresholdMet:  false,
		AlertSent:     false,
	}

	if baseline > 0 && current > 0 {
		dropPct := ((baseline - current) / baseline) * 100
		res.DropPct = dropPct
		if dropPct >= rule.ThresholdPct {
			res.ThresholdMet = true
		}
	}

	return res, nil
}
