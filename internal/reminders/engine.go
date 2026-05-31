package reminders

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ExpiringContractsFn returns contracts expiring within the given days.
type ExpiringContractsFn func(ctx context.Context, daysAhead int) ([]ContractRow, error)

// ContractRow represents a contract row from the calendar store.
type ContractRow struct {
	ID          string
	Vendor      string
	SKU         string
	RenewalDate string
}

// Engine implements renewal reminder logic (read-only).
type Engine struct {
	contractsFn ExpiringContractsFn
	logger      *slog.Logger
}

// NewEngine creates a renewal reminders engine.
func NewEngine(contractsFn ExpiringContractsFn, logger *slog.Logger) *Engine {
	return &Engine{
		contractsFn: contractsFn,
		logger:      logger,
	}
}

// CheckRenewals reads upcoming contracts and categorizes them by urgency.
func (e *Engine) CheckRenewals(ctx context.Context) (*RenewalCheckResult, error) {
	contracts, err := e.contractsFn(ctx, 60)
	if err != nil {
		return nil, fmt.Errorf("check renewals: %w", err)
	}

	result := &RenewalCheckResult{}
	now := time.Now()

	for _, c := range contracts {
		renewalDate, err := time.Parse(time.DateOnly, c.RenewalDate)
		if err != nil {
			e.logger.Warn("parse renewal date", "contract_id", c.ID, "date", c.RenewalDate, "error", err)
			continue
		}

		daysUntil := int(renewalDate.Sub(now).Hours() / 24)
		if daysUntil < 0 {
			daysUntil = 0
		}

		item := RenewalItem{
			ContractID:    c.ID,
			Vendor:        c.Vendor,
			SKU:           c.SKU,
			RenewalDate:   c.RenewalDate,
			DaysUntil:     daysUntil,
			AutoNegotiate: false,
			Notified:      false,
		}

		switch {
		case daysUntil < 7:
			result.Critical = append(result.Critical, item)
		case daysUntil < 30:
			result.Soon = append(result.Soon, item)
		case daysUntil < 60:
			result.Upcoming = append(result.Upcoming, item)
		}
	}

	return result, nil
}
