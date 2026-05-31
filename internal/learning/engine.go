package learning

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	_ "modernc.org/sqlite"
)

// Engine analyzes past negotiation outcomes to recommend optimal strategies.
type Engine struct {
	db        *sql.DB
	histStore *history.Store
	logger    *slog.Logger
}

// NewEngine creates a learning engine backed by the history store's DB.
func NewEngine(histStore *history.Store, logger *slog.Logger) (*Engine, error) {
	e := &Engine{
		db:        histStore.DB(),
		histStore: histStore,
		logger:    logger,
	}
	if err := e.migrate(); err != nil {
		return nil, fmt.Errorf("learning migrate: %w", err)
	}
	return e, nil
}

func (e *Engine) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS learning_outcomes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		sku TEXT,
		strategy TEXT NOT NULL,
		discount_pct REAL,
		rounds_complete INTEGER,
		outcome TEXT,
		budget_used REAL,
		total_before REAL,
		total_after REAL,
		timestamp TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_learning_vendor ON learning_outcomes(vendor);

	CREATE TABLE IF NOT EXISTS failure_autopsies (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		vendor TEXT,
		sku TEXT,
		strategy TEXT,
		failure_reason TEXT,
		final_offer REAL,
		vendor_best REAL,
		gap REAL,
		tactic_used TEXT,
		created_at TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_failure_vendor ON failure_autopsies(vendor);
	`
	_, err := e.db.Exec(schema)
	return err
}

// RecordOutcome saves a negotiation outcome for future learning.
func (e *Engine) RecordOutcome(ctx context.Context, outcome StrategyOutcome) error {
	ts := outcome.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO learning_outcomes (vendor, sku, strategy, discount_pct, rounds_complete, outcome, budget_used, total_before, total_after, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, outcome.Vendor, outcome.SKU, outcome.Strategy, outcome.DiscountPct,
		outcome.RoundsComplete, outcome.Outcome, outcome.BudgetUsed,
		outcome.TotalBefore, outcome.TotalAfter, ts.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("record outcome: %w", err)
	}
	e.logger.Debug("recorded learning outcome",
		"vendor", outcome.Vendor, "strategy", outcome.Strategy, "outcome", outcome.Outcome)
	return nil
}

// GetRecommendation analyzes past deals for a vendor and returns the best strategy.
//
//	High confidence: >=10 deals with the winning strategy
//	Medium: 3-9 deals
//	Low: <3 deals or zero deals (returns "balanced" as default)
func (e *Engine) GetRecommendation(ctx context.Context, vendor string) (*StrategyRecommendation, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT strategy, COUNT(*) as cnt,
		       AVG(discount_pct) as avg_disc,
		       COALESCE(SUM(CASE WHEN outcome = 'accepted' THEN 1 ELSE 0 END), 0) as wins
		FROM learning_outcomes
		WHERE vendor = ?
		GROUP BY strategy
		ORDER BY avg_disc DESC
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("query learning outcomes: %w", err)
	}
	defer rows.Close()

	rec := &StrategyRecommendation{
		Vendor:    vendor,
		Breakdown: make(map[string]VendorStrategyStats),
	}

	type strategyRow struct {
		name  string
		count int
		avg   float64
		wins  int
	}

	var rowsData []strategyRow
	for rows.Next() {
		var r strategyRow
		if err := rows.Scan(&r.name, &r.count, &r.avg, &r.wins); err != nil {
			return nil, fmt.Errorf("scan strategy row: %w", err)
		}
		rowsData = append(rowsData, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(rowsData) == 0 {
		return &StrategyRecommendation{
			Vendor:              vendor,
			RecommendedStrategy: "balanced",
			Confidence:          "low",
			AvgDiscount:         0,
			TotalDeals:          0,
			Breakdown:           map[string]VendorStrategyStats{},
		}, nil
	}

	totalDeals := 0
	for _, r := range rowsData {
		totalDeals += r.count
		winRate := 0.0
		if r.count > 0 {
			winRate = math.Round(float64(r.wins)/float64(r.count)*10000) / 100
		}
		avgDisc := math.Round(r.avg*100) / 100
		rec.Breakdown[r.name] = VendorStrategyStats{
			AvgDiscount: avgDisc,
			WinRate:     winRate,
			TotalDeals:  r.count,
		}
	}
	rec.TotalDeals = totalDeals

	// Pick best strategy (highest avg discount)
	bestStrategy := rowsData[0].name
	bestAvg := rowsData[0].avg
	bestCount := rowsData[0].count
	for _, r := range rowsData {
		if r.avg > bestAvg {
			bestStrategy = r.name
			bestAvg = r.avg
			bestCount = r.count
		}
	}
	rec.RecommendedStrategy = bestStrategy
	rec.AvgDiscount = math.Round(bestAvg*100) / 100

	// Determine confidence based on best strategy deal count
	switch {
	case bestCount >= 10:
		rec.Confidence = "high"
	case bestCount >= 3:
		rec.Confidence = "medium"
	default:
		rec.Confidence = "low"
	}

	return rec, nil
}

// GetGlobalInsights returns overall strategy performance across all vendors.
func (e *Engine) GetGlobalInsights(ctx context.Context) (map[string]interface{}, error) {
	var totalOutcomes int
	var totalAccepted int
	err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN outcome = 'accepted' THEN 1 ELSE 0 END), 0)
		FROM learning_outcomes
	`).Scan(&totalOutcomes, &totalAccepted)
	if err != nil {
		return nil, fmt.Errorf("query global insights: %w", err)
	}

	// Per-strategy aggregate stats
	rows, err := e.db.QueryContext(ctx, `
		SELECT strategy,
		       COUNT(*) as cnt,
		       AVG(discount_pct) as avg_disc,
		       COALESCE(SUM(CASE WHEN outcome = 'accepted' THEN 1 ELSE 0 END), 0) as wins
		FROM learning_outcomes
		GROUP BY strategy
		ORDER BY avg_disc DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query strategy aggregates: %w", err)
	}
	defer rows.Close()

	type strategyStats struct {
		Strategy    string  `json:"strategy"`
		TotalDeals  int     `json:"total_deals"`
		AvgDiscount float64 `json:"avg_discount_pct"`
		WinRate     float64 `json:"win_rate"`
	}

	var perStrategy []strategyStats
	for rows.Next() {
		var s strategyStats
		var wins int
		if err := rows.Scan(&s.Strategy, &s.TotalDeals, &s.AvgDiscount, &wins); err != nil {
			return nil, fmt.Errorf("scan strategy stats: %w", err)
		}
		s.AvgDiscount = math.Round(s.AvgDiscount*100) / 100
		if s.TotalDeals > 0 {
			s.WinRate = math.Round(float64(wins)/float64(s.TotalDeals)*10000) / 100
		}
		perStrategy = append(perStrategy, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Top vendors by deal count
	vendorRows, err := e.db.QueryContext(ctx, `
		SELECT vendor, COUNT(*) as cnt,
		       AVG(discount_pct) as avg_disc,
		       COALESCE(SUM(CASE WHEN outcome = 'accepted' THEN 1 ELSE 0 END), 0) as wins
		FROM learning_outcomes
		GROUP BY vendor
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("query vendor insights: %w", err)
	}
	defer vendorRows.Close()

	type vendorStats struct {
		Vendor      string  `json:"vendor"`
		TotalDeals  int     `json:"total_deals"`
		AvgDiscount float64 `json:"avg_discount_pct"`
		WinRate     float64 `json:"win_rate"`
	}
	var perVendor []vendorStats
	for vendorRows.Next() {
		var v vendorStats
		var wins int
		if err := vendorRows.Scan(&v.Vendor, &v.TotalDeals, &v.AvgDiscount, &wins); err != nil {
			return nil, fmt.Errorf("scan vendor stats: %w", err)
		}
		v.AvgDiscount = math.Round(v.AvgDiscount*100) / 100
		if v.TotalDeals > 0 {
			v.WinRate = math.Round(float64(wins)/float64(v.TotalDeals)*10000) / 100
		}
		perVendor = append(perVendor, v)
	}

	overallWinRate := 0.0
	if totalOutcomes > 0 {
		overallWinRate = math.Round(float64(totalAccepted)/float64(totalOutcomes)*10000) / 100
	}

	return map[string]interface{}{
		"total_outcomes":   totalOutcomes,
		"total_accepted":   totalAccepted,
		"overall_win_rate": overallWinRate,
		"strategies":       perStrategy,
		"top_vendors":      perVendor,
	}, nil
}
