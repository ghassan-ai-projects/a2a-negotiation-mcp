package strategycomparison

// StrategyComparisonRequest is the input for a strategy comparison.
type StrategyComparisonRequest struct {
	Vendor     string   `json:"vendor"`
	SKU        string   `json:"sku"`
	Strategies []string `json:"strategies"`
	Budget     float64  `json:"budget"`
}

// StrategyComparisonResult is the output of a strategy comparison.
type StrategyComparisonResult struct {
	Vendor       string           `json:"vendor"`
	SKU          string           `json:"sku"`
	Budget       float64          `json:"budget"`
	Results      []StrategyResult `json:"results"`
	BestStrategy string           `json:"best_strategy"`
}

// StrategyResult is the result of a single strategy simulation.
type StrategyResult struct {
	Strategy        string  `json:"strategy"`
	LikelyOutcome   string  `json:"likely_outcome"`
	ExpectedSavings float64 `json:"expected_savings"`
	RiskLevel       string  `json:"risk_level"`
}
