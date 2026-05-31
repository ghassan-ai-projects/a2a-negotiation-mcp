package pricechart

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/trends"
)

// Engine produces chart data by combining price snapshots and deal outcomes.
type Engine struct {
	trendsStore  *trends.Store
	historyStore *history.Store
}

// NewEngine creates a price chart engine.
func NewEngine(trendsStore *trends.Store, historyStore *history.Store) *Engine {
	return &Engine{trendsStore: trendsStore, historyStore: historyStore}
}

// Chart builds ChartData for a vendor, optionally filtered by SKU and period.
func (e *Engine) Chart(ctx context.Context, vendor, sku, period string) (*ChartData, error) {
	if period == "" {
		period = "1y"
	}

	now := time.Now().UTC()
	var startDate time.Time
	switch period {
	case "30d":
		startDate = now.AddDate(0, 0, -30)
	case "90d":
		startDate = now.AddDate(0, 0, -90)
	case "1y":
		startDate = now.AddDate(0, -12, 0)
	case "2y":
		startDate = now.AddDate(0, -24, 0)
	default:
		return nil, fmt.Errorf("invalid period %q", period)
	}

	// Query price snapshots for list_price data
	snapshots, err := e.trendsStore.Query(ctx, vendor, sku, startDate, now, 0)
	if err != nil {
		return nil, fmt.Errorf("query trends: %w", err)
	}

	db := e.historyStore.DB()

	// Query deal outcomes for negotiated prices, grouped by month
	dealQuery := `
		SELECT strftime('%Y-%m', created_at) AS month,
		       COALESCE(AVG(final_price), 0) AS avg_price,
		       COUNT(*) AS deal_count
		FROM deal_outcomes
		WHERE vendor = ? AND created_at >= ? AND created_at <= ?`
	dealArgs := []any{vendor, startDate.Format(time.RFC3339), now.Format(time.RFC3339)}

	if sku != "" {
		dealQuery = `
			SELECT strftime('%Y-%m', created_at) AS month,
			       COALESCE(AVG(final_price), 0) AS avg_price,
			       COUNT(*) AS deal_count
			FROM deal_outcomes
			WHERE vendor = ? AND sku = ? AND created_at >= ? AND created_at <= ?`
		dealArgs = []any{vendor, sku, startDate.Format(time.RFC3339), now.Format(time.RFC3339)}
	}

	dealQuery += " GROUP BY month ORDER BY month ASC"

	rows, err := db.QueryContext(ctx, dealQuery, dealArgs...)
	if err != nil {
		return nil, fmt.Errorf("query deal outcomes: %w", err)
	}
	defer rows.Close()

	type dealRow struct {
		month     string
		avgPrice  float64
		dealCount int
	}

	var dealRows []dealRow
	for rows.Next() {
		var dr dealRow
		if err := rows.Scan(&dr.month, &dr.avgPrice, &dr.dealCount); err != nil {
			return nil, err
		}
		dealRows = append(dealRows, dr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build month-keyed maps
	monthLabels := buildMonthLabels(startDate, now)

	snapshotMap := make(map[string]float64)
	for _, s := range snapshots {
		key := s.Date.Format("2006-01")
		// Use list_price as the dataset value
		snapshotMap[key] = s.ListPrice
	}

	dealMap := make(map[string]float64)
	for _, dr := range dealRows {
		dealMap[dr.month] = dr.avgPrice
	}

	var listData, negotiatedData []float64
	for _, label := range monthLabels {
		lp, hasList := snapshotMap[label]
		if hasList {
			listData = append(listData, lp)
		} else {
			listData = append(listData, 0)
		}

		dp, hasDeal := dealMap[label]
		if hasDeal {
			negotiatedData = append(negotiatedData, dp)
		} else {
			negotiatedData = append(negotiatedData, 0)
		}
	}

	datasets := []ChartDataset{
		{Label: "List Price", Data: listData, Color: "#ef4444"},
		{Label: "Negotiated", Data: negotiatedData, Color: "#22c55e"},
	}

	// Build summary
	summary := e.buildSummary(ctx, vendor, sku)

	return &ChartData{
		Labels:   monthLabels,
		Datasets: datasets,
		Summary:  summary,
	}, nil
}

func (e *Engine) buildSummary(ctx context.Context, vendor, sku string) ChartSummary {
	db := e.historyStore.DB()

	query := `
		SELECT COALESCE(AVG(final_price), 0),
		       COALESCE(SUM(list_price - final_price), 0),
		       COALESCE(MIN(final_price), 0),
		       COALESCE(MAX(final_price), 0)
		FROM deal_outcomes WHERE vendor = ?`
	args := []any{vendor}

	if sku != "" {
		query += " AND sku = ?"
		args = append(args, sku)
	}

	var avgPrice, totalSavings, bestDeal, worstDeal float64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&avgPrice, &totalSavings, &bestDeal, &worstDeal); err != nil {
		return ChartSummary{}
	}

	return ChartSummary{
		AvgPrice:     math.Round(avgPrice*100) / 100,
		TotalSavings: math.Round(totalSavings*100) / 100,
		BestDeal:     bestDeal,
		WorstDeal:    worstDeal,
	}
}

func buildMonthLabels(start, end time.Time) []string {
	var labels []string
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !current.After(endMonth) {
		labels = append(labels, current.Format("2006-01"))
		current = current.AddDate(0, 1, 0)
	}
	return labels
}
