package calendar

import "time"

// Contract represents a SaaS contract with renewal tracking.
type Contract struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	Vendor              string    `json:"vendor"`
	SKU                 string    `json:"sku"`
	Seats               int       `json:"seats"`
	CurrentPrice        float64   `json:"current_price_per_unit"`
	RenewalDate         time.Time `json:"renewal_date"`
	Status              string    `json:"status"` // "active", "negotiating", "renewed", "cancelled"
	LastNegotiatedPrice float64   `json:"last_negotiated_price,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// RenewalCheck holds the renewal assessment for a single contract.
type RenewalCheck struct {
	Contract         Contract `json:"contract"`
	DaysUntil        int      `json:"days_until_renewal"`
	ActionNeeded     string   `json:"action_needed"` // "urgent", "soon", "monitor", "none"
	Urgency          string   `json:"urgency"`       // "high" (<30d), "medium" (30-90d), "low" (>90d)
	SuggestedSavings float64  `json:"suggested_savings,omitempty"`
}
