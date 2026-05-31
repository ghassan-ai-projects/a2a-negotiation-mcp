package pricingrefresh

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/trends"
)

// Engine handles bulk vendor pricing refresh operations.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a new pricing refresh engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Refresh creates fresh price snapshots for the given vendors (or all vendors if empty)
// in the trends store with a ±3% variation from the current list price.
func (e *Engine) Refresh(ctx context.Context, vendors []string, trendsStore *trends.Store) (*RefreshResult, error) {
	start := time.Now()

	// Query distinct vendors from the pricing store
	query := `
		SELECT DISTINCT v.name
		FROM vendors v
		JOIN pricing_snapshot ps ON ps.vendor_id = v.id
	`
	var args []any
	if len(vendors) > 0 {
		query += " WHERE v.name IN (?"
		for i := 1; i < len(vendors); i++ {
			query += ", ?"
		}
		query += ")"
		for _, v := range vendors {
			args = append(args, v)
		}
	}

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query vendors for refresh: %w", err)
	}
	defer rows.Close()

	var vendorNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan vendor name: %w", err)
		}
		vendorNames = append(vendorNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vendors: %w", err)
	}

	if len(vendorNames) == 0 {
		return &RefreshResult{
			VendorsRefreshed: 0,
			RecordsUpdated:   0,
			DurationMs:       time.Since(start).Milliseconds(),
		}, nil
	}

	// For each vendor, query pricing snapshots and create trend snapshots
	recordsUpdated := 0
	now := time.Now().UTC()

	for _, vendor := range vendorNames {
		pricingRows, err := e.db.QueryContext(ctx, `
			SELECT ps.sku, ps.list_price
			FROM pricing_snapshot ps
			JOIN vendors v ON v.id = ps.vendor_id
			WHERE v.name = ?
		`, vendor)
		if err != nil {
			e.logger.Warn("query pricing for vendor refresh", "vendor", vendor, "error", err)
			continue
		}

		for pricingRows.Next() {
			var sku string
			var listPrice float64
			if err := pricingRows.Scan(&sku, &listPrice); err != nil {
				pricingRows.Close()
				e.logger.Warn("scan pricing row", "vendor", vendor, "error", err)
				continue
			}

			// Apply ±3% random variation
			variation := 1.0 + (rand.Float64()*6.0-3.0)/100.0
			snapshotPrice := math.Round(listPrice*variation*100) / 100

			snapshot := &trends.PriceSnapshot{
				Vendor:    vendor,
				SKU:       sku,
				Price:     snapshotPrice,
				ListPrice: listPrice,
				Date:      now,
				CreatedAt: now,
			}

			if err := trendsStore.Save(ctx, snapshot); err != nil {
				e.logger.Warn("save price snapshot", "vendor", vendor, "sku", sku, "error", err)
				continue
			}
			recordsUpdated++
		}
		pricingRows.Close()
	}

	return &RefreshResult{
		VendorsRefreshed: len(vendorNames),
		RecordsUpdated:   recordsUpdated,
		DurationMs:       time.Since(start).Milliseconds(),
	}, nil
}
