package compliance

import (
	"context"
	"fmt"
	"strings"
)

// rule defines a single compliance check.
type rule struct {
	ID          string
	Keyword     string
	Description string
	Critical    bool // critical rules get "high" severity
}

// Engine checks negotiation terms against regulatory requirements.
type Engine struct {
	rules map[string][]rule
}

// NewEngine creates an Engine pre-populated with rules for GDPR, SOC2, HIPAA, and CCPA.
func NewEngine() *Engine {
	return &Engine{
		rules: map[string][]rule{
			"gdpr": {
				{ID: "gdpr-1", Keyword: "data retention", Description: "Data retention policy is defined", Critical: false},
				{ID: "gdpr-2", Keyword: "right to erasure", Description: "Right to erasure / deletion is addressed", Critical: true},
				{ID: "gdpr-3", Keyword: "data processing", Description: "Data processing terms are specified", Critical: false},
				{ID: "gdpr-4", Keyword: "breach notification", Description: "Breach notification procedure is defined", Critical: true},
				{ID: "gdpr-5", Keyword: "consent", Description: "Consent mechanism for data processing is included", Critical: false},
			},
			"soc2": {
				{ID: "soc2-1", Keyword: "access control", Description: "Access control measures are specified", Critical: true},
				{ID: "soc2-2", Keyword: "encryption", Description: "Encryption standards are defined", Critical: true},
				{ID: "soc2-3", Keyword: "monitoring", Description: "Monitoring and logging practices are included", Critical: false},
				{ID: "soc2-4", Keyword: "incident response", Description: "Incident response plan is defined", Critical: false},
				{ID: "soc2-5", Keyword: "vendor management", Description: "Vendor management process is addressed", Critical: false},
			},
			"hipaa": {
				{ID: "hipaa-1", Keyword: "phi", Description: "Protected health information (PHI) is addressed", Critical: true},
				{ID: "hipaa-2", Keyword: "baa", Description: "Business associate agreement (BAA) is included", Critical: true},
				{ID: "hipaa-3", Keyword: "privacy officer", Description: "Privacy officer designation is included", Critical: false},
				{ID: "hipaa-4", Keyword: "minimum necessary", Description: "Minimum necessary standard is addressed", Critical: false},
				{ID: "hipaa-5", Keyword: "audit control", Description: "Audit control mechanisms are specified", Critical: false},
			},
			"ccpa": {
				{ID: "ccpa-1", Keyword: "personal information", Description: "Personal information definition is included", Critical: true},
				{ID: "ccpa-2", Keyword: "opt out", Description: "Opt-out rights are addressed", Critical: false},
				{ID: "ccpa-3", Keyword: "deletion rights", Description: "Deletion rights are specified", Critical: true},
				{ID: "ccpa-4", Keyword: "category of sources", Description: "Categories of sources of personal information are disclosed", Critical: false},
				{ID: "ccpa-5", Keyword: "business purpose", Description: "Business purpose for data collection is defined", Critical: false},
			},
		},
	}
}

// Check evaluates the given terms against regulatory compliance rules for the specified jurisdiction.
func (e *Engine) Check(ctx context.Context, terms, jurisdiction string) (*ComplianceResult, error) {
	if strings.TrimSpace(terms) == "" {
		return nil, fmt.Errorf("terms must not be empty")
	}

	j := strings.ToLower(strings.TrimSpace(jurisdiction))
	rules, ok := e.rules[j]
	if !ok {
		return nil, fmt.Errorf("unsupported jurisdiction: %q (valid: gdpr, soc2, hipaa, ccpa)", jurisdiction)
	}

	termsLower := strings.ToLower(terms)
	var flags []ComplianceFlag
	passCount := 0

	for _, r := range rules {
		if strings.Contains(termsLower, r.Keyword) {
			passCount++
			continue
		}
		severity := "medium"
		if r.Critical {
			severity = "high"
		}
		flags = append(flags, ComplianceFlag{
			RuleID:         r.ID,
			Description:    r.Description,
			Severity:       severity,
			Recommendation: fmt.Sprintf("Add '%s' clause to the agreement terms", r.Keyword),
		})
	}

	var overallStatus string
	switch {
	case len(flags) == 0:
		overallStatus = "compliant"
	case hasHighSeverity(flags):
		overallStatus = "non-compliant"
	default:
		overallStatus = "needs review"
	}

	return &ComplianceResult{
		Terms:         terms,
		Jurisdiction:  jurisdiction,
		OverallStatus: overallStatus,
		Flags:         flags,
		PassCount:     passCount,
		FlagCount:     len(flags),
	}, nil
}

func hasHighSeverity(flags []ComplianceFlag) bool {
	for _, f := range flags {
		if f.Severity == "high" {
			return true
		}
	}
	return false
}
