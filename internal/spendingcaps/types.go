package spendingcaps

// SpendingCap defines soft and hard spending caps for a vendor.
type SpendingCap struct {
	Vendor    string  `json:"vendor"`
	SoftCap   float64 `json:"soft_cap"`
	HardCap   float64 `json:"hard_cap"`
	Period    string  `json:"period"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// CapCheckResult shows whether a vendor's spend has triggered caps.
type CapCheckResult struct {
	Vendor       string  `json:"vendor"`
	SoftCap      float64 `json:"soft_cap"`
	HardCap      float64 `json:"hard_cap"`
	CurrentSpend float64 `json:"current_spend"`
	SoftReached  bool    `json:"soft_reached"`
	HardReached  bool    `json:"hard_reached"`
	Period       string  `json:"period"`
}

// SetCapResult is the response for setting a spending cap.
type SetCapResult struct {
	Vendor  string  `json:"vendor"`
	SoftCap float64 `json:"soft_cap"`
	HardCap float64 `json:"hard_cap"`
	Period  string  `json:"period"`
	Status  string  `json:"status"`
}

// DeleteCapResult is the response for deleting a spending cap.
type DeleteCapResult struct {
	Vendor string `json:"vendor"`
	Status string `json:"status"`
}
