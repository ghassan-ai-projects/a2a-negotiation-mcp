package trends

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Engine performs price trend analysis using linear regression.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a trends engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// Analyze performs trend analysis for a vendor/SKU over a period.
// Uses linear regression for direction and forecasting.
func (e *Engine) Analyze(ctx context.Context, vendor, sku, period string) (*TrendAnalysis, error) {
	// Parse period
	now := time.Now().UTC()
	var startDate time.Time
	switch period {
	case "30d":
		startDate = now.AddDate(0, 0, -30)
	case "90d":
		startDate = now.AddDate(0, 0, -90)
	case "6m", "6mo":
		startDate = now.AddDate(0, -6, 0)
	case "1y", "":
		if period == "" {
			period = "1y"
		}
		startDate = now.AddDate(0, -12, 0)
	case "2y":
		startDate = now.AddDate(0, -24, 0)
	default:
		return nil, fmt.Errorf("invalid period %q: use 30d, 90d, 6m, 1y, or 2y", period)
	}

	// Query snapshots
	snapshots, err := e.store.Query(ctx, vendor, sku, startDate, now, 0)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}

	if len(snapshots) < 2 {
		// Not enough data for trend analysis
		ta := &TrendAnalysis{
			Vendor:     vendor,
			SKU:        sku,
			Period:     period,
			Direction:  "insufficient_data",
			DataPoints: len(snapshots),
		}
		if len(snapshots) == 1 {
			ta.Snapshots = []PricePoint{{
				Date:  snapshots[0].Date.Format(time.DateOnly),
				Price: snapshots[0].Price,
			}}
		}
		return ta, nil
	}

	// Convert to x,y for linear regression
	// x = days since first snapshot, y = price
	firstDate := snapshots[0].Date
	n := len(snapshots)

	x := make([]float64, n)
	y := make([]float64, n)
	points := make([]PricePoint, n)
	for i, s := range snapshots {
		days := s.Date.Sub(firstDate).Hours() / 24
		x[i] = days
		y[i] = s.Price
		points[i] = PricePoint{
			Date:  s.Date.Format(time.DateOnly),
			Price: s.Price,
		}
	}

	// Linear regression: y = mx + b
	slope, intercept := linearRegression(x, y)

	// Direction
	direction := "stable"
	avgPrice := mean(y)
	if avgPrice > 0 {
		// Normalize slope relative to avg price per month
		slopePerMonth := slope * 30
		changePct := slopePerMonth / avgPrice * 100
		if changePct > 1.0 {
			direction = "up"
		} else if changePct < -1.0 {
			direction = "down"
		}
	}

	// Volatility = stddev / mean
	stddev := stdDev(y)
	volatility := 0.0
	if avgPrice > 0 {
		volatility = stddev / avgPrice
	}

	// Price change over last 6 months (or available period)
	priceChange6M := 0.0
	if len(snapshots) >= 2 {
		firstPrice := snapshots[0].Price
		lastPrice := snapshots[len(snapshots)-1].Price
		if firstPrice > 0 {
			priceChange6M = (lastPrice - firstPrice) / firstPrice * 100
		}
	}

	// Forecast: extend line 3 and 6 months out
	lastDay := x[len(x)-1]
	forecast3M := intercept + slope*(lastDay+90)
	forecast6M := intercept + slope*(lastDay+180)

	// Seasonal check: compare Q4 vs Q1
	seasonal := checkSeasonal(snapshots)

	// Get stats
	_, _, _, _, count, _ := e.store.GetStats(ctx, vendor, sku)

	ta := &TrendAnalysis{
		Vendor:        vendor,
		SKU:           sku,
		Period:        period,
		Direction:     direction,
		Slope:         math.Round(slope*10000) / 10000,
		Volatility:    math.Round(volatility*10000) / 10000,
		PriceChange6M: math.Round(priceChange6M*100) / 100,
		Forecast3M:    math.Round(forecast3M*100) / 100,
		Forecast6M:    math.Round(forecast6M*100) / 100,
		Seasonal:      seasonal,
		DataPoints:    count,
		Snapshots:     points,
	}

	return ta, nil
}

// linearRegression computes slope and intercept for the best-fit line.
func linearRegression(x, y []float64) (slope, intercept float64) {
	n := float64(len(x))
	if n <= 1 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n
	return
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stdDev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	m := mean(vals)
	var sumSq float64
	for _, v := range vals {
		diff := v - m
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func minVal(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxVal(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// checkSeasonal detects if Q4 prices are on average >10% higher than Q1.
func checkSeasonal(snapshots []PriceSnapshot) bool {
	var q1Prices, q4Prices []float64
	for _, s := range snapshots {
		month := s.Date.Month()
		if month >= time.January && month <= time.March {
			q1Prices = append(q1Prices, s.Price)
		}
		if month >= time.October && month <= time.December {
			q4Prices = append(q4Prices, s.Price)
		}
	}
	if len(q1Prices) == 0 || len(q4Prices) == 0 {
		return false
	}

	q1Avg := mean(q1Prices)
	q4Avg := mean(q4Prices)
	if q1Avg <= 0 {
		return false
	}

	return (q4Avg-q1Avg)/q1Avg > 0.10
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
