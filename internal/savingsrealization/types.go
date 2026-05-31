package savingsrealization

// SavingsRealization records actual savings achieved for a deal.
type SavingsRealization struct {
	ID              int64   `json:"id"`
	SessionID       string  `json:"session_id"`
	Vendor          string  `json:"vendor"`
	ProjectedAmount float64 `json:"projected_amount"`
	ActualAmount    float64 `json:"actual_amount"`
	Period          string  `json:"period"`
	CreatedAt       string  `json:"created_at"`
}

// VendorRealization is a per-vendor breakdown in the report.
type VendorRealization struct {
	Vendor    string  `json:"vendor"`
	Projected float64 `json:"projected"`
	Actual    float64 `json:"actual"`
	Rate      float64 `json:"rate"`
}

// RealizationReport aggregates savings realization across all vendors.
type RealizationReport struct {
	TotalProjected  float64             `json:"total_projected"`
	TotalRealized   float64             `json:"total_realized"`
	RealizationRate float64             `json:"realization_rate"`
	ByVendor        []VendorRealization `json:"by_vendor"`
	TopShortfalls   []VendorRealization `json:"top_shortfalls"`
}

// RecordResult is the response for recording a realization.
type RecordResult struct {
	ID        int64   `json:"id"`
	Vendor    string  `json:"vendor"`
	Projected float64 `json:"projected"`
	Actual    float64 `json:"actual"`
	Status    string  `json:"status"`
}
