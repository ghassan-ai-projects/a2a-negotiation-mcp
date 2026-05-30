package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
        "strings"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/miner"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/parallel"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NegotiationServer wraps the MCP server with business logic.
type NegotiationServer struct {
	mcpServer      *mcpserver.MCPServer
	pricingStore   *pricing.Store
	negotiationEng *negotiation.Engine
	historyStore   *history.Store
	minerEng       *miner.Engine
	logger         *slog.Logger
}

// NewNegotiationServer creates a new MCP negotiation server.
func NewNegotiationServer(pricingStore *pricing.Store, historyStore *history.Store, logger *slog.Logger) *NegotiationServer {
	eng := negotiation.NewEngine(pricingStore)
	miningEng := miner.NewEngine(pricingStore, logger)

	ns := &NegotiationServer{
		mcpServer: mcpserver.NewMCPServer(
			"a2a-negotiation-mcp",
			"1.0.0",
			mcpserver.WithToolCapabilities(true),
			mcpserver.WithResourceCapabilities(true, true),
			mcpserver.WithLogging(),
		),
		pricingStore:   pricingStore,
		negotiationEng: eng,
		historyStore:   historyStore,
		minerEng:       miningEng,
		logger:         logger,
	}

	ns.registerTools()
	ns.registerResources()
	return ns
}

// MCPServer returns the underlying MCP server.
func (ns *NegotiationServer) MCPServer() *mcpserver.MCPServer {
	return ns.mcpServer
}

func (ns *NegotiationServer) registerTools() {
	// Tool 1: query_price
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_query_price",
		mcp.WithDescription("Query fair market price range for a SaaS vendor's product. Returns market range, suggested counter-offer, and confidence level."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name (e.g., Slack, GitHub, Salesforce)")),
		mcp.WithString("sku", mcp.Description("Product SKU (optional — returns any match if omitted)")),
		mcp.WithInteger("quantity", mcp.Description("Number of seats/units")),
		mcp.WithInteger("term_months", mcp.Description("Contract term in months")),
	), ns.handleQueryPrice)

	// Tool 2: calculate_savings
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_calculate_savings",
		mcp.WithDescription("Estimate potential savings for a vendor based on current spend and market data."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithNumber("current_spend", mcp.Required(), mcp.Description("Your current annual spend with this vendor")),
	), ns.handleCalculateSavings)

	// Tool 3: create_session
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_create_session",
		mcp.WithDescription("Start a new negotiation session with a specific strategy profile. Returns a session ID for tracking through all rounds."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Description("Product SKU to negotiate (optional)")),
		mcp.WithString("strategy", mcp.Required(), mcp.Description("Negotiation strategy: aggressive, balanced, or conservative")),
		mcp.WithNumber("budget", mcp.Description("Maximum budget per unit (optional)")),
		mcp.WithObject("constraints", mcp.Description("Additional constraints as key-value pairs (optional)")),
	), ns.handleCreateSession)

	// Tool 4: run
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_run",
		mcp.WithDescription("Execute the multi-round negotiation loop for a session. Each round generates a counter-offer. Auto-approves if offer meets threshold."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID from negotiate_create_session")),
		mcp.WithNumber("auto_approve_threshold", mcp.Description("Auto-accept if offer is at or below this amount (optional)")),
	), ns.handleRunNegotiation)

	// Tool 5: history
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_history",
		mcp.WithDescription("View negotiation history and performance metrics. Filter by vendor and time period."),
		mcp.WithString("vendor", mcp.Description("Filter by vendor name (optional)")),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 1y, or all (default: all)")),
	), ns.handleHistory)

	// Tool 6: strategies
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_strategies",
		mcp.WithDescription("List available negotiation strategy profiles with descriptions and aggressiveness levels."),
	), ns.handleStrategies)

        // Tool 7: negotiate_run_parallel
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_run_parallel",
                mcp.WithDescription("Run parallel negotiations across multiple sessions. Evaluates each session concurrently and selects the best result based on the chosen strategy."),
                mcp.WithArray("sessions", mcp.Required(), mcp.WithStringItems(), mcp.Description("Array of session IDs to negotiate in parallel (required)")),
                mcp.WithString("strategy", mcp.Required(), mcp.Description("Selection strategy: best_price, best_discount, or fastest")),
                mcp.WithInteger("timeout", mcp.Description("Timeout per session in seconds (optional, default 30)")),
        ), ns.handleRunParallel)
}

// ─── Tool Handlers ───

