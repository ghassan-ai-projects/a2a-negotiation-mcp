package vendorspend

// VendorSpendEntry is the breakdown for a single vendor.
type VendorSpendEntry struct {
	Vendor        string  `json:"vendor"`
	TotalSpend    float64 `json:"total_spend"`
	SpendPct      float64 `json:"spend_pct"`
	Subscriptions int     `json:"subscriptions"`
	AvgCost       float64 `json:"avg_cost"`
	YoYChangePct  float64 `json:"yoy_change_pct"`
}

// SpendTrendPoint is a single data point in a spend trend.
type SpendTrendPoint struct {
	Month string  `json:"month"`
	Spend float64 `json:"spend"`
}

// VendorSpendReport is the full report response.
type VendorSpendReport struct {
	Period        string             `json:"period"`
	TotalSpend    float64            `json:"total_spend"`
	Vendors       int                `json:"vendors"`
	Subscriptions int                `json:"subscriptions"`
	ByVendor      []VendorSpendEntry `json:"by_vendor"`
	MonthlyTrend  []SpendTrendPoint  `json:"monthly_trend"`
	YoYChangePct  float64            `json:"yoy_change_pct"`
}
