package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/a2a"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contract"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/gamification"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/group"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/health"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/marketplace"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/miner"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/parallel"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/quote"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sell"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sla"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/slack"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/webhooks"
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
	learningEng    *learning.Engine
	healthEng      *health.Engine
        marketplaceEng *marketplace.Engine
	slaEng       *sla.Engine
	webhookEng   *webhooks.Engine
	slackClient  *slack.Client
        apiKeyStore  *a2a.APIKeyStore
        quoteEng     *quote.Engine
        contractEng  *contract.Engine
        gamificationEng *gamification.Engine
}

// NewNegotiationServer creates a new MCP negotiation server.
func NewNegotiationServer(pricingStore *pricing.Store, historyStore *history.Store, groupEngine *group.Engine, sellEngine *sell.Engine, calendarEngine *calendar.Engine, healthEngine *health.Engine, marketplaceEngine *marketplace.Engine, slaEngine *sla.Engine, webhookEng *webhooks.Engine, slackClient *slack.Client, apiKeyStore *a2a.APIKeyStore, logger *slog.Logger) *NegotiationServer {
	eng := negotiation.NewEngine(pricingStore)
	miningEng := miner.NewEngine(pricingStore, logger)
	learningEng, err := learning.NewEngine(historyStore, logger)
	if err != nil {
		logger.Error("failed to create learning engine", "error", err)
		learningEng = nil
	} else {
		eng.SetLearningEngine(learningEng)
	}

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
		learningEng:    learningEng,
                marketplaceEng: marketplaceEngine,
		slackClient:    slackClient,
		apiKeyStore:    apiKeyStore,
                quoteEng:       quote.NewEngine(pricingStore, logger),
		contractEng:    contract.NewEngine(calendarEngine, logger),
                healthEng:      healthEngine,
                slaEng:         slaEngine,
                webhookEng:     webhookEng,
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
                mcp.WithString("user_id", mcp.Description("User identifier for gamification tracking (optional)")),
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

	// Tool 16: negotiate_strategy_recommend
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_strategy_recommend",
		mcp.WithDescription("Get the best negotiation strategy recommendation for a vendor based on past outcomes."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name to get recommendation for")),
	), ns.handleStrategyRecommend)

	// Tool 17: negotiate_learning_insights
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_learning_insights",
		mcp.WithDescription("Get global learning insights across all vendors — strategy performance breakdown and top vendors by deal count."),
	), ns.handleLearningInsights)

        // Tool 18: negotiate_failure_autopsy
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_failure_autopsy",
                mcp.WithDescription("Get a detailed autopsy of why a negotiation failed — failure reason, final offer, vendor best offer, and gap."),
                mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID to autopsy")),
        ), ns.handleFailureAutopsy)

        // Tool 19: negotiate_failure_patterns
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_failure_patterns",
                mcp.WithDescription("Analyze failure patterns for a specific vendor — shows recurring failure reasons and suggested fixes."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name to analyze")),
        ), ns.handleFailurePatterns)

        // Tool 20: negotiate_common_failures
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_common_failures",
                mcp.WithDescription("Get the most common failure patterns across all vendors, ranked by frequency."),
                mcp.WithInteger("limit", mcp.Description("Max number of patterns to return (default 10)")),
        ), ns.handleCommonFailures)

        // Tool 21: negotiate_list_unused_seats
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_unused_seats",
                mcp.WithDescription("List unused SaaS seats for sale on the marketplace. Ask price must be below original list price."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
                mcp.WithString("sku", mcp.Required(), mcp.Description("Product SKU")),
                mcp.WithInteger("seats", mcp.Required(), mcp.Description("Number of unused seats available")),
                mcp.WithNumber("orig_price", mcp.Required(), mcp.Description("Original per-seat price")),
                mcp.WithNumber("ask_price", mcp.Required(), mcp.Description("Asking price per seat")),
                mcp.WithNumber("min_price", mcp.Description("Minimum (walk-away) price per seat")),
                mcp.WithInteger("expires_in_hours", mcp.Description("Listing duration in hours (default: 168)")),
        ), ns.handleListUnusedSeats)

        // Tool 22: negotiate_search_used
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_search_used",
                mcp.WithDescription("Search for unused SaaS seat listings for a vendor/SKU, sorted by ask price ascending."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name to search for")),
                mcp.WithString("sku", mcp.Description("SKU filter (optional)")),
                mcp.WithInteger("max_seats", mcp.Description("Maximum seats filter (optional)")),
        ), ns.handleSearchUsed)

        // Tool 23: negotiate_offer_seats
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_offer_seats",
                mcp.WithDescription("Place a buy offer on a used-seats listing. Auto-accepts if ask price is within your max price."),
                mcp.WithString("listing_id", mcp.Required(), mcp.Description("Listing ID to make an offer on")),
                mcp.WithString("buyer_id", mcp.Required(), mcp.Description("Buyer identifier")),
                mcp.WithInteger("seats", mcp.Required(), mcp.Description("Number of seats to buy")),
                mcp.WithNumber("max_price", mcp.Required(), mcp.Description("Maximum price per seat you are willing to pay")),
        ), ns.handleOfferSeats)

        // Tool 24: negotiate_accept_offer
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_accept_offer",
                mcp.WithDescription("Accept a pending offer on a listing. Creates a transaction with 5% platform fee and marks listing as completed."),
                mcp.WithString("listing_id", mcp.Required(), mcp.Description("Listing ID")),
                mcp.WithString("offer_id", mcp.Required(), mcp.Description("Offer ID to accept")),
        ), ns.handleAcceptOffer)

        // Tool 25: negotiate_marketplace_overview
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_marketplace_overview",
                mcp.WithDescription("Get marketplace overview: active listings count and recent transactions."),
        ), ns.handleMarketplaceOverview)

        // Tool 26: negotiate_configure_slack
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_configure_slack",
                mcp.WithDescription("Configure Slack webhook URL for negotiation alerts."),
                mcp.WithString("webhook_url", mcp.Required(), mcp.Description("Slack incoming webhook URL")),
        ), ns.handleConfigureSlack)

        // Tool 27: negotiate_slack_status
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_slack_status",
                mcp.WithDescription("Check if Slack integration is configured and when the last alert was sent."),
        ), ns.handleSlackStatus)

	// SLA Tools
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_add_sla",
		mcp.WithDescription("Register an SLA contract with a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithNumber("uptime_pct", mcp.Required(), mcp.Description("Guaranteed uptime percentage (e.g. 99.9)")),
		mcp.WithNumber("credit_pct", mcp.Required(), mcp.Description("Service credit percentage (e.g. 10)")),
		mcp.WithNumber("max_credit_pct", mcp.Required(), mcp.Description("Maximum credit percentage cap (e.g. 25)")),
		mcp.WithNumber("monthly_spend", mcp.Required(), mcp.Description("Monthly spend amount")),
	), ns.handleAddSLA)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_record_breach",
		mcp.WithDescription("Record an SLA breach for a vendor service."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithString("date", mcp.Required(), mcp.Description("Breach date (RFC3339)")),
		mcp.WithInteger("duration_mins", mcp.Required(), mcp.Description("Downtime duration in minutes")),
	), ns.handleRecordBreach)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_file_claim",
		mcp.WithDescription("File an SLA breach claim for credit."),
		mcp.WithString("breach_id", mcp.Required(), mcp.Description("Breach ID to file")),
	), ns.handleFileClaim)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_sla_report",
		mcp.WithDescription("Get SLA report for a given month with all contracts, breaches, and credits."),
		mcp.WithString("month", mcp.Required(), mcp.Description("Month (RFC3339 date, e.g. 2026-06-01T00:00:00Z)")),
        ), ns.handleSLAReport)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_analyze_quote",
                mcp.WithDescription("Analyze a vendor quote email. Extracts vendor, SKU, quantity, price, and term from raw text, then cross-references against pricing database for market range and counter-offer recommendation."),
                mcp.WithString("raw_text", mcp.Required(), mcp.Description("The full text of the vendor quote email")),
                mcp.WithString("vendor", mcp.Description("Vendor name override (optional — extracted from text if omitted)")),
                mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
        ), ns.handleAnalyzeQuote)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_generate_counter",
                mcp.WithDescription("Generate a formatted counter-offer text from a quote analysis JSON."),
                mcp.WithString("analysis_json", mcp.Required(), mcp.Description("The full QuoteAnalysis JSON from negotiate_analyze_quote")),
        ), ns.handleGenerateCounter)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_parse_contract",
                mcp.WithDescription("Parse contract text to extract key terms: end dates, auto-renewal, price lock periods, termination notice. Returns structured terms with per-field confidence levels."),
                mcp.WithString("raw_text", mcp.Required(), mcp.Description("The full contract text to parse")),
                mcp.WithString("vendor", mcp.Description("Vendor name (optional)")),
                mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
        ), ns.handleParseContract)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_parse_and_calendar",
                mcp.WithDescription("Parse contract text AND auto-populate the renewal calendar. Combines parsing with calendar entry creation."),
                mcp.WithString("raw_text", mcp.Required(), mcp.Description("The full contract text to parse")),
                mcp.WithString("vendor", mcp.Description("Vendor name (optional)")),
                mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
        ), ns.handleParseAndCalendar)



	// Auth & Rate-limit Tools
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_generate_api_key",
		mcp.WithDescription("Generate a new API key for a given owner. Returns the key exactly once."),
		mcp.WithString("owner", mcp.Required(), mcp.Description("Owner name (e.g. user, agent, service)")),
	), ns.handleGenerateAPIKey)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_rate_limit_status",
		mcp.WithDescription("Check current API key count and rate limit configuration."),
	), ns.handleRateLimitStatus)


	// Tool: negotiate_find_cheapest_model
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_find_cheapest_model",
		mcp.WithDescription("Find the cheapest AI model for a given task type. Returns models sorted by price per 1M input tokens, with optional budget and latency filters."),
		mcp.WithString("task_type", mcp.Required(), mcp.Description("Task type: chat, reasoning, vision, audio, code, or image_generation")),
		mcp.WithNumber("budget", mcp.Description("Maximum budget per 1M input tokens (optional)")),
		mcp.WithNumber("max_latency_ms", mcp.Description("Maximum latency in milliseconds (optional)")),
	), ns.handleFindCheapestModel)


        // Tool: negotiate_vendor_reputation
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_vendor_reputation",
                mcp.WithDescription("Get the negotiation reputation for a vendor — tracks how flexible or rigid they've been across past deals."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name (e.g. Slack, GitHub, Salesforce)")),
        ), ns.handleVendorReputation)

        // Tool: negotiate_rank_flexibility
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_rank_flexibility",
                mcp.WithDescription("Rank all vendors by flexibility (avg discount percentage descending). Higher = more flexible and discount-friendly."),
                mcp.WithInteger("limit", mcp.Description("Maximum number of vendors to return (optional, default all)")),
        ), ns.handleRankFlexibility)


        // Tool 4x: negotiate_get_streak
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_get_streak",
                mcp.WithDescription("Get gamification streak info for a user. Returns current streak, longest streak, total deals, and total savings."),
                mcp.WithString("user_id", mcp.Required(), mcp.Description("User identifier")),
        ), ns.handleGetStreak)

        // Tool 4y: negotiate_savings_leaderboard
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_savings_leaderboard",
                mcp.WithDescription("Get the savings leaderboard. Returns top users ranked by total savings."),
                mcp.WithInteger("limit", mcp.Description("Maximum number of entries to return (optional, default 10)")),
        ), ns.handleSavingsLeaderboard)

        // Tool 4z: negotiate_achievements
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_achievements",
                mcp.WithDescription("Get all gamification badges and their earned status for a user."),
                mcp.WithString("user_id", mcp.Required(), mcp.Description("User identifier")),
        ), ns.handleAchievements)

}
// ─── Tool Handlers ───

