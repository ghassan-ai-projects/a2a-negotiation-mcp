package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/mark3labs/mcp-go/mcp"
)

func (ns *NegotiationServer) registerResources() {
	// Resource 1: negotiate://pricing/{vendor}/{sku}
	ns.mcpServer.AddResourceTemplate(mcp.NewResourceTemplate(
		"negotiate://pricing/{vendor}/{sku}",
		"Market pricing for a vendor SKU",
		mcp.WithTemplateDescription("Current market pricing data for a specific vendor product"),
		mcp.WithTemplateMIMEType("application/json"),
	), ns.handlePricingResource)

	// Resource 2: negotiate://session/{session_id}
	ns.mcpServer.AddResourceTemplate(mcp.NewResourceTemplate(
		"negotiate://session/{session_id}",
		"Negotiation session details",
		mcp.WithTemplateDescription("Full negotiation history and current state for a session"),
		mcp.WithTemplateMIMEType("application/json"),
	), ns.handleSessionResource)

	// Resource 3: negotiate://history/{period}
	ns.mcpServer.AddResourceTemplate(mcp.NewResourceTemplate(
		"negotiate://history/{period}",
		"Aggregated negotiation history",
		mcp.WithTemplateDescription("Aggregated performance statistics over a time period (30d, 90d, 1y, or all)"),
		mcp.WithTemplateMIMEType("application/json"),
	), ns.handleHistoryResource)

	// Resource 4: negotiate://strategies
	ns.mcpServer.AddResource(mcp.NewResource(
		"negotiate://strategies",
		"Available negotiation strategies",
		mcp.WithResourceDescription("List of all available negotiation strategy profiles"),
		mcp.WithMIMEType("application/json"),
	), ns.handleStrategiesResource)
}

// ─── Resource Handlers ───

func (ns *NegotiationServer) handlePricingResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	vendor, _ := req.Params.Arguments["vendor"].(string)
	sku, _ := req.Params.Arguments["sku"].(string)

	if vendor == "" {
		return nil, fmt.Errorf("vendor is required")
	}

	result, err := ns.pricingStore.GetPricingByVendorSKU(ctx, vendor, sku)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"uri":                  req.Params.URI,
		"vendor":               result.Vendor,
		"sku":                  result.SKU,
		"list_price":           result.ListPrice,
		"market_min":           result.MarketMin,
		"market_max":           result.MarketMax,
		"suggested_min":        result.SuggestedMin,
		"suggested_max":        result.SuggestedMax,
		"typical_discount_pct": result.TypicalPct,
		"confidence":           result.Confidence,
		"description":          result.Description,
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}

func (ns *NegotiationServer) handleSessionResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	sessionID, _ := req.Params.Arguments["session_id"].(string)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	sess, err := ns.historyStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	rounds, err := ns.historyStore.GetRounds(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"uri":     req.Params.URI,
		"session": sess,
		"rounds":  rounds,
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}

func (ns *NegotiationServer) handleHistoryResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	period, _ := req.Params.Arguments["period"].(string)
	if period == "" {
		period = "all"
	}

	summary, err := ns.historyStore.GetHistory(ctx, "", period)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"uri":                    req.Params.URI,
		"total_deals":            summary.TotalDeals,
		"win_rate":               summary.WinRate,
		"avg_discount_percentage": summary.AvgDiscountPct,
		"total_savings":          summary.TotalSavings,
		"deals":                  summary.Deals,
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}

func (ns *NegotiationServer) handleStrategiesResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	strategies := negotiation.AvailableStrategies()
	var data []negotiation.StrategyInfo
	for _, s := range strategies {
		data = append(data, negotiation.StrategyInfo{
			Name: s.Name, Description: s.Description,
			Aggressiveness: s.Aggressiveness, IdealFor: s.IdealFor,
		})
	}

	resp := map[string]any{
		"uri":        req.Params.URI,
		"strategies": data,
	}

	b, _ := json.MarshalIndent(resp, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		},
	}, nil
}
