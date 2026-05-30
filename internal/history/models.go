package history

import "time"

// SessionRecord is the full negotiation session record stored in SQLite.
type SessionRecord struct {
	ID             string    `json:"session_id"`
	Vendor         string    `json:"vendor"`
	SKU            string    `json:"sku,omitempty"`
	Strategy       string    `json:"strategy"`
	Budget         float64   `json:"budget,omitempty"`
	Status         string    `json:"status"`
	CurrentOffer   float64   `json:"current_offer"`
	ListPrice      float64   `json:"list_price"`
	RoundsComplete int       `json:"rounds_completed"`
	Outcome        string    `json:"outcome,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RoundRecord is a single negotiation round stored in SQLite.
type RoundRecord struct {
	ID           int       `json:"id"`
	SessionID    string    `json:"session_id"`
	RoundNumber  int       `json:"round_number"`
	Offer        float64   `json:"offer"`
	DiscountPct  float64   `json:"discount_percentage"`
	Counterparty string    `json:"counterparty"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// DealOutcome is a completed deal stored in SQLite.
type DealOutcome struct {
	ID          int64     `json:"id"`
	Vendor      string    `json:"vendor"`
	SKU         string    `json:"sku,omitempty"`
	ListPrice   float64   `json:"list_price"`
	FinalPrice  float64   `json:"final_price"`
	DiscountPct float64   `json:"discount_percentage"`
	Seats       int       `json:"seats,omitempty"`
	TermMonths  int       `json:"term_months"`
	Strategy    string    `json:"strategy"`
	SessionID   string    `json:"session_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistorySummary is the aggregated history response.
type HistorySummary struct {
	TotalDeals     int           `json:"total_deals"`
	WinRate        float64       `json:"win_rate"`
	AvgDiscountPct float64       `json:"avg_discount_percentage"`
	TotalSavings   float64       `json:"total_savings"`
	Deals          []DealOutcome `json:"deals"`
}
