package useractivity

// UserActivityReport provides per-user activity and performance data.
type UserActivityReport struct {
	UserID                string          `json:"user_id"`
	Period                string          `json:"period"`
	TotalSessions         int             `json:"total_sessions"`
	CompletedNegotiations int             `json:"completed_negotiations"`
	TotalSavings          float64         `json:"total_savings"`
	ActiveDays            int             `json:"active_days"`
	LastActive            string          `json:"last_active"`
	FavoriteStrategies    []StrategyUsage `json:"favorite_strategies"`
	TopVendors            []VendorUsage   `json:"top_vendors"`
}

// StrategyUsage shows how often a strategy was used.
type StrategyUsage struct {
	Strategy string `json:"strategy"`
	Count    int    `json:"count"`
}

// VendorUsage shows how often a vendor was negotiated with.
type VendorUsage struct {
	Vendor string `json:"vendor"`
	Count  int    `json:"count"`
}
