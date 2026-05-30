package miner

// NegotiationOpportunity represents a discovered opportunity for negotiation.
type NegotiationOpportunity struct {
	ID              string  `json:"id"`
	Category        string  `json:"category"`         // "software", "hosting", "saas", "carrier", "freelance"
	Vendor          string  `json:"vendor"`           // specific vendor (empty for generic opportunities)
	EstimatedSpend  float64 `json:"estimated_spend"`  // annual estimated spend
	Confidence      string  `json:"confidence"`       // "high", "medium", "low"
	Rationale       string  `json:"rationale"`        // why this is worth negotiating
	TypicalDiscount float64 `json:"typical_discount"` // % typical savings in this category
}

// BusinessProfile describes a business for opportunity discovery.
type BusinessProfile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Employees   int      `json:"employees,omitempty"`
	Industry    string   `json:"industry,omitempty"`
	Vendors     []string `json:"vendors,omitempty"` // known vendor names to cross-reference
}
