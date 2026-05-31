package slacredit

// SLACreditInput holds parameters for SLA credit calculation.
type SLACreditInput struct {
	Vendor           string  `json:"vendor"`
	Service          string  `json:"service"`
	MonthlySpend     float64 `json:"monthly_spend"`
	UptimePct        float64 `json:"uptime_pct"`
	GuaranteedUptime float64 `json:"guaranteed_uptime"`
	CreditRate       float64 `json:"credit_rate"`
}

// SLACreditOutput is the result of an SLA credit calculation.
type SLACreditOutput struct {
	Vendor           string  `json:"vendor"`
	Service          string  `json:"service"`
	MonthlySpend     float64 `json:"monthly_spend"`
	ActualUptime     float64 `json:"actual_uptime"`
	GuaranteedUptime float64 `json:"guaranteed_uptime"`
	CreditRate       float64 `json:"credit_rate"`
	CreditAmount     float64 `json:"credit_amount"`
	Eligible         bool    `json:"eligible"`
}
