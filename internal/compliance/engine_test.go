package compliance

import (
	"context"
	"testing"
)

func TestEngine_GDPR_Compliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `This agreement includes data retention policies, right to erasure provisions,
		data processing terms, breach notification procedures, and consent mechanisms.`

	result, err := eng.Check(ctx, terms, "gdpr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "compliant" {
		t.Errorf("expected 'compliant', got %q", result.OverallStatus)
	}
	if result.PassCount != 5 {
		t.Errorf("expected 5 passes, got %d", result.PassCount)
	}
	if result.FlagCount != 0 {
		t.Errorf("expected 0 flags, got %d", result.FlagCount)
	}
	if result.Jurisdiction != "gdpr" {
		t.Errorf("expected jurisdiction 'gdpr', got %q", result.Jurisdiction)
	}
}

func TestEngine_GDPR_NonCompliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `The parties agree to standard data processing terms.`

	result, err := eng.Check(ctx, terms, "gdpr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "non-compliant" {
		t.Errorf("expected 'non-compliant', got %q", result.OverallStatus)
	}
	if result.PassCount != 1 {
		t.Errorf("expected 1 pass (data processing), got %d", result.PassCount)
	}
	if result.FlagCount != 4 {
		t.Errorf("expected 4 flags, got %d", result.FlagCount)
	}

	// Should have at least one high severity flag (right to erasure or breach notification missing)
	hasHigh := false
	for _, f := range result.Flags {
		if f.Severity == "high" {
			hasHigh = true
			break
		}
	}
	if !hasHigh {
		t.Error("expected at least one high severity flag for missing critical GDPR rules")
	}
}

func TestEngine_InvalidJurisdiction(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.Check(ctx, "some terms", "pci-dss")
	if err == nil {
		t.Fatal("expected error for invalid jurisdiction, got nil")
	}
}

func TestEngine_EmptyTerms(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.Check(ctx, "", "gdpr")
	if err == nil {
		t.Fatal("expected error for empty terms, got nil")
	}
}

func TestEngine_SOC2_Compliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `The vendor shall implement access control measures, encryption at rest and in transit,
		continuous monitoring, incident response procedures, and vendor management processes.`

	result, err := eng.Check(ctx, terms, "soc2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "compliant" {
		t.Errorf("expected 'compliant', got %q", result.OverallStatus)
	}
	if result.PassCount != 5 {
		t.Errorf("expected 5 passes, got %d", result.PassCount)
	}
}

func TestEngine_SOC2_NonCompliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `Standard terms and conditions apply.`

	result, err := eng.Check(ctx, terms, "soc2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "non-compliant" {
		t.Errorf("expected 'non-compliant', got %q", result.OverallStatus)
	}
	if result.FlagCount != 5 {
		t.Errorf("expected 5 flags, got %d", result.FlagCount)
	}
}

func TestEngine_CCPA_Compliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `Personal information is defined per CCPA. Consumers may opt out of data sales.
		Deletion rights are provided. Each category of sources of personal information is disclosed.
		Business purpose for data collection is defined.`

	result, err := eng.Check(ctx, terms, "ccpa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "compliant" {
		t.Errorf("expected 'compliant', got %q", result.OverallStatus)
	}
	if result.PassCount != 5 {
		t.Errorf("expected 5 passes, got %d", result.PassCount)
	}
}

func TestEngine_CCPA_NeedsReview(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	// Only missing the medium-severity rules, but hits all critical ones
	terms := `Personal information is defined. Deletion rights are provided.`

	result, err := eng.Check(ctx, terms, "ccpa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "needs review" {
		t.Errorf("expected 'needs review', got %q", result.OverallStatus)
	}
	if result.PassCount != 2 {
		t.Errorf("expected 2 passes, got %d", result.PassCount)
	}
	if result.FlagCount != 3 {
		t.Errorf("expected 3 flags, got %d", result.FlagCount)
	}

	// No high severity flags since both critical ones passed
	for _, f := range result.Flags {
		if f.Severity == "high" {
			t.Errorf("unexpected high severity flag: %s", f.RuleID)
		}
	}
}

func TestEngine_HIPAA_Compliant(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := `PHI handling procedures are defined. A BAA is attached.
		A privacy officer is designated. Minimum necessary standard applies.
		Audit controls are in place.`

	result, err := eng.Check(ctx, terms, "hipaa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OverallStatus != "compliant" {
		t.Errorf("expected 'compliant', got %q", result.OverallStatus)
	}
	if result.PassCount != 5 {
		t.Errorf("expected 5 passes, got %d", result.PassCount)
	}
}
