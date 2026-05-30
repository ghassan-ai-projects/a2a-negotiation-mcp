package winloss

// WinLossReport is the top-level analytics result for won/lost deal analysis.
type WinLossReport struct {
	Period       string              `json:"period"`
	TotalDeals   int                 `json:"total_deals"`
	Won          int                 `json:"won"`
	Lost         int                 `json:"lost"`
	Pending      int                 `json:"pending"`
	WinRate      float64             `json:"win_rate_pct"`
	ByStrategy   []StrategyBreakdown `json:"by_strategy"`
	ByVendor     []VendorBreakdown   `json:"by_vendor"`
	MonthlyTrend []MonthTrend        `json:"monthly_trend"`
}

// StrategyBreakdown breaks down wins/losses by negotiation strategy.
type StrategyBreakdown struct {
	Strategy string  `json:"strategy"`
	Won      int     `json:"won"`
	Lost     int     `json:"lost"`
	WinRate  float64 `json:"win_rate_pct"`
}

// VendorBreakdown breaks down wins/losses by vendor.
type VendorBreakdown struct {
	Vendor  string  `json:"vendor"`
	Won     int     `json:"won"`
	Lost    int     `json:"lost"`
	WinRate float64 `json:"win_rate_pct"`
}

// MonthTrend is the monthly won/lost trend data point.
type MonthTrend struct {
	Month   string  `json:"month"`
	Won     int     `json:"won"`
	Lost    int     `json:"lost"`
	WinRate float64 `json:"win_rate_pct"`
}
