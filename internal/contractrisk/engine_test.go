package contractrisk

import (
	"testing"
)

func TestHighRiskContract(t *testing.T) {
	eng := NewEngine()
	text := "This agreement will automatically renew and includes price escalation with annual increase."

	report := eng.Analyze(text)

	if report.OverallScore <= 0 {
		t.Errorf("expected non-zero overall score for high-risk contract, got %d", report.OverallScore)
	}
	if report.RiskLevel != "high" {
		t.Errorf("expected risk level 'high', got %q", report.RiskLevel)
	}
	if len(report.Clauses) == 0 {
		t.Fatal("expected at least 1 risky clause")
	}

	// Check auto-renewal was detected
	foundAutoRenew := false
	for _, c := range report.Clauses {
		if c.Detail == "Auto-renewal clause" {
			foundAutoRenew = true
			if c.Score != 70 {
				t.Errorf("expected auto-renew score 70, got %d", c.Score)
			}
			if c.Severity != "high" {
				t.Errorf("expected severity 'high', got %q", c.Severity)
			}
			break
		}
	}
	if !foundAutoRenew {
		t.Error("expected auto-renewal clause to be detected")
	}
}

func TestMediumRiskContract(t *testing.T) {
	eng := NewEngine()
	text := "The vendor shall indemnify the customer against all claims. This is an exclusive agreement."

	report := eng.Analyze(text)

	if report.OverallScore <= 0 {
		t.Errorf("expected non-zero overall score, got %d", report.OverallScore)
	}
	if report.RiskLevel != "medium" {
		t.Errorf("expected risk level 'medium', got %q", report.RiskLevel)
	}
	if len(report.Clauses) < 2 {
		t.Errorf("expected at least 2 clauses, got %d", len(report.Clauses))
	}

	if len(report.Recommendations) == 0 {
		t.Error("expected at least 1 recommendation")
	}
}

func TestNoRiskClauses(t *testing.T) {
	eng := NewEngine()
	text := "This is a simple contract agreement between two parties for the provision of services."

	report := eng.Analyze(text)

	if report.OverallScore != 0 {
		t.Errorf("expected overall score 0, got %d", report.OverallScore)
	}
	if report.RiskLevel != "low" {
		t.Errorf("expected risk level 'low', got %q", report.RiskLevel)
	}
	if len(report.Clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(report.Clauses))
	}
	if len(report.Recommendations) != 0 {
		t.Errorf("expected 0 recommendations, got %d", len(report.Recommendations))
	}
}

func TestEmptyText(t *testing.T) {
	eng := NewEngine()
	report := eng.Analyze("")

	if report.OverallScore != 0 {
		t.Errorf("expected overall score 0, got %d", report.OverallScore)
	}
	if report.RiskLevel != "low" {
		t.Errorf("expected risk level 'low', got %q", report.RiskLevel)
	}
	if len(report.Clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(report.Clauses))
	}
}
