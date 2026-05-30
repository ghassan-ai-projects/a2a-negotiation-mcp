package roi

import "time"

// ROICalculation holds the result of an ROI analysis for a negotiated deal.
type ROICalculation struct {
	ID                   int64     `json:"id"`
	Vendor               string    `json:"vendor"`
	CurrentSpend         float64   `json:"current_spend"`
	NegotiatedPrice      float64   `json:"negotiated_price"`
	ImplementationCosts  float64   `json:"implementation_costs"`
	AnnualOverhead       float64   `json:"annual_overhead"`
	AnnualSavings        float64   `json:"annual_savings"`
	ROIPct               float64   `json:"roi_pct"`
	PaybackMonths        float64   `json:"payback_months"`
	Savings1Y            float64   `json:"savings_1y"`
	Savings3Y            float64   `json:"savings_3y"`
	Savings5Y            float64   `json:"savings_5y"`
	NPV                  float64   `json:"npv"`
	UserID               string    `json:"user_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}
