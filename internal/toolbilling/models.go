package toolbilling

// ToolPrice represents the price per call for a specific tool.
type ToolPrice struct {
	ToolName     string  `json:"tool_name"`
	PricePerCall float64 `json:"price_per_call"`
}

// BillingReport represents a billing summary for a specific API key over a period.
type BillingReport struct {
	KeyID      string         `json:"key_id"`
	TotalCalls int            `json:"total_calls"`
	TotalCost  float64        `json:"total_cost"`
	PeriodFrom string         `json:"period_from"`
	PeriodTo   string         `json:"period_to"`
	PerTool    map[string]int `json:"per_tool"`
}

// UsageTier represents the current usage tier status for an API key.
type UsageTier struct {
	KeyID          string `json:"key_id"`
	CurrentTier    string `json:"current_tier"`
	CallsThisMonth int    `json:"calls_this_month"`
	TierLimit      int    `json:"tier_limit"`
}
