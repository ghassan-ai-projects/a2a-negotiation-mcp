package chartexport

import (
	"context"
	"strings"
	"testing"
)

func TestExportChart_SVG(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.ExportChart(ctx, "test-source", "bar", "svg")
	if err != nil {
		t.Fatalf("ExportChart: %v", err)
	}

	if result.Format != "svg" {
		t.Errorf("expected format svg, got %s", result.Format)
	}
	if result.ChartType != "bar" {
		t.Errorf("expected chart_type bar, got %s", result.ChartType)
	}
	if result.MimeType != "image/svg+xml" {
		t.Errorf("expected mime image/svg+xml, got %s", result.MimeType)
	}
	if result.Width != 800 {
		t.Errorf("expected width 800, got %d", result.Width)
	}
	if result.Height != 400 {
		t.Errorf("expected height 400, got %d", result.Height)
	}
	if !strings.HasPrefix(result.Data, "<svg") {
		t.Errorf("expected SVG data to start with <svg, got %q", result.Data[:20])
	}
	if !strings.Contains(result.Data, "Bar Chart") {
		t.Errorf("expected SVG to contain chart type name, got %q", result.Data)
	}
}

func TestExportChart_PNG(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.ExportChart(ctx, "test-source", "line", "png")
	if err != nil {
		t.Fatalf("ExportChart: %v", err)
	}

	if result.Format != "png" {
		t.Errorf("expected format png, got %s", result.Format)
	}
	if result.ChartType != "line" {
		t.Errorf("expected chart_type line, got %s", result.ChartType)
	}
	if result.MimeType != "image/png" {
		t.Errorf("expected mime image/png, got %s", result.MimeType)
	}
	if result.Width != 800 {
		t.Errorf("expected width 800, got %d", result.Width)
	}
	if result.Height != 400 {
		t.Errorf("expected height 400, got %d", result.Height)
	}
	expected := "[PNG data simulation for line]"
	if result.Data != expected {
		t.Errorf("expected %q, got %q", expected, result.Data)
	}
}

func TestExportChart_InvalidChartType(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.ExportChart(ctx, "source", "invalid", "svg")
	if err == nil {
		t.Fatal("expected error for invalid chart type")
	}
	if !strings.Contains(err.Error(), "invalid chart_type") {
		t.Errorf("expected error about invalid chart_type, got %v", err)
	}
}

func TestExportChart_InvalidFormat(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.ExportChart(ctx, "source", "bar", "pdf")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("expected error about invalid format, got %v", err)
	}
}

func TestListTemplates(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	templates, err := eng.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	if len(templates) != 5 {
		t.Errorf("expected 5 templates, got %d", len(templates))
	}

	chartTypes := make(map[string]bool)
	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("expected non-empty template name")
		}
		if tmpl.Description == "" {
			t.Error("expected non-empty template description")
		}
		if tmpl.ChartType == "" {
			t.Error("expected non-empty chart type")
		}
		if tmpl.ColorScheme == "" {
			t.Error("expected non-empty color scheme")
		}
		chartTypes[tmpl.ChartType] = true
	}

	expectedTypes := []string{"bar", "line", "pie", "area", "scatter"}
	for _, ct := range expectedTypes {
		if !chartTypes[ct] {
			t.Errorf("missing chart type %s in templates", ct)
		}
	}
}
