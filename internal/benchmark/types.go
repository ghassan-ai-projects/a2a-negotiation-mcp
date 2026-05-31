package benchmark

// BenchmarkReport summarizes a user's savings performance vs. all users.
type BenchmarkReport struct {
	UserID         string          `json:"user_id"`
	Period         string          `json:"period"`
	TotalSavings   float64         `json:"total_savings"`
	AvgDiscountPct float64         `json:"avg_discount_pct"`
	DealCount      int             `json:"deal_count"`
	Percentile     float64         `json:"percentile"` // 0-100, higher = better
	ByVendor       []VendorSavings `json:"by_vendor"`
}

// VendorSavings breaks down savings by vendor.
type VendorSavings struct {
	Vendor         string  `json:"vendor"`
	Savings        float64 `json:"savings"`
	AvgDiscountPct float64 `json:"avg_discount_pct"`
	DealCount      int     `json:"deal_count"`
}
