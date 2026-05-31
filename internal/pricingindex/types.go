package pricingindex

// PricingIndex represents the competitive pricing snapshot for a category.
type PricingIndex struct {
	Category      string        `json:"category"`
	Period        string        `json:"period"`
	AvgPrice      float64       `json:"avg_price"`
	PriceRange    PriceRange    `json:"price_range"`
	Vendors       []VendorIndex `json:"vendors"`
	MoMChangePct  float64       `json:"mom_change_pct"`
	VolatilityIdx float64       `json:"volatility_index"`
}

// PriceRange defines the min and max observed prices.
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// VendorIndex captures a single vendor's pricing within the index.
type VendorIndex struct {
	Vendor     string  `json:"vendor"`
	AvgPrice   float64 `json:"avg_price"`
	Category   string  `json:"category"`
	PriceTrend string  `json:"price_trend"` // up/down/stable
}
