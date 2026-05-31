package pricealerts

// PriceAlertRule defines a price drop monitoring rule for a vendor/SKU.
type PriceAlertRule struct {
	Vendor       string  `json:"vendor"`
	SKU          string  `json:"sku,omitempty"`
	ThresholdPct float64 `json:"threshold_pct"`
	Channel      string  `json:"channel"`
	Enabled      bool    `json:"enabled"`
	CreatedAt    string  `json:"created_at"`
	LastChecked  string  `json:"last_checked_at,omitempty"`
}

// PriceAlertResult is the outcome of checking a single rule.
type PriceAlertResult struct {
	Vendor        string  `json:"vendor"`
	SKU           string  `json:"sku,omitempty"`
	PreviousPrice float64 `json:"previous_price"`
	CurrentPrice  float64 `json:"current_price"`
	DropPct       float64 `json:"drop_pct"`
	ThresholdMet  bool    `json:"threshold_met"`
	AlertSent     bool    `json:"alert_sent"`
}
