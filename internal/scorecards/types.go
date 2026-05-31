package scorecards

// VendorScorecard is the aggregated vendor performance score.
type VendorScorecard struct {
	Vendor            string           `json:"vendor"`
	Period            string           `json:"period"`
	OverallScore      float64          `json:"overall_score"`
	PricingScore      float64          `json:"pricing_score"`
	ReliabilityScore  float64          `json:"reliability_score"`
	SupportScore      float64          `json:"support_score"`
	RelationshipScore float64          `json:"relationship_score"`
	Trend             string           `json:"trend"`
	Details           ScorecardDetails `json:"details"`
}

// ScorecardDetails contains the raw metrics behind the scores.
type ScorecardDetails struct {
	TotalDeals       int     `json:"total_deals"`
	AvgDiscount      float64 `json:"avg_discount"`
	WinRate          float64 `json:"win_rate"`
	SLACompliancePct float64 `json:"sla_compliance_pct"`
	SignalCount      int     `json:"signal_count"`
	TenureMonths     int     `json:"tenure_months"`
}
