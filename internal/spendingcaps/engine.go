package spendingcaps

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"time"
)

// Engine manages spending cap enforcement and checks.
type Engine struct {
	store     *Store
	historyDB *sql.DB
	logger    *slog.Logger
}

// NewEngine creates a spendingcaps Engine.
func NewEngine(store *Store, historyDB *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{
		store:     store,
		historyDB: historyDB,
		logger:    logger,
	}
}

// SetCap sets or updates a spending cap for a vendor.
func (e *Engine) SetCap(ctx context.Context, vendor string, softCap, hardCap float64, period string) error {
	return e.store.SetCap(ctx, vendor, softCap, hardCap, period)
}

// CheckCaps checks all spending caps against current spend.
func (e *Engine) CheckCaps(ctx context.Context) ([]CapCheckResult, error) {
	caps, err := e.store.ListCaps(ctx)
	if err != nil {
		return nil, err
	}

	var results []CapCheckResult
	for _, cap := range caps {
		result, err := e.checkVendor(ctx, cap)
		if err != nil {
			e.logger.Warn("check cap: vendor check failed", "vendor", cap.Vendor, "error", err)
			continue
		}
		results = append(results, result)
	}

	if results == nil {
		results = []CapCheckResult{}
	}
	return results, nil
}

// CheckVendor checks a single vendor's spending cap.
func (e *Engine) CheckVendor(ctx context.Context, vendor string) (*CapCheckResult, error) {
	cap, err := e.store.GetCap(ctx, vendor)
	if err != nil {
		return nil, err
	}
	if cap == nil {
		return nil, nil
	}

	result, err := e.checkVendor(ctx, *cap)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *Engine) checkVendor(ctx context.Context, cap SpendingCap) (CapCheckResult, error) {
	var currentSpend float64

	switch cap.Period {
	case "monthly", "":
		currentMonth := time.Now().Format("2006-01")
		err := e.historyDB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(final_price), 0)
			FROM deal_outcomes
			WHERE vendor = ? AND strftime('%Y-%m', created_at) = ?
		`, cap.Vendor, currentMonth).Scan(&currentSpend)
		if err != nil {
			return CapCheckResult{}, err
		}
	case "quarterly":
		now := time.Now()
		qStart := quarterlyStart(now)
		qEnd := quarterlyEnd(now)
		err := e.historyDB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(final_price), 0)
			FROM deal_outcomes
			WHERE vendor = ? AND created_at >= ? AND created_at < ?
		`, cap.Vendor, qStart.Format(time.RFC3339), qEnd.Format(time.RFC3339)).Scan(&currentSpend)
		if err != nil {
			return CapCheckResult{}, err
		}
	case "yearly":
		year := time.Now().Format("2006")
		err := e.historyDB.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(final_price), 0)
			FROM deal_outcomes
			WHERE vendor = ? AND strftime('%Y', created_at) = ?
		`, cap.Vendor, year).Scan(&currentSpend)
		if err != nil {
			return CapCheckResult{}, err
		}
	}

	currentSpend = math.Round(currentSpend*100) / 100

	return CapCheckResult{
		Vendor:       cap.Vendor,
		SoftCap:      cap.SoftCap,
		HardCap:      cap.HardCap,
		CurrentSpend: currentSpend,
		SoftReached:  cap.SoftCap > 0 && currentSpend >= cap.SoftCap,
		HardReached:  cap.HardCap > 0 && currentSpend >= cap.HardCap,
		Period:       cap.Period,
	}, nil
}

func quarterlyStart(t time.Time) time.Time {
	q := (t.Month()-1)/3*3 + 1
	return time.Date(t.Year(), q, 1, 0, 0, 0, 0, t.Location())
}

func quarterlyEnd(t time.Time) time.Time {
	q := (t.Month()-1)/3*3 + 1
	return time.Date(t.Year(), q+3, 1, 0, 0, 0, 0, t.Location())
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
