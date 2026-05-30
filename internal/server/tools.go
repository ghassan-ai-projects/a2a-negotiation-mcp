package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/group"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/miner"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/parallel"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sell"
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
	groupEng       *group.Engine
	sellEng        *sell.Engine
	calendarEng    *calendar.Engine
	logger         *slog.Logger
}

// NewNegotiationServer creates a new MCP negotiation server.
func NewNegotiationServer(pricingStore *pricing.Store, historyStore *history.Store, groupEngine *group.Engine, sellEngine *sell.Engine, calendarEngine *calendar.Engine, logger *slog.Logger) *NegotiationServer {
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
		groupEng:       groupEngine,
		sellEng:        sellEngine,
		calendarEng:    calendarEngine,
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

	// Tool 8: negotiate_create_group
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_create_group",
		mcp.WithDescription("Create a collective buying group targeting a specific vendor SKU."),
		mcp.WithString("target_vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("target_sku", mcp.Required(), mcp.Description("Product SKU")),
		mcp.WithInteger("min_members", mcp.Description("Minimum members required (default 2)")),
		mcp.WithInteger("expires_in_hours", mcp.Description("Group expiration in hours (default 72)")),
	), ns.handleCreateGroup)

	// Tool 9: negotiate_join_group
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_join_group",
		mcp.WithDescription("Join an existing collective buying group."),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group ID")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User identifier")),
		mcp.WithInteger("quantity", mcp.Description("Number of seats/units")),
		mcp.WithNumber("max_price", mcp.Description("Maximum price per unit (optional)")),
	), ns.handleJoinGroup)

	// Tool 10: negotiate_compute_offer
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_compute_offer",
		mcp.WithDescription("Compute a consolidated collective buying offer for a group."),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group ID")),
	), ns.handleComputeOffer)

	// Tool 11: negotiate_group_status
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_group_status",
		mcp.WithDescription("View buying group details and member list."),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group ID")),
	), ns.handleGroupStatus)

	// Tool 12: negotiate_add_contract
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_add_contract",
		mcp.WithDescription("Register a new SaaS contract for renewal tracking."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name (e.g., Slack, GitHub)")),
		mcp.WithString("sku", mcp.Required(), mcp.Description("Product SKU")),
		mcp.WithInteger("seats", mcp.Required(), mcp.Description("Number of seats/units")),
		mcp.WithNumber("current_price_per_unit", mcp.Required(), mcp.Description("Current price per seat/unit")),
		mcp.WithString("renewal_date", mcp.Required(), mcp.Description("Renewal date (RFC3339, e.g. 2026-06-15T00:00:00Z)")),
	), ns.handleAddContract)

	// Tool 13: negotiate_list_contracts
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_contracts",
		mcp.WithDescription("List registered contracts with optional filters."),
		mcp.WithString("vendor", mcp.Description("Filter by vendor name")),
		mcp.WithString("status", mcp.Description("Filter by status: active, negotiating, renewed, cancelled")),
		mcp.WithInteger("expiring_soon", mcp.Description("Only contracts expiring within this many days")),
	), ns.handleListContracts)

	// Tool 14: negotiate_check_renewals
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_renewals",
		mcp.WithDescription("Check upcoming contract renewals with urgency classification and savings estimates."),
		mcp.WithInteger("days_ahead", mcp.Required(), mcp.Description("Number of days to look ahead (e.g. 90)")),
	), ns.handleCheckRenewals)

	// Tool 15: negotiate_trigger_renewal
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_trigger_renewal",
		mcp.WithDescription("Trigger an automatic negotiation for a contract about to renew."),
		mcp.WithString("contract_id", mcp.Required(), mcp.Description("Contract ID to negotiate")),
	), ns.handleTriggerRenewal)

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
		"vendor":                  vendor,
		"sku":                     result.SKU,
		"list_price":              result.ListPrice,
		"market_price_range":      []float64{result.MarketMin, result.MarketMax},
		"suggested_counter_offer": map[string]float64{"min": result.SuggestedMin, "max": result.SuggestedMax},
		"data_points_count":       result.DataPoints + 1,
		"confidence":              result.Confidence,
		"typical_discount_pct":    result.TypicalPct,
		"duration_ms":             time.Since(start).Milliseconds(),
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
		"total_deals":             summary.TotalDeals,
		"win_rate":                summary.WinRate,
		"avg_discount_percentage": summary.AvgDiscountPct,
		"total_savings":           summary.TotalSavings,
		"deals":                   summary.Deals,
		"duration_ms":             time.Since(start).Milliseconds(),
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
		"business_name":     businessName,
		"opportunities":     opportunities,
		"opportunity_count": len(opportunities),
		"duration_ms":       time.Since(start).Milliseconds(),
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
		"winner_session_id":   result.WinnerSessionID,
		"winner_vendor":       result.WinnerVendor,
		"winner_offer":        result.WinnerOffer,
		"winner_discount_pct": result.WinnerDiscount,
		"strategy":            result.Strategy,
		"total_rounds":        result.TotalRounds,
		"all_results":         result.AllResults,
		"duration_ms":         result.DurationMs,
	}
	return ns.jsonResult(resp)
}

