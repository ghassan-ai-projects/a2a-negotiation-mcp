package limitedoffer

import "time"

// OfferInput represents a time-limited offer from a vendor.
type OfferInput struct {
	Vendor       string    `json:"vendor"`
	SKU          string    `json:"sku"`
	OfferPrice   float64   `json:"offer_price"`
	ExpiresAt    time.Time `json:"expires_at"`
	CurrentPrice float64   `json:"current_price,omitempty"`
	CurrentSpend float64   `json:"current_spend,omitempty"`
}

// OfferResult is the analysis result for a time-limited offer.
type OfferResult struct {
	Savings        float64 `json:"savings"`
	DaysRemaining  float64 `json:"days_remaining"`
	Urgency        string  `json:"urgency"`
	Recommendation string  `json:"recommendation"`
	VsBestPricePct float64 `json:"vs_best_price_pct"`
}
