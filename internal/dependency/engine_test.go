package dependency

import (
	"testing"
)

func TestParse_DirectDependencies(t *testing.T) {
	e := NewEngine()
	mod := `module github.com/test/example

go 1.26.3

require (
	github.com/google/uuid v1.6.0
	github.com/mark3labs/mcp-go v0.54.1
	modernc.org/sqlite v1.51.0
)
`
	report, err := e.Parse([]byte(mod))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 3 {
		t.Errorf("expected 3 total dependencies, got %d", report.TotalCount)
	}

	if len(report.Direct) != 3 {
		t.Errorf("expected 3 direct dependencies, got %d", len(report.Direct))
	}

	if len(report.Indirect) != 0 {
		t.Errorf("expected 0 indirect dependencies, got %d", len(report.Indirect))
	}
}

func TestParse_IndirectDependencies(t *testing.T) {
	e := NewEngine()
	mod := `module github.com/test/example

go 1.26.3

require (
	github.com/google/uuid v1.6.0
	github.com/mark3labs/mcp-go v0.54.1 // indirect
	modernc.org/sqlite v1.51.0 // indirect
)
`
	report, err := e.Parse([]byte(mod))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Direct) != 1 {
		t.Errorf("expected 1 direct dependency, got %d", len(report.Direct))
	}

	if len(report.Indirect) != 2 {
		t.Errorf("expected 2 indirect dependencies, got %d", len(report.Indirect))
	}

	if report.TotalCount != 3 {
		t.Errorf("expected 3 total dependencies, got %d", report.TotalCount)
	}
}

func TestParse_SingleLineRequire(t *testing.T) {
	e := NewEngine()
	mod := `module github.com/test/example

go 1.26.3

require github.com/google/uuid v1.6.0
`
	report, err := e.Parse([]byte(mod))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Direct) != 1 {
		t.Errorf("expected 1 direct dependency, got %d", len(report.Direct))
	}
}

func TestParse_EmptyMod(t *testing.T) {
	e := NewEngine()
	report, err := e.Parse([]byte("module github.com/test/example\n\ngo 1.26.3\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 0 {
		t.Errorf("expected 0 total dependencies, got %d", report.TotalCount)
	}
}
