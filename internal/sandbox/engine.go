package sandbox

import (
	"context"
	"fmt"
)

// Engine provides predefined sandbox templates and simulates tool executions.
type Engine struct {
	templates []SandboxTemplate
}

// NewEngine creates a new Engine with predefined templates.
func NewEngine() *Engine {
	return &Engine{
		templates: []SandboxTemplate{
			{
				Name:          "pricing",
				Description:   "Query fair market price for a vendor's product",
				ToolName:      "negotiate_query_price",
				ExampleParams: `{"vendor": "Slack", "sku": "pro", "quantity": 100}`,
			},
			{
				Name:          "negotiation",
				Description:   "Run a negotiation session with strategy profile",
				ToolName:      "negotiate_create_session",
				ExampleParams: `{"vendor": "GitHub", "strategy": "balanced", "budget": 15}`,
			},
			{
				Name:          "contract",
				Description:   "Register a new SaaS contract for renewal tracking",
				ToolName:      "negotiate_add_contract",
				ExampleParams: `{"vendor": "Slack", "sku": "pro", "seats": 50, "current_price_per_unit": 12.5, "renewal_date": "2026-12-31T00:00:00Z"}`,
			},
			{
				Name:          "vendor_comparison",
				Description:   "Compare pricing across multiple vendors",
				ToolName:      "negotiate_vendor_comparison",
				ExampleParams: `{"vendor_a": "Slack", "vendor_b": "Teams", "sku": "enterprise"}`,
			},
			{
				Name:          "savings",
				Description:   "Estimate potential savings for a vendor",
				ToolName:      "negotiate_calculate_savings",
				ExampleParams: `{"vendor": "AWS", "current_spend": 500000}`,
			},
		},
	}
}

// GetTemplates returns all predefined sandbox templates.
func (e *Engine) GetTemplates(ctx context.Context) ([]SandboxTemplate, error) {
	return e.templates, nil
}

// Execute simulates running a tool in the sandbox by returning a formatted result.
func (e *Engine) Execute(ctx context.Context, toolName, params string) (string, error) {
	result := fmt.Sprintf("Executed %s with params: %s", toolName, params)
	return result, nil
}
