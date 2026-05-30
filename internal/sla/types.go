package sla

import "time"

// SLAContract represents a registered SLA contract with a vendor.
type SLAContract struct {
	ID           string  `json:"id"`
	Vendor       string  `json:"vendor"`
	Service      string  `json:"service"`
	UptimePct    float64 `json:"uptime_pct"`
	CreditPct    float64 `json:"credit_pct"`
	MaxCreditPct float64 `json:"max_credit_pct"`
	MonthlySpend float64 `json:"monthly_spend"`
	Status       string  `json:"status"` // "active", "paused", "archived"
}

// SLABreach represents a single SLA breach incident.
type SLABreach struct {
	ID           string    `json:"id"`
	Vendor       string    `json:"vendor"`
	Service      string    `json:"service"`
	Date         time.Time `json:"date"`
	DurationMins int       `json:"duration_mins"`
	CreditDue    float64   `json:"credit_due"`
	Filed        bool      `json:"filed"`
	FiledAt      time.Time `json:"filed_at,omitempty"`
	Payout       float64   `json:"payout,omitempty"`
	Notes        string    `json:"notes,omitempty"`
}

// SLAResult is the aggregate report for a given month.
type SLAResult struct {
	Contract     SLAContract `json:"contract"`
	Breaches     []SLABreach `json:"breaches"`
	TotalCredits float64     `json:"total_credits"`
	FiledCount   int         `json:"filed_count"`
}
