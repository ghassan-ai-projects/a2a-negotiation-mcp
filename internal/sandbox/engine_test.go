package sandbox

import (
	"context"
	"testing"
)

func TestGetTemplates(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	templates, err := e.GetTemplates(ctx)
	if err != nil {
		t.Fatalf("GetTemplates: %v", err)
	}

	if len(templates) != 5 {
		t.Errorf("expected 5 templates, got %d", len(templates))
	}

	expectedNames := []string{"pricing", "negotiation", "contract", "vendor_comparison", "savings"}
	for _, expected := range expectedNames {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == expected {
				found = true
				if tmpl.ToolName == "" {
					t.Errorf("template %s has empty ToolName", expected)
				}
				if tmpl.Description == "" {
					t.Errorf("template %s has empty Description", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected template %s not found", expected)
		}
	}
}

func TestExecuteReturnsResult(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	result, err := e.Execute(ctx, "negotiate_query_price", `{"vendor":"Slack"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result == "" {
		t.Fatal("expected non-empty result")
	}
}
