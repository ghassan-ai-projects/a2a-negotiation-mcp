package communitybench

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

// Store manages community benchmark data in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a CommunityBenchmarkStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate communitybench: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS community_benchmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT '',
		discount_pct REAL NOT NULL DEFAULT 0,
		deal_value REAL NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// UploadBenchmark inserts a new benchmark entry and returns it with the generated ID and timestamp.
func (s *Store) UploadBenchmark(ctx context.Context, vendor, category string, discountPct, dealValue float64) (*BenchmarkEntry, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO community_benchmarks (vendor, category, discount_pct, deal_value, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, vendor, category, discountPct, dealValue, now)
	if err != nil {
		return nil, fmt.Errorf("upload benchmark: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("upload benchmark last insert id: %w", err)
	}
	return &BenchmarkEntry{
		ID:          int(id),
		Vendor:      vendor,
		Category:    category,
		DiscountPct: discountPct,
		DealValue:   dealValue,
		CreatedAt:   now,
	}, nil
}

// GetBenchmarks returns aggregated benchmark stats grouped by category, optionally filtered by category.
func (s *Store) GetBenchmarks(ctx context.Context, category string) ([]CommunityBenchmark, error) {
	query := `SELECT category, AVG(discount_pct), COUNT(*), AVG(deal_value) FROM community_benchmarks`
	args := []any{}
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` GROUP BY category ORDER BY category`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get benchmarks: %w", err)
	}
	defer rows.Close()

	type row struct {
		category    string
		avgDisc     float64
		count       int
		avgDeal     float64
	}
	var raw []row
	// Collect rows first, then we need deal values for median calculation
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.category, &r.avgDisc, &r.count, &r.avgDeal); err != nil {
			return nil, fmt.Errorf("scan benchmark: %w", err)
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute median deal value per category by re-querying raw values
	var benchmarks []CommunityBenchmark
	for _, r := range raw {
		median, err := s.medianDealValue(ctx, r.category)
		if err != nil {
			return nil, fmt.Errorf("median deal for %s: %w", r.category, err)
		}
		benchmarks = append(benchmarks, CommunityBenchmark{
			Category:    r.category,
			AvgDiscount: math.Round(r.avgDisc*100) / 100,
			MedianDeal:  math.Round(median*100) / 100,
			SampleCount: r.count,
		})
	}
	if benchmarks == nil {
		benchmarks = []CommunityBenchmark{}
	}
	return benchmarks, nil
}

// medianDealValue computes the median deal_value for a given category.
func (s *Store) medianDealValue(ctx context.Context, category string) (float64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT deal_value FROM community_benchmarks WHERE category = ? ORDER BY deal_value`, category)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var values []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(values) == 0 {
		return 0, nil
	}
	sort.Float64s(values)
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2, nil
	}
	return values[n/2], nil
}

// CompareToBenchmark compares an input deal against benchmarks for a category and returns percentile-like metrics.
func (s *Store) CompareToBenchmark(ctx context.Context, myDiscount, myValue float64, category string) (map[string]any, error) {
	benchmarks, err := s.GetBenchmarks(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("compare to benchmark: %w", err)
	}

	result := map[string]any{
		"category":          category,
		"my_discount_pct":   myDiscount,
		"my_deal_value":     myValue,
	}

	if len(benchmarks) == 0 {
		result["avg_discount"] = nil
		result["median_deal"] = nil
		result["sample_count"] = 0
		result["discount_vs_avg"] = 0
		result["value_vs_median"] = 0
		result["message"] = "No benchmark data available for this category."
		return result, nil
	}

	// Use first (and only) benchmark since we filtered by category
	b := benchmarks[0]

	result["avg_discount"] = b.AvgDiscount
	result["median_deal"] = b.MedianDeal
	result["sample_count"] = b.SampleCount

	// How my discount compares to average (positive = better)
	if b.AvgDiscount > 0 {
		result["discount_vs_avg"] = math.Round((myDiscount-b.AvgDiscount)/b.AvgDiscount*100*100) / 100
	} else {
		result["discount_vs_avg"] = 0
	}

	// How my deal value compares to median
	if b.MedianDeal > 0 {
		result["value_vs_median"] = math.Round((myValue-b.MedianDeal)/b.MedianDeal*100*100) / 100
	} else {
		result["value_vs_median"] = 0
	}

	return result, nil
}
