package contractrisk

import (
	"strings"
)

// Engine analyzes contract text for risky clauses.
type Engine struct{}

// NewEngine creates a contract risk engine.
func NewEngine() *Engine {
	return &Engine{}
}

type clausePattern struct {
	pattern  string
	score    int
	severity string
	detail   string
}

var riskPatterns = []clausePattern{
	{pattern: "auto-renew", score: 70, severity: "high", detail: "Auto-renewal clause"},
	{pattern: "automatically renew", score: 70, severity: "high", detail: "Auto-renewal clause"},
	{pattern: "price escalation", score: 60, severity: "high", detail: "Price escalation"},
	{pattern: "annual increase", score: 60, severity: "high", detail: "Price escalation"},
	{pattern: "non-refundable", score: 50, severity: "medium", detail: "Non-refundable terms"},
	{pattern: "no refund", score: 50, severity: "medium", detail: "Non-refundable terms"},
	{pattern: "indemnify", score: 40, severity: "medium", detail: "Indemnification"},
	{pattern: "hold harmless", score: 40, severity: "medium", detail: "Indemnification"},
	{pattern: "exclusive", score: 45, severity: "medium", detail: "Exclusivity clause"},
	{pattern: "exclusivity", score: 45, severity: "medium", detail: "Exclusivity clause"},
	{pattern: "termination for convenience", score: 30, severity: "medium", detail: "Termination for convenience"},
	{pattern: "terminate at any time", score: 30, severity: "medium", detail: "Termination for convenience"},
}

// Analyze checks contract text for risky clauses and returns a RiskReport.
func (e *Engine) Analyze(contractText string) RiskReport {
	text := strings.ToLower(contractText)
	var clauses []RiskClause

	for _, rp := range riskPatterns {
		if strings.Contains(text, rp.pattern) {
			clauses = append(clauses, RiskClause{
				Clause:   rp.detail,
				Score:    rp.score,
				Severity: rp.severity,
				Detail:   rp.detail,
			})
		}
	}

	// Deduplicate: if multiple patterns match the same detail, keep the highest score
	clauses = deduplicate(clauses)

	overallScore := 0
	if len(clauses) > 0 {
		sum := 0
		for _, c := range clauses {
			sum += c.Score
		}
		overallScore = sum / len(clauses)
	}

	riskLevel := "low"
	switch {
	case overallScore > 60:
		riskLevel = "high"
	case overallScore >= 30:
		riskLevel = "medium"
	}

	recommendations := buildRecommendations(clauses)

	return RiskReport{
		OverallScore:    overallScore,
		RiskLevel:       riskLevel,
		Clauses:         clauses,
		Recommendations: recommendations,
	}
}

func deduplicate(clauses []RiskClause) []RiskClause {
	seen := map[string]int{}
	var result []RiskClause
	for _, c := range clauses {
		idx, exists := seen[c.Detail]
		if exists {
			if c.Score > result[idx].Score {
				result[idx].Score = c.Score
			}
		} else {
			seen[c.Detail] = len(result)
			result = append(result, c)
		}
	}
	return result
}

func buildRecommendations(clauses []RiskClause) []string {
	var recs []string
	for _, c := range clauses {
		switch c.Detail {
		case "Auto-renewal clause":
			recs = append(recs, "Review auto-renewal terms — consider adding notice period or opt-out clause.")
		case "Price escalation":
			recs = append(recs, "Negotiate a cap on annual price increases to manage long-term costs.")
		case "Non-refundable terms":
			recs = append(recs, "Request partial refund terms for early termination or service failures.")
		case "Indemnification":
			recs = append(recs, "Limit indemnification obligations to mutual terms and cap liability.")
		case "Exclusivity clause":
			recs = append(recs, "Evaluate exclusivity scope — consider sunset clause or narrower definition.")
		case "Termination for convenience":
			recs = append(recs, "Ensure reciprocal termination rights and adequate notice period.")
		}
	}
	return recs
}
