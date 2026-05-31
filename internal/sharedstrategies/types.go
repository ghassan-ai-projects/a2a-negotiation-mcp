package sharedstrategies

// SharedStrategy represents a negotiation strategy shared by a user.
type SharedStrategy struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Notes        string `json:"notes"`
	StrategyType string `json:"strategy_type"`
	UsageCount   int    `json:"usage_count"`
	CreatedAt    string `json:"created_at"`
}
