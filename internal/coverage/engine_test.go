package coverage

import (
	"testing"
)

func TestParseOutput_CoveredPackages(t *testing.T) {
	e := NewEngine()
	data := []byte(`ok  	github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/coverage	0.5s	coverage: 85.2% of statements
ok  	github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server	1.2s	coverage: 72.1% of statements
`)

	report, err := e.ParseOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.OverallPct < 78 || report.OverallPct > 80 {
		t.Errorf("expected overall ~78.65%%, got %.1f%%", report.OverallPct)
	}

	if len(report.Packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(report.Packages))
	}

	if report.Packages[0].CoveragePct != 85.2 {
		t.Errorf("expected 85.2 coverage, got %.1f", report.Packages[0].CoveragePct)
	}
}

func TestParseOutput_UntestedPackages(t *testing.T) {
	e := NewEngine()
	data := []byte(`?  	github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/newpkg	[no test files]
ok  	github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/coverage	0.5s	coverage: 85.2% of statements
`)

	report, err := e.ParseOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.UntestedPackages) != 1 {
		t.Errorf("expected 1 untested package, got %d: %v", len(report.UntestedPackages), report.UntestedPackages)
	}

	if len(report.UntestedPackages) > 0 && report.UntestedPackages[0] != "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/newpkg" {
		t.Errorf("unexpected untested package name: %s", report.UntestedPackages[0])
	}
}

func TestParseOutput_Empty(t *testing.T) {
	e := NewEngine()
	report, err := e.ParseOutput([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.OverallPct != 0 {
		t.Errorf("expected 0 overall coverage, got %.1f%%", report.OverallPct)
	}
	if len(report.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(report.Packages))
	}
}

func TestGenerateRecommendation_HighCoverage(t *testing.T) {
	e := NewEngine()
	rec := e.generateRecommendation(90.0, nil)
	if rec == "" {
		t.Error("expected non-empty recommendation")
	}
}

func TestGenerateRecommendation_Untested(t *testing.T) {
	e := NewEngine()
	rec := e.generateRecommendation(30.0, []string{"pkg/foo", "pkg/bar"})
	if rec == "" {
		t.Error("expected non-empty recommendation")
	}
}
