package communitybench

// BenchmarkEntry represents a single uploaded benchmark data point.
type BenchmarkEntry struct {
	ID          int     `json:"id"`
	Vendor      string  `json:"vendor"`
	Category    string  `json:"category"`
	DiscountPct float64 `json:"discount_pct"`
	DealValue   float64 `json:"deal_value"`
	CreatedAt   string  `json:"created_at"`
}

// CommunityBenchmark represents aggregated benchmark stats for a category.
type CommunityBenchmark struct {
	Category    string  `json:"category"`
	AvgDiscount float64 `json:"avg_discount"`
	MedianDeal  float64 `json:"median_deal"`
	SampleCount int     `json:"sample_count"`
}