// ─── Collective Buying Group Handlers ───

func (ns *NegotiationServer) handleCreateGroup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("target_vendor")
	sku, _ := req.RequireString("target_sku")
	minMembers := int(req.GetInt("min_members", 2))
	expiresInHours := int(req.GetInt("expires_in_hours", 72))

	if minMembers < 1 {
		minMembers = 1
	}
	if expiresInHours < 1 {
		expiresInHours = 1
	}

	ns.logger.Debug("create_group called", "vendor", vendor, "sku", sku, "min_members", minMembers, "expires_in", expiresInHours)

	group, err := ns.groupEng.CreateGroup(ctx, vendor, sku, minMembers, expiresInHours)
	if err != nil {
		ns.logger.Warn("create_group failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Group creation failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"group_id":      group.ID,
		"target_vendor": group.TargetVendor,
		"target_sku":    group.TargetSKU,
		"min_members":   group.MinMembers,
		"status":        group.Status,
		"created_at":    group.CreatedAt.Format(time.RFC3339),
		"expires_at":    group.ExpiresAt.Format(time.RFC3339),
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleJoinGroup(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	groupID, _ := req.RequireString("group_id")
	userID, _ := req.RequireString("user_id")
	quantity := int(req.GetInt("quantity", 1))
	maxPrice := req.GetFloat("max_price", 0)

	if quantity < 1 {
		quantity = 1
	}

	ns.logger.Debug("join_group called", "group_id", groupID, "user_id", userID, "quantity", quantity)

	member, err := ns.groupEng.JoinGroup(ctx, groupID, userID, quantity, maxPrice)
	if err != nil {
		ns.logger.Warn("join_group failed", "group_id", groupID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Join group failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"membership_id": member.ID,
		"group_id":      member.GroupID,
		"user_id":       member.UserID,
		"quantity":      member.Quantity,
		"max_price":     member.MaxPrice,
		"committed_at":  member.CommittedAt.Format(time.RFC3339),
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleComputeOffer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	groupID, _ := req.RequireString("group_id")

	ns.logger.Debug("compute_offer called", "group_id", groupID)

	offer, err := ns.groupEng.ComputeOffer(ctx, groupID)
	if err != nil {
		ns.logger.Warn("compute_offer failed", "group_id", groupID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Compute offer failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"group_id":         offer.GroupID,
		"vendor":           offer.Vendor,
		"sku":              offer.SKU,
		"total_demand":     offer.TotalDemand,
		"member_count":     offer.MemberCount,
		"list_price":       offer.ListPrice,
		"offer_price":      offer.OfferPrice,
		"savings_per_unit": offer.SavingsPerUnit,
		"discount_pct":     offer.DiscountPct,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGroupStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	groupID, _ := req.RequireString("group_id")

	ns.logger.Debug("group_status called", "group_id", groupID)

	group, err := ns.groupEng.GroupStore().GetGroup(ctx, groupID)
	if err != nil {
		ns.logger.Warn("group_status failed", "group_id", groupID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Group lookup failed: %s", err.Error())), nil
	}

	members, err := ns.groupEng.GroupStore().GetMembers(ctx, groupID)
	if err != nil {
		ns.logger.Warn("group_status: failed to get members", "group_id", groupID, "error", err.Error())
	}

	resp := map[string]any{
		"group": map[string]any{
			"id":            group.ID,
			"target_vendor": group.TargetVendor,
			"target_sku":    group.TargetSKU,
			"target_price":  group.TargetPrice,
			"min_members":   group.MinMembers,
			"status":        group.Status,
			"created_at":    group.CreatedAt.Format(time.RFC3339),
			"expires_at":    group.ExpiresAt.Format(time.RFC3339),
		},
		"members":      members,
		"member_count": len(members),
		"duration_ms":  time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Sell-Side Handlers ───

func (ns *NegotiationServer) handleListItem(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	title, _ := req.RequireString("title")
	desiredPrice := req.GetFloat("desired_price", 0)
	strategy, _ := req.RequireString("strategy")
	minPrice := req.GetFloat("min_price", 0)
	expiresInHours := int(req.GetInt("expires_in_hours", 0))

	// Default user_id for MCP context (actual auth would come from client)
	userID := req.GetString("user_id", "anonymous")

	ns.logger.Debug("list_item called", "title", title, "strategy", strategy, "desired_price", desiredPrice)

	listing, err := ns.sellEng.ListItem(ctx, userID, title, "", desiredPrice, minPrice, strategy, expiresInHours)
	if err != nil {
		ns.logger.Warn("list_item failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("List item failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"listing_id":    listing.ID,
		"title":         listing.Title,
		"desired_price": listing.DesiredPrice,
		"min_price":     listing.MinPrice,
		"strategy":      listing.Strategy,
		"status":        listing.Status,
		"created_at":    listing.CreatedAt.Format(time.RFC3339),
		"expires_at":    listing.ExpiresAt.Format(time.RFC3339),
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handlePlaceBid(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")
	bidderID, _ := req.RequireString("bidder_id")
	amount := req.GetFloat("amount", 0)
	message := req.GetString("message", "")

	ns.logger.Debug("place_bid called", "listing_id", listingID, "bidder_id", bidderID, "amount", amount)

	bid, err := ns.sellEng.PlaceBid(ctx, listingID, bidderID, amount, message)
	if err != nil {
		ns.logger.Warn("place_bid failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Place bid failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"bid_id":      bid.ID,
		"listing_id":  bid.ListingID,
		"bidder_id":   bid.BidderID,
		"amount":      bid.Amount,
		"message":     bid.Message,
		"created_at":  bid.CreatedAt.Format(time.RFC3339),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAcceptBid(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")
	bidID, _ := req.RequireString("bid_id")

	ns.logger.Debug("accept_bid called", "listing_id", listingID, "bid_id", bidID)

	result, err := ns.sellEng.AcceptBid(ctx, listingID, bidID)
	if err != nil {
		ns.logger.Warn("accept_bid failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Accept bid failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"listing": map[string]any{
			"id":            result.Listing.ID,
			"user_id":       result.Listing.UserID,
			"title":         result.Listing.Title,
			"desired_price": result.Listing.DesiredPrice,
			"strategy":      result.Listing.Strategy,
			"status":        result.Listing.Status,
		},
		"winning_bid": map[string]any{
			"id":        result.WinningBid.ID,
			"bidder_id": result.WinningBid.BidderID,
			"amount":    result.WinningBid.Amount,
		},
		"status":      result.Status,
		"final_price": result.FinalPrice,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRejectBid(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")
	bidID, _ := req.RequireString("bid_id")
	counterMessage := req.GetString("counter_message", "")

	ns.logger.Debug("reject_bid called", "listing_id", listingID, "bid_id", bidID)

	if err := ns.sellEng.RejectBid(ctx, listingID, bidID, counterMessage); err != nil {
		ns.logger.Warn("reject_bid failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Reject bid failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"status":          "rejected",
		"listing_id":      listingID,
		"bid_id":          bidID,
		"counter_message": counterMessage,
		"duration_ms":     time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListingStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")

	ns.logger.Debug("listing_status called", "listing_id", listingID)

	listing, err := ns.sellEng.Store().GetListing(ctx, listingID)
	if err != nil {
		ns.logger.Warn("listing_status failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Listing lookup failed: %s", err.Error())), nil
	}

	bids, err := ns.sellEng.Store().GetBids(ctx, listingID)
	if err != nil {
		ns.logger.Warn("listing_status: failed to get bids", "listing_id", listingID, "error", err.Error())
	}

	resp := map[string]any{
		"listing": map[string]any{
			"id":            listing.ID,
			"user_id":       listing.UserID,
			"title":         listing.Title,
			"description":   listing.Description,
			"desired_price": listing.DesiredPrice,
			"min_price":     listing.MinPrice,
			"strategy":      listing.Strategy,
			"status":        listing.Status,
			"created_at":    listing.CreatedAt.Format(time.RFC3339),
			"expires_at":    listing.ExpiresAt.Format(time.RFC3339),
		},
		"bids":        bids,
		"bid_count":   len(bids),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Calendar / Renewal Contract Handlers ───

func (ns *NegotiationServer) handleAddContract(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	seats := int(req.GetInt("seats", 0))
	currentPrice := req.GetFloat("current_price_per_unit", 0)
	renewalDateStr, _ := req.RequireString("renewal_date")

	ns.logger.Debug("add_contract called", "vendor", vendor, "sku", sku, "seats", seats)

	renewalDate, err := time.Parse(time.RFC3339, renewalDateStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid renewal_date: must be RFC3339 format (e.g. 2026-06-15T00:00:00Z)")), nil
	}

	contract := &calendar.Contract{
		UserID:       "default",
		Vendor:       vendor,
		SKU:          sku,
		Seats:        seats,
		CurrentPrice: currentPrice,
		RenewalDate:  renewalDate,
		Status:       "active",
	}

	if err := ns.calendarEng.Store().CreateContract(ctx, contract); err != nil {
		ns.logger.Warn("add_contract failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create contract: %s", err.Error())), nil
	}

	resp := map[string]any{
		"contract_id":            contract.ID,
		"vendor":                 contract.Vendor,
		"sku":                    contract.SKU,
		"seats":                  contract.Seats,
		"current_price_per_unit": contract.CurrentPrice,
		"renewal_date":           contract.RenewalDate.Format(time.RFC3339),
		"status":                 contract.Status,
		"created_at":             contract.CreatedAt.Format(time.RFC3339),
		"duration_ms":            time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListContracts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	filter := calendar.ContractFilter{
		Vendor:       req.GetString("vendor", ""),
		Status:       req.GetString("status", ""),
		ExpiringSoon: int(req.GetInt("expiring_soon", 0)),
	}

	ns.logger.Debug("list_contracts called", "vendor", filter.Vendor, "status", filter.Status, "expiring_soon", filter.ExpiringSoon)

	contracts, err := ns.calendarEng.Store().ListContracts(ctx, filter)
	if err != nil {
		ns.logger.Warn("list_contracts failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list contracts: %s", err.Error())), nil
	}

	if contracts == nil {
		contracts = []calendar.Contract{}
	}

	resp := map[string]any{
		"contracts":   contracts,
		"count":       len(contracts),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCheckRenewals(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	daysAhead := int(req.GetInt("days_ahead", 0))
	if daysAhead <= 0 {
		return mcp.NewToolResultError("days_ahead must be a positive integer"), nil
	}

	ns.logger.Debug("check_renewals called", "days_ahead", daysAhead)

	checks, err := ns.calendarEng.CheckRenewals(ctx, daysAhead)
	if err != nil {
		ns.logger.Warn("check_renewals failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check renewals: %s", err.Error())), nil
	}

	if checks == nil {
		checks = []calendar.RenewalCheck{}
	}

	resp := map[string]any{
		"renewals":    checks,
		"count":       len(checks),
		"days_ahead":  daysAhead,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleTriggerRenewal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	contractID, _ := req.RequireString("contract_id")

	ns.logger.Debug("trigger_renewal called", "contract_id", contractID)

	session, err := ns.calendarEng.TriggerNegotiation(ctx, contractID)
	if err != nil {
		ns.logger.Warn("trigger_renewal failed", "contract_id", contractID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger renewal: %s", err.Error())), nil
	}

	resp := map[string]any{
		"session_id":    session.ID,
		"status":        session.Status,
		"outcome":       session.Outcome,
		"vendor":        session.Vendor,
		"current_offer": session.CurrentOffer,
		"list_price":    session.ListPrice,
		"rounds":        session.RoundsComplete,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
