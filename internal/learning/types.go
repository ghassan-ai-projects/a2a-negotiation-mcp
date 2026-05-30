package learning

import "time"

// StrategyOutcome records the result of a single negotiation for learning.
type StrategyOutcome struct {
	Vendor         string    `json:"vendor"`
	SKU            string    `json:"sku"`
	Strategy       string    `json:"strategy"`
	DiscountPct    float64   `json:"discount_pct"`
	RoundsComplete int       `json:"rounds_complete"`
	Outcome        string    `json:"outcome"`
	BudgetUsed     float64   `json:"budget_used,omitempty"`
	TotalBefore    float64   `json:"total_before"`
	TotalAfter     float64   `json:"total_after"`
	Timestamp      time.Time `json:"timestamp"`
}

// StrategyRecommendation is the recommendation returned for a vendor.
type StrategyRecommendation struct {
	Vendor              string                         `json:"vendor"`
	RecommendedStrategy string                         `json:"recommended_strategy"`
	Confidence          string                         `json:"confidence"` // "high", "medium", "low"
	AvgDiscount         float64                        `json:"avg_discount_pct"`
	TotalDeals          int                            `json:"total_deals"`
	Breakdown           map[string]VendorStrategyStats `json:"breakdown"`
}

// VendorStrategyStats holds per-strategy statistics for a vendor.
type VendorStrategyStats struct {
	AvgDiscount float64 `json:"avg_discount"`
	WinRate     float64 `json:"win_rate"`
	TotalDeals  int     `json:"total_deals"`
}
