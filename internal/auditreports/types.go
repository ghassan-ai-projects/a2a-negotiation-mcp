package auditreports

type AuditRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Format string `json:"format"`
}

type AuditReport struct {
	PeriodFrom string      `json:"period_from"`
	PeriodTo   string      `json:"period_to"`
	Format     string      `json:"format"`
	Data       string      `json:"data"`
	RowCount   int         `json:"row_count"`
	Summary    AuditSummary `json:"summary"`
}

type AuditSummary struct {
	TotalNegotiations int     `json:"total_negotiations"`
	TotalSavings      float64 `json:"total_savings"`
	AvgDiscount       float64 `json:"avg_discount"`
	PeriodDescription string  `json:"period_description"`
}

type AuditTrailEntry struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Action     string `json:"action"`
	Timestamp  string `json:"timestamp"`
}
