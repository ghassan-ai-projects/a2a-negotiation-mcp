package chartexport

import (
	"context"
	"fmt"
	"strings"
)

// Engine is a stateless engine for simulating chart export (PNG/SVG).
type Engine struct{}

// NewEngine creates a new chart export engine.
func NewEngine() *Engine {
	return &Engine{}
}

var validChartTypes = map[string]bool{
	"bar":     true,
	"line":    true,
	"pie":     true,
	"area":    true,
	"scatter": true,
}

var validFormats = map[string]bool{
	"png": true,
	"svg": true,
}

// ExportChart generates simulated chart output for the given parameters.
func (e *Engine) ExportChart(_ context.Context, dataSource, chartType, format string) (*ChartResult, error) {
	if strings.TrimSpace(dataSource) == "" {
		return nil, fmt.Errorf("data_source must not be empty")
	}
	if !validChartTypes[chartType] {
		return nil, fmt.Errorf("invalid chart_type: %q (must be one of: bar, line, pie, area, scatter)", chartType)
	}
	if !validFormats[format] {
		return nil, fmt.Errorf("invalid format: %q (must be one of: png, svg)", format)
	}

	width := 800
	height := 400

	var data string
	var mimeType string

	switch format {
	case "svg":
		mimeType = "image/svg+xml"
		title := strings.ToUpper(chartType[:1]) + chartType[1:]
		data = fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
  <rect width="%d" height="%d" fill="#f8f9fa"/>
  <text x="%d" y="%d" text-anchor="middle" font-family="sans-serif" font-size="20" fill="#333">%s Chart</text>
  <text x="%d" y="%d" text-anchor="middle" font-family="sans-serif" font-size="14" fill="#666">Data source: %s</text>
</svg>`, width, height, width, height, width, height, width/2, 40, title, width/2, height-20, dataSource)
	case "png":
		mimeType = "image/png"
		data = fmt.Sprintf("[PNG data simulation for %s]", chartType)
	}

	return &ChartResult{
		Format:    format,
		ChartType: chartType,
		Data:      data,
		Width:     width,
		Height:    height,
		MimeType:  mimeType,
	}, nil
}

// ListTemplates returns predefined chart templates.
func (e *Engine) ListTemplates(_ context.Context) ([]ChartTemplate, error) {
	return []ChartTemplate{
		{
			Name:        "Vendor Price Comparison",
			Description: "Bar chart comparing prices across vendors for a given SKU",
			ChartType:   "bar",
			ColorScheme: "default",
		},
		{
			Name:        "Price Trend Over Time",
			Description: "Line chart showing price changes for a vendor over a date range",
			ChartType:   "line",
			ColorScheme: "default",
		},
		{
			Name:        "Spend Distribution",
			Description: "Pie chart breaking down total spend by vendor category",
			ChartType:   "pie",
			ColorScheme: "default",
		},
		{
			Name:        "Savings Over Time",
			Description: "Area chart illustrating accumulated savings across quarters",
			ChartType:   "area",
			ColorScheme: "default",
		},
		{
			Name:        "Category Correlation",
			Description: "Scatter plot correlating vendor list price with typical discount percentage",
			ChartType:   "scatter",
			ColorScheme: "default",
		},
	}, nil
}
