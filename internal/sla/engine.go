package sla

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const (
	totalMonthlyMins = 43200 // 30 days * 24 hours * 60 minutes
)

// Engine provides SLA breach tracking and auto-filing logic.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a new SLA engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		logger: logger,
	}
}

// AddContract registers an SLA contract.
func (e *Engine) AddContract(ctx context.Context, vendor, service string, uptimePct, creditPct, maxCreditPct, monthlySpend float64) (*SLAContract, error) {
	c := &SLAContract{
		Vendor:       vendor,
		Service:      service,
		UptimePct:    uptimePct,
		CreditPct:    creditPct,
		MaxCreditPct: maxCreditPct,
		MonthlySpend: monthlySpend,
		Status:       "active",
	}
	if err := e.store.AddContract(ctx, c); err != nil {
		return nil, fmt.Errorf("add contract: %w", err)
	}
	e.logger.Info("SLA contract added", "contract_id", c.ID, "vendor", vendor, "service", service)
	return c, nil
}

// RecordBreach logs an SLA breach and calculates the credit due.
//
// Credit calculation:
//
//	base_credit = monthly_spend * (credit_pct / 100)
//	ratio = min(duration_mins / total_monthly_mins, max_credit_pct / 100)
//	credit_due = base_credit * ratio
func (e *Engine) RecordBreach(ctx context.Context, vendor, service string, date time.Time, durationMins int) (*SLABreach, error) {
	// Find the active contract for this vendor/service
	contracts, err := e.store.ListContracts(ctx, "active")
	if err != nil {
		return nil, fmt.Errorf("list active contracts: %w", err)
	}

	var matched *SLAContract
	for i := range contracts {
		if contracts[i].Vendor == vendor && contracts[i].Service == service {
			matched = &contracts[i]
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("no active SLA contract found for %s/%s", vendor, service)
	}

	creditDue := calculateCredit(matched.MonthlySpend, matched.CreditPct, matched.MaxCreditPct, durationMins)

	b := &SLABreach{
		Vendor:       vendor,
		Service:      service,
		Date:         date,
		DurationMins: durationMins,
		CreditDue:    creditDue,
	}
	if err := e.store.AddBreach(ctx, b); err != nil {
		return nil, fmt.Errorf("record breach: %w", err)
	}

	e.logger.Info("SLA breach recorded",
		"breach_id", b.ID, "vendor", vendor, "service", service,
		"duration_mins", durationMins, "credit_due", creditDue)
	return b, nil
}

// FileClaim marks a breach as filed, simulating claim submission filing.
func (e *Engine) FileClaim(ctx context.Context, breachID string) (*SLABreach, error) {
	breach, err := e.store.GetBreach(ctx, breachID)
	if err != nil {
		return nil, fmt.Errorf("get breach: %w", err)
	}
	if breach.Filed {
		return nil, fmt.Errorf("breach %s already filed", breachID)
	}

	// Payout is the full credit due (simulates full payout on filing)
	if err := e.store.FileBreach(ctx, breachID, breach.CreditDue); err != nil {
		return nil, fmt.Errorf("file breach: %w", err)
	}

	breach.Filed = true
	breach.FiledAt = time.Now().UTC()
	breach.Payout = breach.CreditDue

	e.logger.Info("SLA claim filed", "breach_id", breachID, "payout", breach.Payout)
	return breach, nil
}

// GetReport returns all active contracts and their breaches for the given month,
// along with aggregate credit totals.
func (e *Engine) GetReport(ctx context.Context, month time.Time) (*SLAResult, error) {
	// Determine the month boundaries
	year, m := month.Year(), month.Month()
	startOfMonth := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	contracts, err := e.store.ListContracts(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}

	// Aggregate across all contracts (the spec expects a single contract result,
	// but we'll return the last in the list for backward compat; the total agg is correct)
	var aggregated SLAResult
	var totalCredits float64
	var filedCount int

	for i := range contracts {
		breaches, err := e.store.ListBreaches(ctx, contracts[i].Vendor, startOfMonth, endOfMonth)
		if err != nil {
			return nil, fmt.Errorf("list breaches for %s: %w", contracts[i].Vendor, err)
		}

		// Filter breaches for this service
		var serviceBreaches []SLABreach
		for _, b := range breaches {
			if b.Service == contracts[i].Service {
				serviceBreaches = append(serviceBreaches, b)
			}
		}

		for _, b := range serviceBreaches {
			totalCredits += b.CreditDue
			if b.Filed {
				filedCount++
			}
		}

		// Return result for the last active contract (keeps the API simple)
		if contracts[i].Status == "active" {
			aggregated = SLAResult{
				Contract:     contracts[i],
				Breaches:     serviceBreaches,
				TotalCredits: math.Round(totalCredits*100) / 100,
				FiledCount:   filedCount,
			}
		}
	}

	if aggregated.Contract.ID == "" && len(contracts) > 0 {
		// Fallback: no active contract, return the first one
		aggregated.Contract = contracts[0]
	}

	return &aggregated, nil
}

// calculateCredit computes the SLA credit due for a given breach duration.
//
//	base_credit = monthly_spend * (credit_pct / 100)
//	ratio = min(duration_mins / total_monthly_mins, max_credit_pct / 100)
//	credit_due = base_credit * ratio
func calculateCredit(monthlySpend, creditPct, maxCreditPct float64, durationMins int) float64 {
	baseCredit := monthlySpend * (creditPct / 100.0)
	ratio := float64(durationMins) / totalMonthlyMins
	maxRatio := maxCreditPct / 100.0
	if ratio > maxRatio {
		ratio = maxRatio
	}
	credit := baseCredit * ratio
	return math.Round(credit*100) / 100
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
