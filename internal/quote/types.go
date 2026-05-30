package quote

// QuoteInput represents the input for analyzing a vendor quote email.
type QuoteInput struct {
	RawText string `json:"raw_text"`
	Vendor  string `json:"vendor,omitempty"`
	SKU     string `json:"sku,omitempty"`
}

// Quote represents an extracted quote from vendor email text.
type Quote struct {
	Vendor       string  `json:"vendor"`
	SKU          string  `json:"sku,omitempty"`
	Description  string  `json:"description,omitempty"`
	Seats        int     `json:"seats"`
	TermMonths   int     `json:"term_months"`
	PricePerUnit float64 `json:"price_per_unit"`
	TotalPrice   float64 `json:"total_price"`
	ListPrice    float64 `json:"list_price"`
	// DiscountOffered is the % off list price (0 if list price is 0 or unknown).
	DiscountOffered float64 `json:"discount_offered"`
}

// QuoteAnalysis is the full analysis result including market cross-referencing.
type QuoteAnalysis struct {
	Quote            Quote     `json:"quote"`
	MarketRange      []float64 `json:"market_range"`
	CounterOfferMin  float64   `json:"counter_offer_min"`
	CounterOfferMax  float64   `json:"counter_offer_max"`
	PotentialSavings float64   `json:"potential_savings"`
	Confidence       string    `json:"confidence"`
}