func (ns *NegotiationServer) handleQueryPrice(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	vendor, _ := req.RequireString("vendor")
	sku := req.GetString("sku", "")
	quantity := req.GetInt("quantity", 0)
	termMonths := req.GetInt("term_months", 12)

	ns.logger.Debug("query_price called", "vendor", vendor, "sku", sku, "qty", quantity, "term", termMonths)

	result, err := ns.pricingStore.GetPricingByVendorSKU(ctx, vendor, sku)
	if err != nil {
		ns.logger.Warn("query_price failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Pricing lookup failed: %s", err.Error())), nil
	}

	qtyMultiplier := 1.0
	if quantity >= 1000 {
		qtyMultiplier = 1.5
	} else if quantity >= 100 {
		qtyMultiplier = 1.25
	}
	termMultiplier := 1.0
	if termMonths >= 36 {
		termMultiplier = 1.3
	} else if termMonths >= 24 {
		termMultiplier = 1.15
	}

	factor := qtyMultiplier + termMultiplier - 1
	result.SuggestedMin = result.SuggestedMin - (result.ListPrice-result.SuggestedMin)*0.1*factor
	result.SuggestedMax = result.SuggestedMax - (result.ListPrice-result.SuggestedMax)*0.05*factor
	result.SuggestedMin = float64(int(result.SuggestedMin*100)) / 100
	result.SuggestedMax = float64(int(result.SuggestedMax*100)) / 100

	resp := map[string]any{
		"vendor":              vendor,
		"sku":                 result.SKU,
		"list_price":          result.ListPrice,
		"market_price_range":  []float64{result.MarketMin, result.MarketMax},
		"suggested_counter_offer": map[string]float64{"min": result.SuggestedMin, "max": result.SuggestedMax},
		"data_points_count":   result.DataPoints + 1,
		"confidence":          result.Confidence,
		"typical_discount_pct": result.TypicalPct,
		"duration_ms":         time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCalculateSavings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	vendor, _ := req.RequireString("vendor")
	currentSpend := req.GetFloat("current_spend", 0)

	ns.logger.Debug("calculate_savings called", "vendor", vendor, "spend", currentSpend)

	estimate, err := ns.negotiationEng.ComputeSavings(ctx, vendor, currentSpend)
	if err != nil {
		ns.logger.Warn("calculate_savings failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Savings calculation failed: %s", err.Error())), nil
	}

	similarDeals, err := ns.historyStore.GetSimilarDeals(ctx, vendor, 5)
	if err == nil && len(similarDeals) > 0 {
		for _, d := range similarDeals {
			estimate.SimilarDeals = append(estimate.SimilarDeals, pricing.SimilarDeal{
				Vendor: d.Vendor, DiscountPct: d.DiscountPct,
				Seats: d.Seats, TermMonths: d.TermMonths, FinalPrice: d.FinalPrice,
			})
		}
	}

	resp := map[string]any{
		"vendor":               vendor,
		"current_spend":        estimate.CurrentSpend,
		"estimated_savings":    estimate.EstimatedSavings,
		"savings_percentage":   estimate.SavingsPercentage,
		"confidence":           estimate.Confidence,
		"market_average_price": estimate.MarketAveragePrice,
		"duration_ms":          time.Since(start).Milliseconds(),
	}
	if len(estimate.SimilarDeals) > 0 {
		resp["similar_deals"] = estimate.SimilarDeals
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCreateSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	vendor, _ := req.RequireString("vendor")
	sku := req.GetString("sku", "")
	strategyName, _ := req.RequireString("strategy")
	budget := req.GetFloat("budget", 0)

	rawConstraints, _ := req.GetArguments()["constraints"]
	constraints, _ := rawConstraints.(map[string]any)

	ns.logger.Debug("create_session called", "vendor", vendor, "sku", sku, "strategy", strategyName, "budget", budget)

	session, err := ns.negotiationEng.CreateSession(ctx, vendor, sku, strategyName, budget, constraints)
	if err != nil {
		ns.logger.Warn("create_session failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Session creation failed: %s", err.Error())), nil
	}

	session.ID = uuid.New().String()

	strategy := negotiation.GetStrategy(strategyName)

	histSess := &history.SessionRecord{
		ID: session.ID, Vendor: session.Vendor, SKU: session.SKU,
		Strategy: session.Strategy, Budget: session.Budget, Status: session.Status,
		CurrentOffer: session.CurrentOffer, ListPrice: session.ListPrice,
		RoundsComplete: session.RoundsComplete, Outcome: session.Outcome,
		CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt,
	}
	if err := ns.historyStore.SaveSession(ctx, histSess); err != nil {
		ns.logger.Error("failed to save session", "error", err.Error())
	}

	strategyDesc := "No strategy description available"
	if strategy != nil {
		strategyDesc = strategy.Description
	}

	resp := map[string]any{
		"session_id":           session.ID,
		"initial_offer":        session.CurrentOffer,
		"list_price":           session.ListPrice,
		"strategy":             strategyName,
		"strategy_description": strategyDesc,
		"created_at":           session.CreatedAt.Format(time.RFC3339),
		"duration_ms":          time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRunNegotiation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	sessionID, _ := req.RequireString("session_id")
	autoApproveThreshold := req.GetFloat("auto_approve_threshold", 0)

	ns.logger.Debug("run_negotiation called", "session_id", sessionID, "threshold", autoApproveThreshold)

	sessRec, err := ns.historyStore.GetSession(ctx, sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", err.Error())), nil
	}

	session := &negotiation.Session{
		ID: sessRec.ID, Vendor: sessRec.Vendor, SKU: sessRec.SKU,
		Strategy: sessRec.Strategy, Budget: sessRec.Budget, Status: sessRec.Status,
		CurrentOffer: sessRec.CurrentOffer, ListPrice: sessRec.ListPrice,
		RoundsComplete: sessRec.RoundsComplete, Outcome: sessRec.Outcome,
		CreatedAt: sessRec.CreatedAt, UpdatedAt: sessRec.UpdatedAt,
	}

	result, rounds, err := ns.negotiationEng.RunNegotiation(ctx, session, 0, autoApproveThreshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Negotiation failed: %s", err.Error())), nil
	}

	sessRec.Status = session.Status
	sessRec.CurrentOffer = session.CurrentOffer
	sessRec.RoundsComplete = session.RoundsComplete
	sessRec.Outcome = session.Outcome
	sessRec.UpdatedAt = session.UpdatedAt
	if err := ns.historyStore.UpdateSession(ctx, sessRec); err != nil {
		ns.logger.Error("failed to update session", "error", err.Error())
	}

	var roundRecords []history.RoundRecord
	for _, r := range rounds {
		roundRecords = append(roundRecords, history.RoundRecord{
			SessionID: r.SessionID, RoundNumber: r.RoundNumber, Offer: r.Offer,
			DiscountPct: r.DiscountPct, Counterparty: r.Counterparty, Note: r.Note,
			CreatedAt: r.CreatedAt,
		})
	}
	if err := ns.historyStore.SaveRounds(ctx, roundRecords); err != nil {
		ns.logger.Error("failed to save rounds", "error", err.Error())
	}

	if session.Outcome == "accepted" {
		deal := &history.DealOutcome{
			Vendor: session.Vendor, SKU: session.SKU, ListPrice: session.ListPrice,
			FinalPrice: session.CurrentOffer, DiscountPct: result.TotalDiscount,
			Seats: 0, TermMonths: 12, Strategy: session.Strategy,
			SessionID: session.ID, CreatedAt: time.Now().UTC(),
		}
		if err := ns.historyStore.SaveDealOutcome(ctx, deal); err != nil {
			ns.logger.Error("failed to save deal outcome", "error", err.Error())
		}
	}

	type roundInfo struct {
		RoundNumber  int     `json:"round_number"`
		Offer        float64 `json:"offer"`
		DiscountPct  float64 `json:"discount_percentage"`
		Counterparty string  `json:"counterparty"`
		Note         string  `json:"note"`
		Timestamp    string  `json:"timestamp"`
	}
	var historyResp []roundInfo
	for _, r := range rounds {
		historyResp = append(historyResp, roundInfo{
			RoundNumber: r.RoundNumber, Offer: r.Offer, DiscountPct: r.DiscountPct,
			Counterparty: r.Counterparty, Note: r.Note, Timestamp: r.CreatedAt.Format(time.RFC3339),
		})
	}

	resp := map[string]any{
		"status":             result.Status,
		"current_offer":      result.CurrentOffer,
		"rounds_completed":   result.RoundsComplete,
		"outcome":            result.Outcome,
		"list_price":         result.ListPrice,
		"total_discount_pct": result.TotalDiscount,
		"history":            historyResp,
		"duration_ms":        time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	vendor := req.GetString("vendor", "")
	period := req.GetString("period", "all")

	ns.logger.Debug("history called", "vendor", vendor, "period", period)

	summary, err := ns.historyStore.GetHistory(ctx, vendor, period)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("History query failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"total_deals":              summary.TotalDeals,
		"win_rate":                 summary.WinRate,
		"avg_discount_percentage":  summary.AvgDiscountPct,
		"total_savings":            summary.TotalSavings,
		"deals":                    summary.Deals,
		"duration_ms":              time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleStrategies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	strategies := negotiation.AvailableStrategies()

	var strategyList []negotiation.StrategyInfo
	for _, s := range strategies {
		strategyList = append(strategyList, negotiation.StrategyInfo{
			Name: s.Name, Description: s.Description,
			Aggressiveness: s.Aggressiveness, IdealFor: s.IdealFor,
		})
	}

	resp := map[string]any{
		"strategies":  strategyList,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDiscoverOpportunities(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        businessName, _ := req.RequireString("business_name")
        description, _ := req.RequireString("description")
        industry := req.GetString("industry", "")
        employees := int(req.GetFloat("employees", 0))
        rawVendors, _ := req.GetArguments()["vendors"]
	vendorsList, _ := rawVendors.([]any)

        ns.logger.Debug("discover_opportunities called",
                "name", businessName, "industry", industry, "employees", employees)

        vendors := make([]string, 0, len(vendorsList))
        for _, v := range vendorsList {
                if s, ok := v.(string); ok {
                        vendors = append(vendors, s)
                }
        }

        profile := miner.BusinessProfile{
                Name:        businessName,
                Description: description,
                Employees:   employees,
                Industry:    industry,
                Vendors:     vendors,
        }

        opportunities, err := ns.minerEng.DiscoverOpportunities(ctx, profile)
        if err != nil {
                ns.logger.Error("discover_opportunities failed", "error", err.Error())
                return mcp.NewToolResultError(fmt.Sprintf("Opportunity discovery failed: %s", err.Error())), nil
        }

        resp := map[string]any{
                "business_name": businessName,
                "opportunities": opportunities,
                "opportunity_count": len(opportunities),
                "duration_ms":  time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) jsonResult(data map[string]any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		ns.logger.Error("failed to marshal JSON result", "error", err.Error())
		return mcp.NewToolResultError("internal serialization error"), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: string(b)},
		},
	}, nil
}

func (ns *NegotiationServer) handleRunParallel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	rawSessions, ok := req.GetArguments()["sessions"]
	if !ok {
		return mcp.NewToolResultError("sessions is required"), nil
	}
	sessionsList, ok := rawSessions.([]any)
	if !ok || len(sessionsList) == 0 {
		return mcp.NewToolResultError("sessions must be a non-empty array of vendor strings (e.g. 'Vendor' or 'Vendor/SKU')"), nil
	}

	strategy, _ := req.RequireString("strategy")
	if strategy == "" {
		strategy = "best_price"
	}
	timeout := req.GetInt("timeout", 30)

	ns.logger.Debug("run_parallel called", "sessions", sessionsList, "strategy", strategy, "timeout", timeout)

	// Create sessions for each vendor/SKU
	var sessionIDs []string
	for _, raw := range sessionsList {
		entry, ok := raw.(string)
		if !ok || entry == "" {
			continue
		}

		vendor := entry
		sku := ""
		if idx := strings.Index(entry, "/"); idx >= 0 {
			vendor = entry[:idx]
			sku = entry[idx+1:]
		}

		session, err := ns.negotiationEng.CreateSession(ctx, vendor, sku, strategy, 0, nil)
		if err != nil {
			ns.logger.Warn("run_parallel: failed to create session", "vendor", vendor, "error", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("Session creation failed for %s: %s", vendor, err.Error())), nil
		}

		session.ID = uuid.New().String()

		histSess := &history.SessionRecord{
			ID: session.ID, Vendor: session.Vendor, SKU: session.SKU,
			Strategy: session.Strategy, Budget: session.Budget, Status: session.Status,
			CurrentOffer: session.CurrentOffer, ListPrice: session.ListPrice,
			RoundsComplete: session.RoundsComplete, Outcome: session.Outcome,
			CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt,
		}
		if err := ns.historyStore.SaveSession(ctx, histSess); err != nil {
			ns.logger.Error("run_parallel: failed to save session", "error", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("Failed to save session: %s", err.Error())), nil
		}

		sessionIDs = append(sessionIDs, session.ID)
	}

	if len(sessionIDs) == 0 {
		return mcp.NewToolResultError("No valid sessions could be created from the input"), nil
	}

	// Build parallel engine
	parEng := parallel.NewEngine(ns.negotiationEng, ns.historyStore, ns.pricingStore, ns.logger)

	cfg := parallel.ParallelConfig{
		SessionIDs: sessionIDs,
		Strategy:   strategy,
		Timeout:    timeout,
	}

	result, err := parEng.RunParallel(ctx, cfg)
	if err != nil {
		ns.logger.Warn("run_parallel failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Parallel negotiation failed: %s", err.Error())), nil
	}

	// Add duration
	result.DurationMs = time.Since(start).Milliseconds()

	resp := map[string]any{
		"winner_session_id":  result.WinnerSessionID,
		"winner_vendor":      result.WinnerVendor,
		"winner_offer":       result.WinnerOffer,
		"winner_discount_pct": result.WinnerDiscount,
		"strategy":           result.Strategy,
		"total_rounds":       result.TotalRounds,
		"all_results":        result.AllResults,
		"duration_ms":        result.DurationMs,
	}
	return ns.jsonResult(resp)
}
