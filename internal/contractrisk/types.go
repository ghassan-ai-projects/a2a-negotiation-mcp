package contractrisk

// RiskReport is the result of a contract risk analysis.
type RiskReport struct {
	OverallScore    int          `json:"overall_score"`
	RiskLevel       string       `json:"risk_level"`
	Clauses         []RiskClause `json:"clauses"`
	Recommendations []string     `json:"recommendations"`
}

// RiskClause represents a single risky clause found in the contract.
type RiskClause struct {
	Clause   string `json:"clause"`
	Score    int    `json:"score"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}
