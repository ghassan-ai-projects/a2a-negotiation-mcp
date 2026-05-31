package alerthistory

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
)

// Engine provides a merged alert feed from multiple sources.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates an Engine that reads from the shared DB.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// GetAlerts returns a merged, sorted, and optionally filtered feed of alerts.
// typeFilter filters by alert_type ("budget", "renewal", "price_change", or "all").
func (e *Engine) GetAlerts(ctx context.Context, typeFilter, vendorFilter string, limit int) (*AlertFeed, error) {
	if limit <= 0 {
		limit = 50
	}

	var entries []AlertEntry

	// 1. Budget alerts from budget_alert_history
	if typeFilter == "all" || typeFilter == "budget" {
		budgetAlerts, err := e.queryBudgetAlerts(ctx, vendorFilter)
		if err != nil {
			e.logger.Warn("alerthistory: budget_alerts query failed", "error", err)
		} else {
			entries = append(entries, budgetAlerts...)
		}
	}

	// 2. Renewal alerts from contracts
	if typeFilter == "all" || typeFilter == "renewal" {
		renewalAlerts, err := e.queryRenewalAlerts(ctx, vendorFilter)
		if err != nil {
			e.logger.Warn("alerthistory: renewal_alerts query failed", "error", err)
		} else {
			entries = append(entries, renewalAlerts...)
		}
	}

	// 3. Price change alerts from price_snapshots
	if typeFilter == "all" || typeFilter == "price_change" {
		priceAlerts, err := e.queryPriceChangeAlerts(ctx, vendorFilter)
		if err != nil {
			e.logger.Warn("alerthistory: price_change query failed", "error", err)
		} else {
			entries = append(entries, priceAlerts...)
		}
	}

	// Sort by created_at DESC
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt > entries[j].CreatedAt
	})

	// Apply limit
	if len(entries) > limit {
		entries = entries[:limit]
	}

	// Group by alert_type
	grouped := make(map[string][]AlertEntry)
	for _, e := range entries {
		grouped[e.AlertType] = append(grouped[e.AlertType], e)
	}

	return &AlertFeed{
		Entries: entries,
		Grouped: grouped,
	}, nil
}

func (e *Engine) queryBudgetAlerts(ctx context.Context, vendorFilter string) ([]AlertEntry, error) {
	query := `SELECT id, vendor, consumed_pct, level, created_at FROM budget_alert_history`
	args := []any{}
	if vendorFilter != "" {
		query += " WHERE vendor = ?"
		args = append(args, vendorFilter)
	}
	query += " ORDER BY created_at DESC"

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query budget alerts: %w", err)
	}
	defer rows.Close()

	var alerts []AlertEntry
	for rows.Next() {
		var id int64
		var vendor, level, createdAt string
		var consumedPct float64
		if err := rows.Scan(&id, &vendor, &consumedPct, &level, &createdAt); err != nil {
			return nil, fmt.Errorf("scan budget alert: %w", err)
		}
		alerts = append(alerts, AlertEntry{
			ID:        id,
			AlertType: "budget",
			Vendor:    vendor,
			Message:   fmt.Sprintf("Budget consumed %.1f%%", consumedPct),
			Level:     level,
			CreatedAt: createdAt,
		})
	}
	return alerts, rows.Err()
}

func (e *Engine) queryRenewalAlerts(ctx context.Context, vendorFilter string) ([]AlertEntry, error) {
	query := `SELECT id, vendor, renewal_date FROM contracts WHERE status = 'active'`
	args := []any{}
	if vendorFilter != "" {
		query += " AND vendor = ?"
		args = append(args, vendorFilter)
	}
	query += " ORDER BY renewal_date ASC"

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query renewal alerts: %w", err)
	}
	defer rows.Close()

	var alerts []AlertEntry
	var seq int64
	for rows.Next() {
		var id int64
		var vendor, renewalDate string
		if err := rows.Scan(&id, &vendor, &renewalDate); err != nil {
			return nil, fmt.Errorf("scan renewal alert: %w", err)
		}
		seq++

		level := "info"
		message := fmt.Sprintf("Contract renewal approaching: %s", renewalDate)
		alerts = append(alerts, AlertEntry{
			ID:        seq,
			AlertType: "renewal",
			Vendor:    vendor,
			Message:   message,
			Level:     level,
			CreatedAt: renewalDate,
		})
	}
	return alerts, rows.Err()
}

func (e *Engine) queryPriceChangeAlerts(ctx context.Context, vendorFilter string) ([]AlertEntry, error) {
	query := `SELECT id, vendor, sku, price, list_price, date, created_at FROM price_snapshots ORDER BY created_at DESC`
	args := []any{}
	if vendorFilter != "" {
		query = `SELECT id, vendor, sku, price, list_price, date, created_at FROM price_snapshots WHERE vendor = ? ORDER BY created_at DESC`
		args = append(args, vendorFilter)
	}

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query price change alerts: %w", err)
	}
	defer rows.Close()

	var alerts []AlertEntry
	for rows.Next() {
		var id int64
		var vendor, sku, date, createdAt string
		var price, listPrice float64
		if err := rows.Scan(&id, &vendor, &sku, &price, &listPrice, &date, &createdAt); err != nil {
			return nil, fmt.Errorf("scan price change: %w", err)
		}
		message := fmt.Sprintf("Price snapshot for %s/%s: $%.2f (list $%.2f)", vendor, sku, price, listPrice)
		level := "info"
		if listPrice > 0 && price > listPrice {
			level = "warning"
		}
		alerts = append(alerts, AlertEntry{
			ID:        id,
			AlertType: "price_change",
			Vendor:    vendor,
			Message:   message,
			Level:     level,
			CreatedAt: createdAt,
		})
	}
	return alerts, rows.Err()
}
