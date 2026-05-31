package tco

// TCOInput represents the input parameters for a TCO calculation.
type TCOInput struct {
	Vendor              string  `json:"vendor"`
	SKU                 string  `json:"sku"`
	Seats               int     `json:"seats"`
	TermMonths          int     `json:"term_months"`
	ImplementationCosts float64 `json:"implementation_costs"`
	TrainingCosts       float64 `json:"training_costs"`
	SupportCosts        float64 `json:"support_costs"`
}

// TCOOutput represents the complete TCO calculation result.
type TCOOutput struct {
	Vendor                 string   `json:"vendor"`
	SKU                    string   `json:"sku"`
	Seats                  int      `json:"seats"`
	TermMonths             int      `json:"term_months"`
	PerUnitCost            float64  `json:"per_unit_cost"`
	AnnualSubscription     float64  `json:"annual_subscription"`
	Total1YTCO             float64  `json:"total_1y_tco"`
	Total3YTCO             float64  `json:"total_3y_tco"`
	CostPerUserPerMonth    float64  `json:"cost_per_user_per_month"`
	MarketAvgCUPM          float64  `json:"market_avg_cupm"`
	SavingsVsMarketPct     float64  `json:"savings_vs_market_pct"`
	HiddenCostsFlagged     []string `json:"hidden_costs_flagged"`
}
