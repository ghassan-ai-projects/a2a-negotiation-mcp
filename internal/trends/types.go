package trends

import "time"

// PriceSnapshot is a recorded price point for a vendor SKU at a point in time.
type PriceSnapshot struct {
	ID        int64     `json:"id"`
	Vendor    string    `json:"vendor"`
	SKU       string    `json:"sku"`
	Price     float64   `json:"price"`
	ListPrice float64   `json:"list_price,omitempty"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

// TrendAnalysis is the full analysis result for a vendor/SKU.
type TrendAnalysis struct {
	Vendor        string       `json:"vendor"`
	SKU           string       `json:"sku"`
	Period        string       `json:"period"`
	Direction     string       `json:"direction"` // "up", "down", "stable"
	Slope         float64      `json:"slope"`
	Volatility    float64      `json:"volatility"` // stddev / mean
	PriceChange6M float64      `json:"price_change_6m_pct"`
	Forecast3M    float64      `json:"forecast_3m"`
	Forecast6M    float64      `json:"forecast_6m"`
	Seasonal      bool         `json:"seasonal"` // true if Q4 > Q1 by >10%
	DataPoints    int          `json:"data_points"`
	Snapshots     []PricePoint `json:"snapshots,omitempty"`
}

// PricePoint is a simplified snapshot for API responses.
type PricePoint struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}
