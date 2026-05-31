package scorecards

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
)

// Engine computes vendor scorecards from multiple data stores.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a scorecard engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Scorecard generates a vendor scorecard for the given period.
func (e *Engine) Scorecard(ctx context.Context, vendor, period string) (*VendorScorecard, error) {
	if period == "" {
		period = "1y"
	}

	since := parsePeriod(period)

	// Query deal_outcomes
	details := e.loadDealStats(ctx, vendor, since)

	// Query vendor_health for signals count
	details.SignalCount = e.loadSignalCount(ctx, vendor)

	// Query sla_breaches for compliance
	details.SLACompliancePct = e.loadSLACompliance(ctx, vendor, since)

	// Compute tenure from earliest deal
	details.TenureMonths = e.loadTenure(ctx, vendor)

	// Compute dimension scores
	pricingScore := math.Min(details.AvgDiscount/50.0*100, 100)
	if math.IsNaN(pricingScore) || math.IsInf(pricingScore, 0) {
		pricingScore = 0
	}

	reliabilityScore := details.SLACompliancePct

	supportScore := math.Min(50+float64(details.SignalCount)*5, 100)

	relationshipScore := math.Min(float64(details.TenureMonths)/24.0*100, 100)

	overall := pricingScore*0.3 + reliabilityScore*0.3 + supportScore*0.2 + relationshipScore*0.2

	return &VendorScorecard{
		Vendor:            vendor,
		Period:            period,
		OverallScore:      math.Round(overall*100) / 100,
		PricingScore:      math.Round(pricingScore*100) / 100,
		ReliabilityScore:  math.Round(reliabilityScore*100) / 100,
		SupportScore:      math.Round(supportScore*100) / 100,
		RelationshipScore: math.Round(relationshipScore*100) / 100,
		Trend:             "stable",
		Details:           details,
	}, nil
}

func (e *Engine) loadDealStats(ctx context.Context, vendor string, since time.Time) ScorecardDetails {
	var details ScorecardDetails

	row := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(AVG(discount_pct), 0), COALESCE(AVG(CASE WHEN status = 'won' THEN 1.0 ELSE 0 END), 0)
		 FROM deal_outcomes WHERE vendor = ? AND created_at >= ?`,
		vendor, since.Format(time.RFC3339),
	)
	if err := row.Scan(&details.TotalDeals, &details.AvgDiscount, &details.WinRate); err != nil {
		e.logger.Warn("scorecard: failed to query deal_outcomes", "vendor", vendor, "error", err)
	}

	return details
}

func (e *Engine) loadSignalCount(ctx context.Context, vendor string) int {
	var count sql.NullInt64
	err := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM health_signals WHERE vendor = ?`, vendor,
	).Scan(&count)
	if err != nil || !count.Valid {
		return 0
	}
	return int(count.Int64)
}

func (e *Engine) loadSLACompliance(ctx context.Context, vendor string, since time.Time) float64 {
	var total, breaches sql.NullFloat64
	err := e.db.QueryRowContext(ctx,
		`SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(CASE WHEN filed = 1 THEN 1 ELSE 0 END), 0)
		 FROM sla_breaches WHERE vendor = ? AND date >= ?`,
		vendor, since.Format(time.RFC3339),
	).Scan(&total, &breaches)
	if err != nil || !total.Valid || total.Float64 == 0 {
		return 100
	}
	pct := (1 - breaches.Float64/total.Float64) * 100
	if pct < 0 {
		pct = 0
	}
	return math.Round(pct*100) / 100
}

func (e *Engine) loadTenure(ctx context.Context, vendor string) int {
	var earliest sql.NullString
	err := e.db.QueryRowContext(ctx,
		`SELECT MIN(created_at) FROM deal_outcomes WHERE vendor = ?`, vendor,
	).Scan(&earliest)
	if err != nil || !earliest.Valid || earliest.String == "" {
		return 0
	}

	t, err := time.Parse(time.RFC3339, earliest.String)
	if err != nil {
		e.logger.Warn("scorecard: failed to parse tenure date", "vendor", vendor, "date", earliest.String)
		return 0
	}

	months := int(time.Since(t).Hours() / (24 * 30))
	if months < 0 {
		months = 0
	}
	return months
}

func parsePeriod(period string) time.Time {
	now := time.Now().UTC()
	period = strings.TrimSpace(strings.ToLower(period))
	switch {
	case strings.HasSuffix(period, "y"):
		n := 1
		fmt.Sscanf(period, "%dy", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(-n, 0, 0)
	case strings.HasSuffix(period, "m"):
		n := 1
		fmt.Sscanf(period, "%dm", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(0, -n, 0)
	case strings.HasSuffix(period, "d"):
		n := 90
		fmt.Sscanf(period, "%dd", &n)
		if n <= 0 {
			n = 90
		}
		return now.AddDate(0, 0, -n)
	default:
		return now.AddDate(-1, 0, 0)
	}
}
