package contribguide

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_ContainsExpectedSections(t *testing.T) {
	// Use the actual project root (where go.mod exists)
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	e := NewEngine()
	guide, err := e.Generate(projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if guide.Content == "" {
		t.Error("expected non-empty content")
	}

	if len(guide.Sections) == 0 {
		t.Error("expected at least one section")
	}

	// Verify essential sections
	expectedSections := []string{"Development Setup", "Building", "Testing", "Package Conventions", "MCP Tool Registration", "PR Workflow"}
	for _, s := range expectedSections {
		found := false
		for _, section := range guide.Sections {
			if section == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected section %q not found in guide", s)
		}
	}

	// Verify content contains key terms
	for _, term := range []string{"Go 1.26", "SQLite", "go build", "go test"} {
		if !contains(guide.Content, term) {
			t.Errorf("expected content to contain %q", term)
		}
	}
}

func TestDiscoverSections_WithGoMod(t *testing.T) {
	e := NewEngine()
	// Create a temp dir that looks like a Go project
	dir := t.TempDir()
	tmpGoMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(tmpGoMod, []byte("module test"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	tmpInternal := filepath.Join(dir, "internal")
	if err := os.Mkdir(tmpInternal, 0755); err != nil {
		t.Fatalf("failed to create internal dir: %v", err)
	}

	sections := e.discoverSections(dir)
	if len(sections) == 0 {
		t.Error("expected sections to be discovered")
	}
}

func TestBuildMarkdown_NoError(t *testing.T) {
	e := NewEngine()
	content := e.buildMarkdown("/tmp", []string{"Development Setup", "Testing"})
	if content == "" {
		t.Error("expected non-empty markdown")
	}
}

// findProjectRoot walks up from the test file to find the project root.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