func (ns *NegotiationServer) handleGenerateAPIKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	owner, _ := req.RequireString("owner")

	if ns.apiKeyStore == nil {
		return mcp.NewToolResultError("API key store is not configured"), nil
	}

	key, err := ns.apiKeyStore.GenerateKey(owner)
	if err != nil {
		ns.logger.Warn("generate_api_key failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate API key: %s", err.Error())), nil
	}

	ns.logger.Info("api key generated", "owner", owner)
	resp := map[string]any{
		"api_key":    key,
		"owner":      owner,
		"note":       "This key will not be shown again. Store it securely.",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRateLimitStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	keyCount := 0
	rateConfig := "unlimited"
	if ns.apiKeyStore != nil {
		keyCount = ns.apiKeyStore.KeyCount()
	}

	resp := map[string]any{
		"api_key_count": keyCount,
		"rate_limit":    rateConfig,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

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
	culture := req.GetString("culture", "us")
	budget := req.GetFloat("budget", 0)

	rawConstraints, _ := req.GetArguments()["constraints"]
	constraints, _ := rawConstraints.(map[string]any)

	ns.logger.Debug("create_session called", "vendor", vendor, "sku", sku, "strategy", strategyName, "budget", budget)

	session, err := ns.negotiationEng.CreateSession(ctx, vendor, sku, strategyName, budget, constraints, culture)
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

	// Update vendor reputation based on this outcome.
	succeeded := session.Outcome == "accepted"
	discountPct := result.TotalDiscount
	if !succeeded {
		discountPct = 0
	}
	if err := ns.healthEng.UpdateReputation(ctx, session.Vendor, discountPct, succeeded); err != nil {

        // Record gamification data after successful negotiation
        if succeeded && ns.gamificationEng != nil {
                userID := req.GetString("user_id", "anonymous")
                savings := session.ListPrice - session.CurrentOffer
                if savings < 0 {
                        savings = 0
                }
                if err := ns.gamificationEng.RecordNegotiation(ctx, userID, savings); err != nil {
                        ns.logger.Error("failed to record gamification", "user_id", userID, "error", err.Error())
                } else {
                        streak, _ := ns.gamificationEng.GetStreak(ctx, userID)
                        awarded, bErr := ns.gamificationEng.CheckAndAwardBadges(ctx, userID, streak)
                        if bErr != nil {
                                ns.logger.Error("failed to check badges", "user_id", userID, "error", bErr.Error())
                        } else if len(awarded) > 0 {
                                var badgeIDs []string
                                for _, b := range awarded {
                                        badgeIDs = append(badgeIDs, b.ID)
                                }
                                ns.logger.Info("badges awarded", "user_id", userID, "badges", badgeIDs)
                        }
                }
        }

		ns.logger.Error("failed to update vendor reputation", "vendor", session.Vendor, "error", err.Error())
	}

	// Record failure autopsy for non-accepted outcomes
	if !succeeded && ns.learningEng != nil {
		failureReason := deriveFailureReason(session.Outcome, session.Budget, session.CurrentOffer, session.ListPrice)
		gap := session.ListPrice - session.CurrentOffer
		if gap < 0 {
			gap = 0
		}
		autopsy := learning.Autopsy{
			SessionID:     session.ID,
			Vendor:        session.Vendor,
			SKU:           session.SKU,
			Strategy:      session.Strategy,
			FailureReason: failureReason,
			FinalOffer:    session.CurrentOffer,
			VendorBest:    session.CurrentOffer,
			Gap:           gap,
			TacticUsed:    session.Strategy,
		}
		if err := ns.learningEng.RecordFailure(ctx, autopsy); err != nil {
			ns.logger.Error("failed to record failure autopsy", "session_id", session.ID, "error", err.Error())
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

func (ns *NegotiationServer) handleCulturalProfiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        profiles := negotiation.ListCulturalProfiles()

        resp := map[string]any{
                "profiles":    profiles,
                "count":       len(profiles),
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

// modelTaskTypes maps AI model SKUs to their compatible task types.
var modelTaskTypes = map[string][]string{
	"gpt-4o":              {"chat", "vision", "code"},
	"gpt-4o-mini":         {"chat", "code"},
	"gpt-4o-audio":        {"audio"},
	"o1":                  {"reasoning", "code"},
	"o1-mini":             {"reasoning", "code"},
	"dall-e-3":            {"image_generation"},
	"whisper-1":           {"audio"},
	"tts-1":               {"audio"},
	"claude-3.5-sonnet":   {"chat", "vision", "code"},
	"claude-3-opus":       {"chat", "reasoning", "code"},
	"claude-3-haiku":      {"chat", "code"},
	"gemini-2.5-flash":    {"chat", "vision", "code"},
	"gemini-2.5-pro":      {"chat", "reasoning", "vision", "code"},
	"gemini-2.0-flash":    {"chat", "vision", "code"},
	"deepseek-v4":         {"chat", "reasoning", "code"},
	"deepseek-r1":         {"reasoning", "code"},
	"deepseek-chat":       {"chat", "code"},
	"mistral-large":       {"chat", "code"},
	"mistral-small":       {"chat", "code"},
	"command-r-plus":      {"chat", "code"},
	"command-r":           {"chat", "code"},
}

// ─── Find Cheapest Model Handler ───

func (ns *NegotiationServer) handleFindCheapestModel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	taskType, _ := req.RequireString("task_type")
	budget := req.GetFloat("budget", 0)
	maxLatency := req.GetFloat("max_latency_ms", 0)

	ns.logger.Debug("find_cheapest_model called", "task_type", taskType, "budget", budget, "max_latency_ms", maxLatency)

	// Validate task_type
	validTasks := map[string]bool{"chat": true, "reasoning": true, "vision": true, "audio": true, "code": true, "image_generation": true}
	if !validTasks[taskType] {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid task_type: %q. Valid values: chat, reasoning, vision, audio, code, image_generation", taskType)), nil
	}

	// Query all AI models from the pricing store
	pricingResults, err := ns.pricingStore.ListPricingByCategory(ctx, "ai")
	if err != nil {
		ns.logger.Warn("find_cheapest_model failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query AI models: %s", err.Error())), nil
	}

	// Build ModelResult list, filter by task type and budget
	var models []pricing.ModelResult
	for _, pr := range pricingResults {
		taskTypes, ok := modelTaskTypes[pr.SKU]
		if !ok {
			continue
		}

		// Filter by task_type
		matchesTask := false
		for _, tt := range taskTypes {
			if tt == taskType {
				matchesTask = true
				break
			}
		}
		if !matchesTask {
			continue
		}

		// Filter by budget (compare against min_observed which is MarketMin)
		if budget > 0 && pr.MarketMin > budget {
			continue
		}

		models = append(models, pricing.ModelResult{
			Vendor:       pr.Vendor,
			SKU:          pr.SKU,
			Description:  pr.Description,
			PricePerUnit: pr.MarketMin,
			Unit:         "/1M_input_tokens",
			TaskTypes:    taskTypes,
		})
	}

	// Sort by price ascending (bubble sort for simplicity)
	for i := 0; i < len(models); i++ {
		for j := i + 1; j < len(models); j++ {
			if models[j].PricePerUnit < models[i].PricePerUnit {
				models[i], models[j] = models[j], models[i]
			}
		}
	}

	if models == nil {
		models = []pricing.ModelResult{}
	}

	resp := map[string]any{
		"task_type":   taskType,
		"models":      models,
		"count":       len(models),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}


// ─── Vendor Reputation Handlers ───

func (ns *NegotiationServer) handleVendorReputation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("vendor_reputation called", "vendor", vendor)

	rep, err := ns.healthEng.GetReputation(ctx, vendor)
	if err != nil {
		ns.logger.Warn("vendor_reputation failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Reputation lookup failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":              rep.Vendor,
		"deal_count":          rep.DealCount,
		"avg_discount_pct":    rep.AvgDiscountPct,
		"max_discount_pct":    rep.MaxDiscountPct,
		"negotiability":       rep.Negotiability,
		"win_rate":            rep.WinRate,
		"duration_ms":         time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRankFlexibility(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	limit := int(req.GetInt("limit", 0))

	ns.logger.Debug("rank_flexibility called", "limit", limit)

	rankings, err := ns.healthEng.RankFlexibility(ctx, limit)
	if err != nil {
		ns.logger.Warn("rank_flexibility failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Rank flexibility failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"rankings":    rankings,
		"count":       len(rankings),
		"duration_ms": time.Since(start).Milliseconds(),
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

		session, err := ns.negotiationEng.CreateSession(ctx, vendor, sku, strategy, 0, nil, "")
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

	// Send Slack alerts for urgent renewals (<30 days)
	if ns.slackClient != nil && ns.slackClient.Enabled() {
		for _, c := range checks {
			if c.Urgency == "high" {
				blocks := slack.RenewalAlert(c.Contract, c.SuggestedSavings, c.DaysUntil)
				if err := ns.slackClient.Send(ctx, blocks); err != nil {
					ns.logger.Warn("failed to send Slack renewal alert", "vendor", c.Contract.Vendor, "error", err.Error())
				}
			}
		}
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

// ─── Learning / Strategy Recommendation Handlers ───

func (ns *NegotiationServer) handleStrategyRecommend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("strategy_recommend called", "vendor", vendor)

	if ns.learningEng == nil {
		return mcp.NewToolResultError("Learning engine is not available"), nil
	}

	rec, err := ns.learningEng.GetRecommendation(ctx, vendor)
	if err != nil {
		ns.logger.Warn("strategy_recommend failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Recommendation failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":                rec.Vendor,
		"recommended_strategy":  rec.RecommendedStrategy,
		"confidence":            rec.Confidence,
		"avg_discount_pct":      rec.AvgDiscount,
		"total_deals":           rec.TotalDeals,
		"breakdown":             rec.Breakdown,
		"duration_ms":           time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleLearningInsights(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("learning_insights called")

	if ns.learningEng == nil {
		return mcp.NewToolResultError("Learning engine is not available"), nil
	}

	insights, err := ns.learningEng.GetGlobalInsights(ctx)
	if err != nil {
		ns.logger.Warn("learning_insights failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Insights query failed: %s", err.Error())), nil
	}

	insights["duration_ms"] = time.Since(start).Milliseconds()
	return ns.jsonResult(insights)
}

// ─── Failure Autopsy Handlers ───

func (ns *NegotiationServer) handleFailureAutopsy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        sessionID, _ := req.RequireString("session_id")

        ns.logger.Debug("failure_autopsy called", "session_id", sessionID)

        if ns.learningEng == nil {
                return mcp.NewToolResultError("Learning engine is not available"), nil
        }

        // Load session from history store to build autopsy
        sessRec, err := ns.historyStore.GetSession(ctx, sessionID)
        if err != nil {
                return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", err.Error())), nil
        }

        // Derive failure reason from session outcome
        failureReason := deriveFailureReason(sessRec.Outcome, sessRec.Budget, sessRec.CurrentOffer, sessRec.ListPrice)
        gap := sessRec.ListPrice - sessRec.CurrentOffer
        if gap < 0 {
                gap = 0
        }

        autopsy := learning.Autopsy{
                SessionID:     sessionID,
                Vendor:        sessRec.Vendor,
                SKU:           sessRec.SKU,
                Strategy:      sessRec.Strategy,
                FailureReason: failureReason,
                FinalOffer:    sessRec.CurrentOffer,
                VendorBest:    sessRec.CurrentOffer,
                Gap:           gap,
                TacticUsed:    sessRec.Strategy,
        }

        resp := map[string]any{
                "session_id":     autopsy.SessionID,
                "vendor":         autopsy.Vendor,
                "sku":            autopsy.SKU,
                "strategy":       autopsy.Strategy,
                "failure_reason": autopsy.FailureReason,
                "final_offer":    autopsy.FinalOffer,
                "vendor_best":    autopsy.VendorBest,
                "gap":            autopsy.Gap,
                "tactic_used":    autopsy.TacticUsed,
                "duration_ms":    time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleFailurePatterns(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        vendor, _ := req.RequireString("vendor")

        ns.logger.Debug("failure_patterns called", "vendor", vendor)

        if ns.learningEng == nil {
                return mcp.NewToolResultError("Learning engine is not available"), nil
        }

        patterns, err := ns.learningEng.AnalyzeFailures(ctx, vendor)
        if err != nil {
                ns.logger.Warn("failure_patterns failed", "vendor", vendor, "error", err.Error())
                return mcp.NewToolResultError(fmt.Sprintf("Failure analysis failed: %s", err.Error())), nil
        }

        resp := map[string]any{
                "vendor":   vendor,
                "patterns": patterns,
                "count":    len(patterns),
                "duration_ms": time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCommonFailures(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        limit := int(req.GetInt("limit", 10))

        ns.logger.Debug("common_failures called", "limit", limit)

        if ns.learningEng == nil {
                return mcp.NewToolResultError("Learning engine is not available"), nil
        }

        patterns, err := ns.learningEng.CommonFailureModes(ctx, limit)
        if err != nil {
                ns.logger.Warn("common_failures failed", "error", err.Error())
                return mcp.NewToolResultError(fmt.Sprintf("Common failures query failed: %s", err.Error())), nil
        }

        resp := map[string]any{
                "patterns":   patterns,
                "count":      len(patterns),
                "limit":      limit,
                "duration_ms": time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

// deriveFailureReason maps negotiation outcome + session state to a failure reason string.
func deriveFailureReason(outcome string, budget float64, currentOffer float64, listPrice float64) string {
        switch outcome {
        case "walked_away":
                return "vendor_refused"
        case "rejected":
                if budget > 0 && currentOffer > budget {
                        return "budget_exceeded"
                }
                return "price_too_high"
        case "timeout":
                return "timeout"
        default:
                return "counter_too_low"
        }
}

// ─── Marketplace Handlers ───

func (ns *NegotiationServer) handleListUnusedSeats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	seats := int(req.GetInt("seats", 0))
	origPrice := req.GetFloat("orig_price", 0)
	askPrice := req.GetFloat("ask_price", 0)
	minPrice := req.GetFloat("min_price", 0)
	expiresInHours := int(req.GetInt("expires_in_hours", 168))

	ns.logger.Debug("list_unused_seats called", "vendor", vendor, "sku", sku, "seats", seats)

	listing, err := ns.marketplaceEng.ListSeats(ctx, vendor, sku, seats, origPrice, askPrice, minPrice, expiresInHours)
	if err != nil {
		ns.logger.Warn("list_unused_seats failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("List seats failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"listing_id":  listing.ID,
		"vendor":      listing.Vendor,
		"sku":         listing.SKU,
		"seats":       listing.Seats,
		"orig_price":  listing.OrigPrice,
		"ask_price":   listing.AskPrice,
		"min_price":   listing.MinPrice,
		"status":      listing.Status,
		"created_at":  listing.CreatedAt.Format(time.RFC3339),
		"expires_at":  listing.ExpiresAt.Format(time.RFC3339),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSearchUsed(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku := req.GetString("sku", "")
	maxSeats := int(req.GetInt("max_seats", 0))

	ns.logger.Debug("search_used called", "vendor", vendor, "sku", sku, "max_seats", maxSeats)

	listings, err := ns.marketplaceEng.Search(ctx, vendor, sku, maxSeats)
	if err != nil {
		ns.logger.Warn("search_used failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %s", err.Error())), nil
	}

	if listings == nil {
		listings = []marketplace.Listing{}
	}

	resp := map[string]any{
		"listings":    listings,
		"count":       len(listings),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleOfferSeats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")
	buyerID, _ := req.RequireString("buyer_id")
	seats := int(req.GetInt("seats", 0))
	maxPrice := req.GetFloat("max_price", 0)

	ns.logger.Debug("offer_seats called", "listing_id", listingID, "buyer_id", buyerID, "seats", seats, "max_price", maxPrice)

	offer, err := ns.marketplaceEng.MakeOffer(ctx, listingID, buyerID, seats, maxPrice)
	if err != nil {
		ns.logger.Warn("offer_seats failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Offer failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"offer_id":    offer.ID,
		"listing_id":  offer.ListingID,
		"buyer_id":    offer.BuyerID,
		"seats":       offer.Seats,
		"max_price":   offer.MaxPrice,
		"status":      offer.Status,
		"created_at":  offer.CreatedAt.Format(time.RFC3339),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAcceptOffer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	listingID, _ := req.RequireString("listing_id")
	offerID, _ := req.RequireString("offer_id")

	ns.logger.Debug("accept_offer called", "listing_id", listingID, "offer_id", offerID)

	txn, err := ns.marketplaceEng.AcceptOffer(ctx, listingID, offerID)
	if err != nil {
		ns.logger.Warn("accept_offer failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Accept offer failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"transaction_id": txn.ID,
		"listing_id":     txn.ListingID,
		"vendor":         txn.Vendor,
		"sku":            txn.SKU,
		"seats":          txn.Seats,
		"price_per_seat": txn.PricePerSeat,
		"total":          txn.Total,
		"platform_fee":   txn.PlatformFee,
		"seller_id":      txn.SellerID,
		"buyer_id":       txn.BuyerID,
		"status":         txn.Status,
		"created_at":     txn.CreatedAt.Format(time.RFC3339),
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleMarketplaceOverview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("marketplace_overview called")

	overview, err := ns.marketplaceEng.Overview(ctx)
	if err != nil {
		ns.logger.Warn("marketplace_overview failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Overview failed: %s", err.Error())), nil
	}

	overview["duration_ms"] = time.Since(start).Milliseconds()
	return ns.jsonResult(overview)
}

// ─── Slack Integration Handlers ───

func (ns *NegotiationServer) handleConfigureSlack(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	webhookURL, _ := req.RequireString("webhook_url")

	ns.logger.Debug("configure_slack called", "configured", webhookURL != "")

	ns.slackClient = slack.NewClient(webhookURL, ns.logger)

	resp := map[string]any{
		"status":  "configured",
		"enabled": ns.slackClient.Enabled(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSlackStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns.logger.Debug("slack_status called")

	configured := ns.slackClient != nil && ns.slackClient.Enabled()
	lastSent := ""
	if ns.slackClient != nil {
		ts := ns.slackClient.LastSent()
		if !ts.IsZero() {
			lastSent = ts.Format(time.RFC3339)
		}
	}

	resp := map[string]any{
		"configured":      configured,
		"last_alert_sent": lastSent,
	}
	return ns.jsonResult(resp)
}

// ─── Vendor Health Handlers ───

func (ns *NegotiationServer) handleVendorHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("vendor_health called", "vendor", vendor)

	leverage, err := ns.healthEng.GetLeverage(ctx, vendor)
	if err != nil {
		ns.logger.Warn("vendor_health failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Health lookup failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":     leverage.Vendor,
		"health": map[string]any{
			"score":        leverage.Health.Score,
			"category":     leverage.Health.Category,
			"last_updated": leverage.Health.LastUpdated.Format(time.RFC3339),
			"signals":      leverage.Health.Signals,
		},
		"leverage":   leverage.Leverage,
		"suggestion": leverage.Suggestion,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRecordSignal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	signalType, _ := req.RequireString("type")
	detail, _ := req.RequireString("detail")
	weight := int(req.GetInt("weight", 0))

	ns.logger.Debug("record_signal called", "vendor", vendor, "type", signalType)

	if err := ns.healthEng.RecordSignal(ctx, vendor, signalType, "manual", detail, weight); err != nil {
		ns.logger.Warn("record_signal failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Record signal failed: %s", err.Error())), nil
	}

	// Re-fetch leverage for the response
	leverage, err := ns.healthEng.GetLeverage(ctx, vendor)
	if err != nil {
		ns.logger.Warn("record_signal: leverage re-fetch failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Signal recorded but re-fetch failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":     leverage.Vendor,
		"recorded":   true,
		"health": map[string]any{
			"score":    leverage.Health.Score,
			"category": leverage.Health.Category,
		},
		"leverage":   leverage.Leverage,
		"suggestion": leverage.Suggestion,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleHealthOverview(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("health_overview called")

	vendors, err := ns.healthEng.Store().ListAll(ctx)
	if err != nil {
		ns.logger.Warn("health_overview failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Health overview failed: %s", err.Error())), nil
	}

	if vendors == nil {
		vendors = []health.VendorHealth{}
	}

	resp := map[string]any{
		"vendors":     vendors,
		"vendor_count": len(vendors),
		"duration_ms":  time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}


// ─── SLA Handlers ───

func (ns *NegotiationServer) handleAddSLA(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	service, _ := req.RequireString("service")
	uptimePct := req.GetFloat("uptime_pct", 0)
	creditPct := req.GetFloat("credit_pct", 0)
	maxCreditPct := req.GetFloat("max_credit_pct", 0)
	monthlySpend := req.GetFloat("monthly_spend", 0)

	ns.logger.Debug("add_sla called", "vendor", vendor, "service", service)

	contract, err := ns.slaEng.AddContract(ctx, vendor, service, uptimePct, creditPct, maxCreditPct, monthlySpend)
	if err != nil {
		ns.logger.Warn("add_sla failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add SLA contract: %s", err.Error())), nil
	}

	resp := map[string]any{
		"sla_id":          contract.ID,
		"vendor":          contract.Vendor,
		"service":         contract.Service,
		"uptime_pct":      contract.UptimePct,
		"credit_pct":      contract.CreditPct,
		"max_credit_pct":  contract.MaxCreditPct,
		"monthly_spend":   contract.MonthlySpend,
		"status":          contract.Status,
		"duration_ms":     time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRecordBreach(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	service, _ := req.RequireString("service")
	dateStr, _ := req.RequireString("date")
	durationMins := int(req.GetInt("duration_mins", 0))

	ns.logger.Debug("record_breach called", "vendor", vendor, "service", service, "duration_mins", durationMins)

	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid date: must be RFC3339 format (e.g. 2026-06-01T00:00:00Z)")), nil
	}

	breach, err := ns.slaEng.RecordBreach(ctx, vendor, service, date, durationMins)
	if err != nil {
		ns.logger.Warn("record_breach failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to record breach: %s", err.Error())), nil
	}

	resp := map[string]any{
		"breach_id":     breach.ID,
		"vendor":        breach.Vendor,
		"service":       breach.Service,
		"date":          breach.Date.Format(time.RFC3339),
		"duration_mins": breach.DurationMins,
		"credit_due":    breach.CreditDue,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleFileClaim(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	breachID, _ := req.RequireString("breach_id")

	ns.logger.Debug("file_claim called", "breach_id", breachID)

	breach, err := ns.slaEng.FileClaim(ctx, breachID)
	if err != nil {
		ns.logger.Warn("file_claim failed", "breach_id", breachID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to file claim: %s", err.Error())), nil
	}

	resp := map[string]any{
		"breach_id":   breach.ID,
		"vendor":      breach.Vendor,
		"service":     breach.Service,
		"credit_due":  breach.CreditDue,
		"filed":       breach.Filed,
		"filed_at":    breach.FiledAt.Format(time.RFC3339),
		"payout":      breach.Payout,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSLAReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	monthStr, _ := req.RequireString("month")

	ns.logger.Debug("sla_report called", "month", monthStr)

	month, err := time.Parse(time.RFC3339, monthStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid month: must be RFC3339 format (e.g. 2026-06-01T00:00:00Z)")), nil
	}

	report, err := ns.slaEng.GetReport(ctx, month)
	if err != nil {
		ns.logger.Warn("sla_report failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get SLA report: %s", err.Error())), nil
	}

	resp := map[string]any{
		"contract":      report.Contract,
		"breaches":      report.Breaches,
		"total_credits": report.TotalCredits,
		"filed_count":   report.FiledCount,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAnalyzeQuote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	rawText, _ := req.RequireString("raw_text")
	vendor := req.GetString("vendor", "")
	sku := req.GetString("sku", "")

	ns.logger.Debug("analyze_quote called", "vendor", vendor, "sku", sku, "text_length", len(rawText))

	input := quote.QuoteInput{
		RawText: rawText,
		Vendor:  vendor,
		SKU:     sku,
	}

	analysis, err := ns.quoteEng.AnalyzeQuote(ctx, input)
	if err != nil {
		ns.logger.Warn("analyze_quote failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Quote analysis failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"quote": map[string]any{
			"vendor":          analysis.Quote.Vendor,
			"sku":             analysis.Quote.SKU,
			"description":     analysis.Quote.Description,
			"seats":           analysis.Quote.Seats,
			"term_months":     analysis.Quote.TermMonths,
			"price_per_unit":  analysis.Quote.PricePerUnit,
			"total_price":     analysis.Quote.TotalPrice,
			"list_price":      analysis.Quote.ListPrice,
			"discount_offered": analysis.Quote.DiscountOffered,
		},
		"market_range":       analysis.MarketRange,
		"counter_offer_min":  analysis.CounterOfferMin,
		"counter_offer_max":  analysis.CounterOfferMax,
		"potential_savings":  analysis.PotentialSavings,
		"confidence":         analysis.Confidence,
		"duration_ms":        time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGenerateCounter(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	analysisJSON, _ := req.RequireString("analysis_json")

	ns.logger.Debug("generate_counter called", "analysis_length", len(analysisJSON))

	var analysis quote.QuoteAnalysis
	if err := json.Unmarshal([]byte(analysisJSON), &analysis); err != nil {
		ns.logger.Warn("generate_counter failed to parse analysis_json", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Invalid analysis_json: %s", err.Error())), nil
	}

	counterText, err := ns.quoteEng.GenerateCounterOffer(ctx, &analysis)
	if err != nil {
		ns.logger.Warn("generate_counter failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Counter-offer generation failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"counter_offer": counterText,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleParseContract(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        rawText, _ := req.RequireString("raw_text")
        vendor := req.GetString("vendor", "")
        sku := req.GetString("sku", "")

        ns.logger.Debug("parse_contract called", "vendor", vendor, "sku", sku, "text_length", len(rawText))

        result, err := ns.contractEng.ParseContract(ctx, rawText, vendor, sku)
        if err != nil {
                ns.logger.Warn("parse_contract failed", "error", err.Error())
                return mcp.NewToolResultError(fmt.Sprintf("Contract parse failed: %s", err.Error())), nil
        }

        resp := map[string]any{
                "terms":            result.Terms,
                "field_confidence": result.FieldConf,
                "warnings":         result.Warnings,
                "auto_populated":   result.AutoPopulated,
                "duration_ms":      time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleParseAndCalendar(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()
        rawText, _ := req.RequireString("raw_text")
        vendor := req.GetString("vendor", "")
        sku := req.GetString("sku", "")

        ns.logger.Debug("parse_and_calendar called", "vendor", vendor, "sku", sku, "text_length", len(rawText))

        result, err := ns.contractEng.ParseContract(ctx, rawText, vendor, sku)
        if err != nil {
                ns.logger.Warn("parse_and_calendar failed at parse", "error", err.Error())
                return mcp.NewToolResultError(fmt.Sprintf("Contract parse failed: %s", err.Error())), nil
        }

        if result.Terms.EndDate != "" {
                if err := ns.contractEng.PopulateCalendar(ctx, result); err != nil {
                        ns.logger.Warn("parse_and_calendar failed at calendar population", "error", err.Error())
                        // Return partial result with warning
                        resp := map[string]any{
                                "terms":            result.Terms,
                                "field_confidence": result.FieldConf,
                                "warnings":         append(result.Warnings, "calendar population failed: "+err.Error()),
                                "auto_populated":   false,
                                "duration_ms":      time.Since(start).Milliseconds(),
                        }
                        return ns.jsonResult(resp)
                }
        }

        resp := map[string]any{
                "terms":            result.Terms,
                "field_confidence": result.FieldConf,
                "warnings":         result.Warnings,
                "auto_populated":   result.AutoPopulated,
                "duration_ms":      time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}


// ─── Gamification ───

// SetGamificationEngine attaches the gamification engine for streak and badge tracking.
func (ns *NegotiationServer) SetGamificationEngine(eng *gamification.Engine) {
	ns.gamificationEng = eng
}

func (ns *NegotiationServer) handleGetStreak(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	userID, _ := req.RequireString("user_id")

	ns.logger.Debug("get_streak called", "user_id", userID)

	streak, err := ns.gamificationEng.GetStreak(ctx, userID)
	if err != nil {
		ns.logger.Warn("get_streak failed", "user_id", userID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Get streak failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"user_id":             streak.UserID,
		"current_streak":      streak.CurrentStreak,
		"longest_streak":      streak.LongestStreak,
		"last_negotiation_at": streak.LastNegotiationAt.Format(time.RFC3339),
		"total_savings":       streak.TotalSavings,
		"total_deals":         streak.TotalDeals,
		"duration_ms":         time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSavingsLeaderboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	limit := int(req.GetInt("limit", 10))

	ns.logger.Debug("savings_leaderboard called", "limit", limit)

	entries, err := ns.gamificationEng.GetLeaderboard(ctx, limit)
	if err != nil {
		ns.logger.Warn("savings_leaderboard failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Leaderboard query failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"leaderboard": entries,
		"count":       len(entries),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAchievements(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	userID, _ := req.RequireString("user_id")

	ns.logger.Debug("achievements called", "user_id", userID)

	badges, err := ns.gamificationEng.GetBadges(ctx, userID)
	if err != nil {
		ns.logger.Warn("achievements failed", "user_id", userID, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Get achievements failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"user_id":    userID,
		"badges":     badges,
		"count":      len(badges),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
