package pricing

import "time"

// Vendor represents a SaaS vendor in the database.
type Vendor struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// PricingDataPoint is a single market pricing observation for a vendor SKU.
type PricingDataPoint struct {
	ID          int64   `json:"id"`
	VendorID    int64   `json:"vendor_id"`
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	ListPrice   float64 `json:"list_price"`
	MinObserved float64 `json:"min_observed"`
	MaxObserved float64 `json:"max_observed"`
	TypicalPct  float64 `json:"typical_pct"`
	Unit        string  `json:"unit"`
	UpdatedAt   time.Time
}

// PriceQueryResult is the result of a price lookup.
type PriceQueryResult struct {
	Vendor       string  `json:"vendor"`
	SKU          string  `json:"sku"`
	ListPrice    float64 `json:"list_price"`
	MarketMin    float64 `json:"market_min"`
	MarketMax    float64 `json:"market_max"`
	SuggestedMin float64 `json:"suggested_min"`
	SuggestedMax float64 `json:"suggested_max"`
	DataPoints   int     `json:"data_points_count"`
	Confidence   string  `json:"confidence"`
	TypicalPct   float64 `json:"typical_discount_pct"`
	Description  string  `json:"description,omitempty"`
}

// SavingsEstimate is returned by the savings calculator.
type SavingsEstimate struct {
	Vendor             string        `json:"vendor"`
	CurrentSpend       float64       `json:"current_spend"`
	EstimatedSavings   float64       `json:"estimated_savings"`
	SavingsPercentage  float64       `json:"savings_percentage"`
	Confidence         string        `json:"confidence"`
	SimilarDeals       []SimilarDeal `json:"similar_deals"`
	MarketAveragePrice float64       `json:"market_average_price"`
}

// SimilarDeal is a comparable deal from the history.
type SimilarDeal struct {
	Vendor      string  `json:"vendor"`
	DiscountPct float64 `json:"discount_percentage"`
	Seats       int     `json:"seats"`
	TermMonths  int     `json:"term_months"`
	FinalPrice  float64 `json:"final_price"`
}

// MarketRange holds min/max/average price data.
type MarketRange struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// ModelResult is a single AI model pricing result for the negotiate_find_cheapest_model tool.
type ModelResult struct {
	Vendor       string   `json:"vendor"`
	SKU          string   `json:"sku"`
	Description  string   `json:"description"`
	PricePerUnit float64  `json:"price_per_unit"`
	Unit         string   `json:"unit"`
	TaskTypes    []string `json:"task_types"`
}
