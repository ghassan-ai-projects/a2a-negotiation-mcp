package parallel

// ParallelConfig defines how to select the best result from parallel negotiations.
type ParallelConfig struct {
	SessionIDs []string `json:"session_ids"` // cannot be empty
	Strategy   string   `json:"strategy"`    // "best_price", "best_discount", "fastest"
	Timeout    int      `json:"timeout"`     // seconds per session (optional, default 30)
}

// ParallelResult is the output of parallel negotiation.
type ParallelResult struct {
	WinnerSessionID string           `json:"winner_session_id"`
	WinnerVendor    string           `json:"winner_vendor"`
	WinnerOffer     float64          `json:"winner_offer"`
	WinnerDiscount  float64          `json:"winner_discount_pct"`
	Strategy        string           `json:"strategy"`
	TotalRounds     int              `json:"total_rounds"`
	AllResults      []SessionSummary `json:"all_results"`
	DurationMs      int64            `json:"duration_ms"`
}

// SessionSummary holds the outcome of a single session within parallel negotiation.
type SessionSummary struct {
	SessionID   string  `json:"session_id"`
	Vendor      string  `json:"vendor"`
	SKU         string  `json:"sku"`
	Strategy    string  `json:"strategy"`
	Offer       float64 `json:"offer"`
	DiscountPct float64 `json:"discount_pct"`
	Outcome     string  `json:"outcome"`
	Rounds      int     `json:"rounds"`
	DurationMs  int64   `json:"duration_ms"`
}
