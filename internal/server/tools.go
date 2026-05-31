package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/a2a"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/aiperformance"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/alerthistory"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/apidocs"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/apikeyrotation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/approvals"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/auditlog"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/autocomplete"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/batchnegotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/benchmark"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budget"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budgetalerts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budgetmgmt"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/commlog"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/compliance"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contract"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contractclauses"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contractrisk"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contracttemplates"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contribguide"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/costallocation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/coverage"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dataimport"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/datresidency"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dataretention"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dependency"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/effectiveness"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/esignature"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/chartexport"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/monitordash"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/export"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/gamification"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/group"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/health"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/healthcheck"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/industryreports"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/webhooklog"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/vendorknowledge"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ipwhitelist"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/limitedoffer"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/marketplace"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/metrics"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/miner"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/modelabtesting"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/notes"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/notify"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/parallel"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/playbook"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricealerts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricechart"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricingindex"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricingrefresh"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/prompts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/quote"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ratelimitdashboard"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/reminders"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/reports"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/roi"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/savingsrealization"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/scorecards"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sell"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sentiment"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sharedstrategies"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/shutdown"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/summarizer"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sla"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/slack"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/slacredit"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/spendingcaps"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/strategycomparison"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/tco"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/toolbilling"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/toolstats"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/translation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/training"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/trends"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/useractivity"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/vendorcomparison"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/vendorspend"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/webhooks"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/winloss"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/workspaces"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dashboard"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NegotiationServer wraps the MCP server with business logic.
type NegotiationServer struct {
	mcpServer             *mcpserver.MCPServer
	pricingStore          *pricing.Store
	negotiationEng        *negotiation.Engine
	historyStore          *history.Store
	minerEng              *miner.Engine
	groupEng              *group.Engine
	sellEng               *sell.Engine
	sentimentEng          *sentiment.Engine
	calendarEng           *calendar.Engine
	playbookEng           *playbook.Engine
        chartExportEng       *chartexport.Engine
	monitorDashEng      *monitordash.Engine
	trainingEng           *training.Engine
	logger                *slog.Logger
	learningEng           *learning.Engine
	healthEng             *health.Engine
	marketplaceEng        *marketplace.Engine
	slaEng                *sla.Engine
	webhookEng            *webhooks.Engine
	slackClient           *slack.Client
	apiKeyStore           *a2a.APIKeyStore
	quoteEng              *quote.Engine
	contractEng           *contract.Engine
	gamificationEng       *gamification.Engine
	roiEng                *roi.Engine
	benchmarkEng          *benchmark.Engine
	trendsEng             *trends.Engine
	winlossEng            *winloss.Engine
	exportEng             *export.Engine
	notifyEng             *notify.Engine
	budgetEng             *budget.Engine
	vendorspendEng        *vendorspend.Engine
	effectivenessEng      *effectiveness.Engine
	priceAlertEng         *pricealerts.Engine
	reminderEng           *reminders.Engine
	budgetAlertEng        *budgetalerts.Engine
	reportsEng            *reports.Engine
	pricingIndexEng       *pricingindex.Engine
	priceChartEng         *pricechart.Engine
	vendorComparisonEng   *vendorcomparison.Engine
	batchNegotiationEng   *batchnegotiation.Engine
	strategyComparisonEng *strategycomparison.Engine
	workspacesEng         *workspaces.Engine
	auditLogEng           *auditlog.Engine
	userActivityEng       *useractivity.Engine
	sharedStrategiesEng   *sharedstrategies.Engine
	notesEng              *notes.Engine
	approvalsEng          *approvals.Engine
	contractTemplatesEng  *contracttemplates.Engine
	contractRiskEng       *contractrisk.Engine
	scorecardsEng         *scorecards.Engine
	budgetMgmtEng         *budgetmgmt.Engine
	spendingCapsEng       *spendingcaps.Engine
	savingsRealizationEng *savingsrealization.Engine
	tcoEng                *tco.Engine
	alertHistoryEng       *alerthistory.Engine
	slaCreditEng          *slacredit.Engine
	commLogEng            *commlog.Engine
	dataImportEng         *dataimport.Engine
	costAllocationEng     *costallocation.Engine
	limitedOfferEng       *limitedoffer.Engine
	pricingRefreshEng     *pricingrefresh.Engine
	rateLimitDashEng      *ratelimitdashboard.Engine
	apiDocsEng            *apidocs.Engine
	toolStatsEng          *toolstats.Engine
	healthCheckEng        *healthcheck.Engine
	apiKeyRotateEng       *apikeyrotation.Engine
	ipWhitelistEng        *ipwhitelist.Engine
	dataRetentionEng      *dataretention.Engine
	autocompleteEng       *autocomplete.Engine
	metricsEng            *metrics.Engine
	shutdownEng           *shutdown.Engine
	coverageEng           *coverage.Engine
	dependencyEng         *dependency.Engine
	contribguideEng       *contribguide.Engine
	industryReportsStore  *industryreports.Store
	webhookLogStore       *webhooklog.Store
	aiPerfStore           *aiperformance.Store
	modelABTestEng        *modelabtesting.Engine
	promptsStore          *prompts.Store
	vendorKnowledgeStore *vendorknowledge.Store
	translationStore      *translation.Store
	translationEng        *translation.Engine
        summarizerEng *summarizer.Engine
	esigStore *esignature.Store
	esigEng *esignature.Engine
	clausesStore *contractclauses.Store
	complianceEng       *compliance.Engine
	residencyStore *datresidency.Store
	dashboardStore *dashboard.Store
	toolBillingStore     *toolbilling.Store
}

// NewNegotiationServer creates a new MCP negotiation server.
func NewNegotiationServer(pricingStore *pricing.Store, historyStore *history.Store, groupEngine *group.Engine, sellEngine *sell.Engine, calendarEngine *calendar.Engine, healthEngine *health.Engine, marketplaceEngine *marketplace.Engine, slaEngine *sla.Engine, webhookEng *webhooks.Engine, slackClient *slack.Client, apiKeyStore *a2a.APIKeyStore, roiStore *roi.Store, trendsStore *trends.Store, exportStore *export.Store, notifyStore *notify.Store, budgetStore *budget.Store, vendorspendEng *vendorspend.Engine, effectivenessEng *effectiveness.Engine, priceAlertStore *pricealerts.Store, budgetAlertStore *budgetalerts.Store, reportsEng *reports.Engine, pricingIndexEng *pricingindex.Engine, priceChartEng *pricechart.Engine, vendorComparisonEng *vendorcomparison.Engine, batchNegotiationEng *batchnegotiation.Engine, strategyComparisonEng *strategycomparison.Engine, workspacesEng *workspaces.Engine, auditLogEng *auditlog.Engine, userActivityEng *useractivity.Engine, contractTemplatesEng *contracttemplates.Engine, contractRiskEng *contractrisk.Engine, scorecardsEng *scorecards.Engine, sharedStrategiesEng *sharedstrategies.Engine, notesEng *notes.Engine, approvalsEng *approvals.Engine, budgetMgmtEng *budgetmgmt.Engine, spendingCapsEng *spendingcaps.Engine, savingsRealizationEng *savingsrealization.Engine, tcoEng *tco.Engine, dataImportEng *dataimport.Engine, costAllocationEng *costallocation.Engine, alertHistoryEng *alerthistory.Engine, slaCreditEng *slacredit.Engine, commLogEng *commlog.Engine, limitedOfferEng *limitedoffer.Engine, pricingRefreshEng *pricingrefresh.Engine, rateLimitDashEng *ratelimitdashboard.Engine, apiDocsEng *apidocs.Engine, toolStatsEng *toolstats.Engine, healthCheckEng *healthcheck.Engine, autocompleteEng *autocomplete.Engine, metricsEng *metrics.Engine, shutdownEng *shutdown.Engine, coverageEng *coverage.Engine, dependencyEng *dependency.Engine, contribguideEng *contribguide.Engine, apiKeyRotateEng *apikeyrotation.Engine, ipWhitelistEng *ipwhitelist.Engine, dataRetentionEng *dataretention.Engine, playbookEng *playbook.Engine, trainingEng *training.Engine, industryReportsStore *industryreports.Store, aiPerfStore *aiperformance.Store, promptsStore *prompts.Store, modelABTestEng *modelabtesting.Engine, vendorKnowledgeStore *vendorknowledge.Store, summarizerEng *summarizer.Engine, sentimentEng *sentiment.Engine, translationStore *translation.Store, translationEng *translation.Engine, complianceEng *compliance.Engine, clausesStore *contractclauses.Store, esigStore *esignature.Store, esigEng *esignature.Engine, residencyStore *datresidency.Store, dashboardStore *dashboard.Store, chartExportEng *chartexport.Engine, monitorDashEng *monitordash.Engine, webhookLogStore *webhooklog.Store, toolBillingStore *toolbilling.Store, logger *slog.Logger) *NegotiationServer {
	eng := negotiation.NewEngine(pricingStore)
	miningEng := miner.NewEngine(pricingStore, logger)
	learningEng, err := learning.NewEngine(historyStore, logger)
	if err != nil {
		logger.Error("failed to create learning engine", "error", err)
		learningEng = nil
	} else {
		eng.SetLearningEngine(learningEng)
	}

	budgetEng := budget.NewEngine(budgetStore, pricingStore.DB(), logger)

	priceAlertEng := pricealerts.NewEngine(priceAlertStore, func(ctx context.Context, vendor, sku string) (float64, error) {
		snap, err := trendsStore.GetLatest(ctx, vendor, sku)
		if err != nil {
			return 0, err
		}
		if snap == nil {
			return 0, nil
		}
		return snap.Price, nil
	}, logger)

	reminderEng := reminders.NewEngine(func(ctx context.Context, daysAhead int) ([]reminders.ContractRow, error) {
		contracts, err := calendarEngine.Store().GetContractsExpiringSoon(ctx, daysAhead)
		if err != nil {
			return nil, err
		}
		rows := make([]reminders.ContractRow, len(contracts))
		for i, c := range contracts {
			rows[i] = reminders.ContractRow{
				ID:          c.ID,
				Vendor:      c.Vendor,
				SKU:         c.SKU,
				RenewalDate: c.RenewalDate.Format(time.DateOnly),
			}
		}
		return rows, nil
	}, logger)

	budgetAlertEng := budgetalerts.NewEngine(budgetAlertStore, pricingStore.DB(), logger)

	ns := &NegotiationServer{
		mcpServer: mcpserver.NewMCPServer(
			"a2a-negotiation-mcp",
			"1.0.0",
			mcpserver.WithToolCapabilities(true),
			mcpserver.WithResourceCapabilities(true, true),
			mcpserver.WithLogging(),
		),
		pricingStore:          pricingStore,
		negotiationEng:        eng,
		historyStore:          historyStore,
		minerEng:              miningEng,
		groupEng:              groupEngine,
		sellEng:               sellEngine,
	sentimentEng:          sentimentEng,
		calendarEng:           calendarEngine,
        chartExportEng:       chartExportEng,
		monitorDashEng:      monitorDashEng,
		webhookLogStore:     webhookLogStore,
		logger:                logger,
		learningEng:           learningEng,
		marketplaceEng:        marketplaceEngine,
		slackClient:           slackClient,
		apiKeyStore:           apiKeyStore,
		quoteEng:              quote.NewEngine(pricingStore, logger),
		roiEng:                roi.NewEngine(roiStore),
		benchmarkEng:          benchmark.NewEngine(historyStore, logger),
		trendsEng:             trends.NewEngine(trendsStore, logger),
		winlossEng:            winloss.NewEngine(historyStore, logger),
		exportEng:             export.NewEngine(exportStore, historyStore, logger),
		notifyEng:             notify.NewEngine(notifyStore, logger),
		contractEng:           contract.NewEngine(calendarEngine, logger),
		healthEng:             healthEngine,
		slaEng:                slaEngine,
		webhookEng:            webhookEng,
		budgetEng:             budgetEng,
		vendorspendEng:        vendorspendEng,
		priceAlertEng:         priceAlertEng,
		reminderEng:           reminderEng,
		budgetAlertEng:        budgetAlertEng,
		reportsEng:            reportsEng,
		pricingIndexEng:       pricingIndexEng,
		priceChartEng:         priceChartEng,
		effectivenessEng:      effectivenessEng,
		contractTemplatesEng:  contractTemplatesEng,
		contractRiskEng:       contractRiskEng,
		scorecardsEng:         scorecardsEng,
		sharedStrategiesEng:   sharedStrategiesEng,
		notesEng:              notesEng,
		approvalsEng:          approvalsEng,
		budgetMgmtEng:         budgetMgmtEng,
		spendingCapsEng:       spendingCapsEng,
		savingsRealizationEng: savingsRealizationEng,
		alertHistoryEng:       alertHistoryEng,
		slaCreditEng:          slaCreditEng,
		commLogEng:            commLogEng,
		limitedOfferEng:       limitedOfferEng,
		pricingRefreshEng:     pricingRefreshEng,
		rateLimitDashEng:      rateLimitDashEng,
		apiDocsEng:            apiDocsEng,
		toolStatsEng:          toolStatsEng,
		healthCheckEng:        healthCheckEng,
		autocompleteEng:       autocompleteEng,
		metricsEng:            metricsEng,
		shutdownEng:           shutdownEng,
		coverageEng:           coverageEng,
		dependencyEng:         dependencyEng,
		contribguideEng:       contribguideEng,
		apiKeyRotateEng:       apiKeyRotateEng,
		ipWhitelistEng:        ipWhitelistEng,
		dataRetentionEng:      dataRetentionEng,
		playbookEng:           playbookEng,
		trainingEng:           trainingEng,
		industryReportsStore:  industryReportsStore,
		aiPerfStore:           aiPerfStore,
		modelABTestEng:        modelABTestEng,
		promptsStore:          promptsStore,
		vendorKnowledgeStore:  vendorKnowledgeStore,
		translationStore:     translationStore,
		translationEng:       translationEng,
                summarizerEng:  summarizerEng,
                clausesStore:         clausesStore,
                complianceEng:       complianceEng,
                residencyStore:       residencyStore,
                dashboardStore:       dashboardStore,
		toolBillingStore:     toolBillingStore,
	}

	ns.registerTools()
	ns.registerResources()
	return ns
}
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

	// Tool 5a: negotiate_calculate_roi
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_calculate_roi",
		mcp.WithDescription("Calculate ROI for a negotiated deal. Returns annual savings, ROI percentage, payback period, multi-year savings, and NPV at 8% discount rate."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithNumber("current_spend", mcp.Required(), mcp.Description("Current annual spend with this vendor")),
		mcp.WithNumber("negotiated_price", mcp.Required(), mcp.Description("Negotiated annual price")),
		mcp.WithNumber("implementation_costs", mcp.Description("One-time implementation/migration costs (optional, default 0)")),
		mcp.WithNumber("annual_overhead", mcp.Description("Annual management/overhead costs (optional, default 0)")),
	), ns.handleCalculateROI)

	// Tool 5b: negotiate_benchmark_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_benchmark_report",
		mcp.WithDescription("Generate a savings benchmark report comparing your performance against all users. Returns percentile rank, savings by vendor, and average discount."),
		mcp.WithString("vendor", mcp.Description("Filter by vendor (optional)")),
		mcp.WithString("category", mcp.Description("Filter by category (optional)")),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 1y (default: 90d)")),
	), ns.handleBenchmarkReport)

	// Tool 5c: negotiate_price_trends
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_price_trends",
		mcp.WithDescription("Analyze price trends for a vendor SKU. Uses linear regression to determine direction (up/down/stable), volatility, and forecasts."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 6m, 1y, 2y (default: 1y)")),
	), ns.handlePriceTrends)
	// Tool 5d: negotiate_win_loss_analysis
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_win_loss_analysis",
		mcp.WithDescription("Analyze win/loss rates for negotiations. Returns win rate percentage, breakdowns by strategy and vendor, and monthly trends."),
		mcp.WithString("vendor", mcp.Description("Filter by vendor (optional)")),
		mcp.WithString("strategy", mcp.Description("Filter by strategy (optional)")),
		mcp.WithString("period", mcp.Description("Time period: all, 30d, 90d, 1y (default: all)")),
	), ns.handleWinLossAnalysis)

	// Tool 5e: negotiate_export_data
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_export_data",
		mcp.WithDescription("Export negotiation data in CSV or JSON format. Supports deals, sessions, analytics, or all data."),
		mcp.WithString("format", mcp.Description("Export format: csv or json (default: csv)")),
		mcp.WithString("type", mcp.Description("Data type: deals, sessions, analytics, or all (default: deals)")),
		mcp.WithString("vendor", mcp.Description("Filter by vendor (optional)")),
		mcp.WithString("date_from", mcp.Description("Filter by start date (RFC3339, optional)")),
		mcp.WithString("date_to", mcp.Description("Filter by end date (RFC3339, optional)")),
	), ns.handleExportData)

	// Tool 5f: negotiate_set_preferences
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_preferences",
		mcp.WithDescription("Set notification preferences for a channel. Configure enabled notification types, digest frequency, and webhook URL."),
		mcp.WithString("channel", mcp.Required(), mcp.Description("Notification channel: slack or webhook")),
		mcp.WithArray("enabled_types", mcp.WithStringItems(), mcp.Description("Enabled notification types (e.g., deal_closed, renewal, alert)")),
		mcp.WithString("digest_frequency", mcp.Description("Digest frequency: daily, weekly, or never (default: never)")),
		mcp.WithString("webhook_url", mcp.Description("Webhook URL (required for webhook channel)")),
	), ns.handleSetPreferences)

	// Tool 5g: negotiate_get_preferences
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_get_preferences",
		mcp.WithDescription("Get current notification preferences."),
	), ns.handleGetPreferences)

	// Tool 5h: negotiate_send_notification
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_send_notification",
		mcp.WithDescription("Send a notification via the configured channel. Logs the notification and sends to webhook if configured."),
		mcp.WithString("type", mcp.Required(), mcp.Description("Notification type (e.g., deal_closed, renewal, alert)")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Notification message body")),
		mcp.WithString("priority", mcp.Description("Priority: low, normal, high, urgent (default: normal)")),
	), ns.handleSendNotification)

	// Tool 5i: negotiate_budget_dashboard
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_budget_dashboard",
		mcp.WithDescription("Get budget vs actual spend dashboard. Shows variance, monthly trends, and overspend warnings."),
		mcp.WithString("period", mcp.Description("Period: monthly, quarterly, yearly (default: monthly)")),
	), ns.handleBudgetDashboard)

	// Tool 5j: negotiate_set_budget
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_budget",
		mcp.WithDescription("Set or update a budget for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithNumber("budget_amount", mcp.Required(), mcp.Description("Budget amount")),
		mcp.WithString("period_month", mcp.Description("Period month (YYYY-MM, optional)")),
	), ns.handleSetBudget)

	// Tool 5k: negotiate_delete_budget
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_delete_budget",
		mcp.WithDescription("Delete a budget for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
	), ns.handleDeleteBudget)

	// Tool 5m: negotiate_set_monthly_budget
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_monthly_budget",
		mcp.WithDescription("Set the monthly budget allocation for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("month", mcp.Required(), mcp.Description("Month (YYYY-MM)")),
		mcp.WithNumber("budget_amount", mcp.Required(), mcp.Description("Budget amount")),
	), ns.handleSetMonthlyBudget)

	// Tool 5n: negotiate_budget_forecast
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_budget_forecast",
		mcp.WithDescription("Get budget forecast for a vendor (YTD vs projected annual)."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
	), ns.handleBudgetForecast)

	// Tool 5o: negotiate_set_spending_cap
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_spending_cap",
		mcp.WithDescription("Set soft and hard spending caps for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithNumber("soft_cap", mcp.Required(), mcp.Description("Soft cap amount")),
		mcp.WithNumber("hard_cap", mcp.Description("Hard cap amount (optional)")),
		mcp.WithString("period", mcp.Description("Period: monthly, quarterly, yearly (default: monthly)")),
	), ns.handleSetSpendingCap)

	// Tool 5p: negotiate_check_spending_caps
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_spending_caps",
		mcp.WithDescription("Check all spending caps against current spend."),
	), ns.handleCheckSpendingCaps)

	// Tool 5q: negotiate_delete_spending_cap
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_delete_spending_cap",
		mcp.WithDescription("Delete a spending cap for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
	), ns.handleDeleteSpendingCap)

	// Tool 5r: negotiate_record_realization
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_record_realization",
		mcp.WithDescription("Record savings realization for a deal."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Negotiation session ID")),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithNumber("projected_amount", mcp.Required(), mcp.Description("Projected savings amount")),
		mcp.WithNumber("actual_amount", mcp.Required(), mcp.Description("Actual savings amount")),
		mcp.WithString("period", mcp.Description("Period (default: monthly)")),
	), ns.handleRecordRealization)

	// Tool 5s: negotiate_realization_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_realization_report",
		mcp.WithDescription("Get aggregated savings realization report."),
		mcp.WithString("period", mcp.Description("Period filter (default: 90d)")),
	), ns.handleRealizationReport)

	// Tool 5l: negotiate_vendor_spend
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_vendor_spend",
		mcp.WithDescription("Get vendor spend analytics. Aggregates deal outcomes by vendor with spend percentage, monthly trends, and YoY comparison."),
		mcp.WithString("vendor", mcp.Description("Filter by vendor name (optional)")),
		mcp.WithString("period", mcp.Description("Period: 30d, 90d, 1y (default: 1y)")),
	), ns.handleVendorSpend)

	// Tool 5m: negotiate_effectiveness_score
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_effectiveness_score",
		mcp.WithDescription("Get negotiation effectiveness score (0-100) with component breakdown, trend, and improvement tips."),
		mcp.WithString("user_id", mcp.Description("User ID for streak info (optional)")),
		mcp.WithString("period", mcp.Description("Period: 30d, 90d, 1y (default: 90d)")),
	), ns.handleEffectivenessScore)
	// Tool 5n: negotiate_enable_price_alert
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_enable_price_alert",
		mcp.WithDescription("Enable a price drop alert for a vendor/SKU. Records baseline price and monitors for drops exceeding the threshold."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
		mcp.WithNumber("threshold_pct", mcp.Description("Price drop percentage to trigger alert (default: 10)")),
		mcp.WithString("channel", mcp.Description("Notification channel (default: webhook)")),
	), ns.handleEnablePriceAlert)

	// Tool 5o: negotiate_check_price_alerts
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_price_alerts",
		mcp.WithDescription("Check all enabled price alerts against current prices. Returns results with drop percentages and threshold status."),
	), ns.handleCheckPriceAlerts)

	// Tool 5p: negotiate_disable_price_alert
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_disable_price_alert",
		mcp.WithDescription("Disable a price alert for a vendor/SKU."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
	), ns.handleDisablePriceAlert)

	// Tool 5q: negotiate_check_renewal_reminders
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_renewal_reminders",
		mcp.WithDescription("Check upcoming contract renewals and categorize by urgency: critical (<7d), soon (<30d), upcoming (<60d)."),
	), ns.handleCheckRenewalReminders)

	// Tool 5r: negotiate_check_budget_alerts
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_budget_alerts",
		mcp.WithDescription("Check all vendor budgets against actual spend. Flags info (>80%), warning (>90%), and critical (>100%) levels."),
	), ns.handleCheckBudgetAlerts)

	// Tool 5s: negotiate_budget_alert_history
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_budget_alert_history",
		mcp.WithDescription("Get budget alert history for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithInteger("limit", mcp.Description("Max records (default: 10)")),
	), ns.handleBudgetAlertHistory)
	// Tool: negotiate_build_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_build_report",
		mcp.WithDescription("Build a custom report from analytics data sources. Supports sections: savings, vendor_breakdown, win_loss, benchmarks, budget, trends."),
		mcp.WithArray("sections", mcp.Required(), mcp.WithStringItems(), mcp.Description("Report sections to include: savings, vendor_breakdown, win_loss, benchmarks, budget, trends")),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 1y, all (default: all)")),
		mcp.WithString("vendor", mcp.Description("Filter by vendor name (optional)")),
	), ns.handleBuildReport)

	// Tool: negotiate_pricing_index
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_pricing_index",
		mcp.WithDescription("Get competitive pricing index for a category or vendor. Returns average prices, ranges, and vendor breakdown."),
		mcp.WithString("category", mcp.Description("Category filter (e.g., ai, Communication) (optional)")),
		mcp.WithString("vendor", mcp.Description("Vendor filter (optional)")),
	), ns.handlePricingIndex)

	// Tool: negotiate_price_chart
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_price_chart",
		mcp.WithDescription("Get price history chart data for a vendor. Returns monthly labels, datasets (list_price, negotiated), and summary stats."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Description("Product SKU (optional)")),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 1y, 2y (default: 1y)")),
	), ns.handlePriceChart)

	// Tool: negotiate_list_contract_templates
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_contract_templates",
		mcp.WithDescription("List available contract templates, optionally filtered by category."),
		mcp.WithString("category", mcp.Description("Category filter (optional)")),
	), ns.handleListContractTemplates)

	// Tool: negotiate_generate_contract
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_generate_contract",
		mcp.WithDescription("Generate a contract from a template by filling in vendor name and custom parameters."),
		mcp.WithString("template_id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithObject("params", mcp.Description("Custom parameters as key-value object (optional)")),
	), ns.handleGenerateContract)

	// Tool: negotiate_contract_risk
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_contract_risk",
		mcp.WithDescription("Analyze contract text for risky clauses and return a risk report."),
		mcp.WithString("contract_text", mcp.Required(), mcp.Description("Full contract text to analyze")),
	), ns.handleContractRisk)

	// Tool: negotiate_vendor_scorecard
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_vendor_scorecard",
		mcp.WithDescription("Get a vendor scorecard with pricing, reliability, support, and relationship scores."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("period", mcp.Description("Time period: 1y, 90d, 30d (default: 1y)")),
	), ns.handleVendorScorecard)

	// Tool: negotiate_share_strategy
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_share_strategy",
		mcp.WithDescription("Share a negotiation strategy with a name, notes, and type."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Strategy name")),
		mcp.WithString("notes", mcp.Description("Notes about the strategy")),
		mcp.WithString("strategy_type", mcp.Description("Strategy type: aggressive, balanced, conservative (default: balanced)")),
	), ns.handleShareStrategy)

	// Tool: negotiate_list_shared_strategies
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_shared_strategies",
		mcp.WithDescription("List all shared strategies."),
	), ns.handleListSharedStrategies)

	// Tool: negotiate_import_strategy
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_import_strategy",
		mcp.WithDescription("Import a shared strategy by ID. Increments its usage count."),
		mcp.WithString("strategy_id", mcp.Required(), mcp.Description("ID of the strategy to import")),
	), ns.handleImportStrategy)

	// Tool: negotiate_add_note
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_add_note",
		mcp.WithDescription("Add a note to a negotiation session."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Negotiation session ID")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Note content")),
	), ns.handleAddNote)

	// Tool: negotiate_list_notes
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_notes",
		mcp.WithDescription("List all notes for a negotiation session."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Negotiation session ID")),
	), ns.handleListNotes)

	// Tool: negotiate_delete_note
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_delete_note",
		mcp.WithDescription("Delete a note by ID."),
		mcp.WithInteger("note_id", mcp.Required(), mcp.Description("Note ID to delete")),
	), ns.handleDeleteNote)

	// Tool: negotiate_request_approval
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_request_approval",
		mcp.WithDescription("Request approval for a negotiation action."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Negotiation session ID")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("Reason for approval")),
		mcp.WithNumber("threshold", mcp.Description("Price threshold (optional)")),
	), ns.handleRequestApproval)

	// Tool: negotiate_approve
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_approve",
		mcp.WithDescription("Approve a pending approval request."),
		mcp.WithString("approval_id", mcp.Required(), mcp.Description("Approval ID to approve")),
	), ns.handleApprove)

	// Tool: negotiate_reject
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_reject",
		mcp.WithDescription("Reject a pending approval request."),
		mcp.WithString("approval_id", mcp.Required(), mcp.Description("Approval ID to reject")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("Reason for rejection")),
	), ns.handleReject)

	// Tool: negotiate_pending_approvals
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_pending_approvals",
		mcp.WithDescription("List all pending approval requests."),
	), ns.handlePendingApprovals)

	// Tool: negotiate_tco
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_tco",
		mcp.WithDescription("Calculate Total Cost of Ownership for a SaaS vendor product. Returns per-unit cost, annual subscription, 1y/3y TCO, cost per user per month, market comparison, and flagged hidden costs."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Required(), mcp.Description("Product SKU")),
		mcp.WithInteger("seats", mcp.Description("Number of seats (default 50)")),
		mcp.WithInteger("term_months", mcp.Description("Contract term in months (default 12)")),
		mcp.WithNumber("implementation_costs", mcp.Description("One-time implementation costs (default 0)")),
		mcp.WithNumber("training_costs", mcp.Description("Training costs (default 0)")),
		mcp.WithNumber("support_costs", mcp.Description("Support costs (default 0)")),
	), ns.handleTCO)

	// Tool: negotiate_import_data
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_import_data",
		mcp.WithDescription("Import deals or pricing data from JSON. Supports validate mode (dry-run preview) and import mode (inserts records)."),
		mcp.WithString("type", mcp.Required(), mcp.Description("Data type: deals or pricing")),
		mcp.WithString("data", mcp.Required(), mcp.Description("JSON array of records to import")),
		mcp.WithString("mode", mcp.Description("Import mode: validate or import (default import)")),
		mcp.WithBoolean("dry_run", mcp.Description("If true, only validates without inserting (default false)")),
	), ns.handleImportData)

	// Tool: negotiate_set_allocation
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_allocation",
		mcp.WithDescription("Set a cost allocation percentage for a vendor to a department."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("department", mcp.Required(), mcp.Description("Department name")),
		mcp.WithNumber("allocation_pct", mcp.Required(), mcp.Description("Allocation percentage (0-100)")),
	), ns.handleSetAllocation)

	// Tool: negotiate_cost_allocation_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_cost_allocation_report",
		mcp.WithDescription("Generate a cost allocation report showing spend distribution across departments by vendor."),
		mcp.WithString("period", mcp.Description("Time period: 30d, 90d, 1y (default 90d)")),
	), ns.handleCostAllocationReport)

	// Tool: negotiate_alert_history
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_alert_history",
		mcp.WithDescription("Retrieve merged alert feed from budget, renewal, and price change sources."),
		mcp.WithString("type", mcp.Description("Alert type filter: all, budget, renewal, price_change (default all)")),
		mcp.WithString("vendor", mcp.Description("Filter by vendor")),
		mcp.WithInteger("limit", mcp.Description("Max results (default 50)")),
	), ns.handleAlertHistory)

	// Tool: negotiate_sla_credit
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_sla_credit",
		mcp.WithDescription("Calculate SLA credit eligibility and amount for a vendor service."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithNumber("monthly_spend", mcp.Required(), mcp.Description("Monthly spend amount")),
		mcp.WithNumber("uptime_pct", mcp.Required(), mcp.Description("Actual uptime percentage")),
		mcp.WithNumber("guaranteed_uptime", mcp.Description("Guaranteed uptime percentage (default 99.9)")),
		mcp.WithNumber("credit_rate", mcp.Description("Credit rate percentage (default 5)")),
	), ns.handleSLACredit)

	// Tool: negotiate_log_communication
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_log_communication",
		mcp.WithDescription("Log a vendor communication entry."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Communication type (email, call, meeting, etc.)")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Brief summary")),
		mcp.WithString("detail", mcp.Description("Detailed notes")),
	), ns.handleLogCommunication)

	// Tool: negotiate_communication_history
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_communication_history",
		mcp.WithDescription("Retrieve communication history for a vendor."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithInteger("limit", mcp.Description("Max results (default 20)")),
	), ns.handleCommunicationHistory)

	// Tool: negotiate_analyze_offer
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_analyze_offer",
		mcp.WithDescription("Analyze a time-limited vendor offer. Returns savings, urgency, recommendation, and price comparison."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("sku", mcp.Required(), mcp.Description("Product SKU")),
		mcp.WithNumber("offer_price", mcp.Required(), mcp.Description("Offered price")),
		mcp.WithString("expires_at", mcp.Required(), mcp.Description("Offer expiration (RFC3339)")),
		mcp.WithNumber("current_price", mcp.Description("Current price per unit (optional)")),
		mcp.WithNumber("current_spend", mcp.Description("Current total spend (optional)")),
	), ns.handleAnalyzeOffer)

	// Tool: negotiate_refresh_pricing
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_refresh_pricing",
		mcp.WithDescription("Refresh pricing snapshots for vendors with ±3% variation. Creates new trend data points."),
		mcp.WithArray("vendors", mcp.WithStringItems(), mcp.Description("Vendor list (empty = all)")),
		mcp.WithString("source", mcp.Description("Data source label (default: seed)")),
	), ns.handleRefreshPricing)

	// Tool: negotiate_rate_limit_dashboard
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_rate_limit_dashboard",
		mcp.WithDescription("Get current rate limit usage status — requests this minute, hour, day, and status color."),
	), ns.handleRateLimitDashboard)

	// Tool: negotiate_log_api_request
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_log_api_request",
		mcp.WithDescription("Log an API request for rate limit tracking."),
		mcp.WithString("api_key_id", mcp.Required(), mcp.Description("API key identifier")),
		mcp.WithString("endpoint", mcp.Required(), mcp.Description("API endpoint called")),
	), ns.handleLogAPIRequest)

	// Tool: negotiate_api_docs
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_api_docs",
		mcp.WithDescription("Generate API documentation for all registered MCP tools. Returns markdown or JSON."),
		mcp.WithString("format", mcp.Description("Output format: markdown or json (default: markdown)")),
		mcp.WithString("tool", mcp.Description("Optional: filter by tool name")),
	), ns.handleAPIDocs)

	// Tool: negotiate_tool_stats
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_tool_stats",
		mcp.WithDescription("Get tool usage statistics for a given period. Shows top and bottom tools by call count."),
		mcp.WithString("period", mcp.Description("Time period: 24h, 7d, or 30d (default: 7d)")),
	), ns.handleToolStats)

	// Tool: negotiate_health
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_health",
		mcp.WithDescription("Get server health status including database connectivity, tool count, DB size, and uptime."),
	), ns.handleHealth)
	// Tool: negotiate_cli_autocomplete
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_cli_autocomplete",
		mcp.WithDescription("Generate shell completion script for the a2a-cli command. Supports bash and zsh shells."),
		mcp.WithString("shell", mcp.Description("Target shell: bash or zsh (default: bash)")),
	), ns.handleCLIAutocomplete)

	// Tool: negotiate_metrics
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_metrics",
		mcp.WithDescription("Generate Prometheus-format metrics for the negotiation server. Includes negotiation totals, deal counts, savings, and active sessions."),
	), ns.handleMetrics)

	// Tool: negotiate_shutdown
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_shutdown",
		mcp.WithDescription("Perform graceful shutdown of the negotiation server. Closes database connections and other resources."),
	), ns.handleShutdown)

	// Tool: negotiate_coverage
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_coverage",
		mcp.WithDescription("Generate a code coverage report by running go test -cover on the project. Returns per-package coverage percentages and recommendations."),
	), ns.handleCoverage)

	// Tool: negotiate_dependencies
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_dependencies",
		mcp.WithDescription("Parse go.mod to produce a dependency report. Returns direct and indirect dependencies with version information."),
		mcp.WithString("format", mcp.Description("Output format: json (default)")),
	), ns.handleDependencies)

	// Tool: negotiate_contribution_guide
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_contribution_guide",
		mcp.WithDescription("Generate a CONTRIBUTING.md guide based on the project's structure. Includes setup, build, test, and conventions."),
	), ns.handleContributionGuide)

	// Tool: negotiate_rotate_key
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_rotate_key",
		mcp.WithDescription("Rotate an API key — revokes the existing key and generates a new replacement key."),
		mcp.WithString("key_id", mcp.Required(), mcp.Description("ID of the API key to rotate")),
	), ns.handleRotateKey)

	// Tool: negotiate_key_health
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_key_health",
		mcp.WithDescription("Retrieve the health status of all API keys — includes status, owner, creation date, expiry, and last rotation."),
	), ns.handleKeyHealth)

	// Tool: negotiate_add_ip
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_add_ip",
		mcp.WithDescription("Add an IP address to the whitelist with a descriptive label."),
		mcp.WithString("ip_address", mcp.Required(), mcp.Description("IP address to whitelist")),
		mcp.WithString("label", mcp.Required(), mcp.Description("Descriptive label for this IP")),
	), ns.handleAddIP)

	// Tool: negotiate_remove_ip
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_remove_ip",
		mcp.WithDescription("Remove an IP address from the whitelist."),
		mcp.WithString("ip_address", mcp.Required(), mcp.Description("IP address to remove")),
	), ns.handleRemoveIP)

	// Tool: negotiate_list_whitelist
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_whitelist",
		mcp.WithDescription("List all IP addresses currently on the whitelist."),
	), ns.handleListWhitelist)

	// Tool: negotiate_set_retention
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_retention",
		mcp.WithDescription("Set a data retention policy for a given data type. Supported types: sessions, outcomes, alerts, audit_log, usage_stats. Actions: delete, archive."),
		mcp.WithString("data_type", mcp.Required(), mcp.Description("Data type to set retention for (sessions, outcomes, alerts, audit_log, usage_stats)")),
		mcp.WithInteger("retention_days", mcp.Required(), mcp.Description("Number of days to retain data")),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action after retention period (delete, archive)")),
	), ns.handleSetRetention)

	// Tool: negotiate_get_retention
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_get_retention",
		mcp.WithDescription("List all data retention policies."),
	), ns.handleGetRetention)

	// Tool: negotiate_purge_old_data
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_purge_old_data",
		mcp.WithDescription("Purge old data according to retention policies. Defaults to dry-run (simulation) unless dry_run=false."),
		mcp.WithBoolean("dry_run", mcp.Description("If true (default), simulate the purge without actually deleting. Set to false to execute.")),
	), ns.handlePurgeOldData)

	// Tool: negotiate_simulate
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_simulate",
		mcp.WithDescription("Run a negotiation training simulation with configurable strategy and parameters."),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
		mcp.WithString("strategy", mcp.Required(), mcp.Description("Negotiation strategy: competitive, collaborative, aggressive, concessionary, or principled")),
		mcp.WithNumber("budget", mcp.Required(), mcp.Description("Budget amount")),
		mcp.WithInteger("rounds", mcp.Description("Number of simulation rounds (1-10, default 3)")),
	), ns.handleSimulate)

	// Tool: negotiate_save_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_save_report",
		mcp.WithDescription("Save an industry research report."),
		mcp.WithString("title", mcp.Required(), mcp.Description("Report title")),
		mcp.WithString("category", mcp.Required(), mcp.Description("Report category")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Report content")),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source URL or name")),
	), ns.handleSaveReport)

	// Tool: negotiate_list_reports
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_reports",
		mcp.WithDescription("List saved industry reports, optionally filtered by category."),
		mcp.WithString("category", mcp.Description("Optional category filter")),
	), ns.handleListReports)

	// Tool: negotiate_get_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_get_report",
		mcp.WithDescription("Get details of a saved industry report by ID."),
		mcp.WithInteger("report_id", mcp.Required(), mcp.Description("Report ID")),
	), ns.handleGetReport)

	// Tool: negotiate_list_webhook_events
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_webhook_events",
		mcp.WithDescription("View webhook events, optionally filtered by status."),
		mcp.WithString("status", mcp.Description("Optional status filter (e.g. success, failed, pending)")),
		mcp.WithInteger("limit", mcp.Description("Maximum number of events to return (default 50)")),
	), ns.handleListWebhookEvents)

	// Tool: negotiate_webhook_event_detail
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_webhook_event_detail",
		mcp.WithDescription("Get full details of a specific webhook event."),
		mcp.WithInteger("event_id", mcp.Required(), mcp.Description("Webhook event ID")),
	), ns.handleWebhookEventDetail)

	// Tool: negotiate_replay_webhook_event
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_replay_webhook_event",
		mcp.WithDescription("Replay a webhook event by incrementing its attempt count and marking it as replayed."),
		mcp.WithInteger("event_id", mcp.Required(), mcp.Description("Webhook event ID to replay")),
	), ns.handleReplayWebhookEvent)

	// Tool: negotiate_webhook_stats
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_webhook_stats",
		mcp.WithDescription("Get aggregated webhook event statistics including success rate and status breakdown."),
	), ns.handleWebhookStats)


        // Tool: negotiate_ingest_document
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_ingest_document",
                mcp.WithDescription("Ingest a vendor document into the knowledge base for RAG search."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
                mcp.WithString("content", mcp.Required(), mcp.Description("Document content")),
                mcp.WithString("doc_type", mcp.Required(), mcp.Description("Document type (e.g. contract, compliance, report)")),
        ), ns.handleIngestDocument)

        // Tool: negotiate_search_vendor_docs
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_search_vendor_docs",
                mcp.WithDescription("Search vendor knowledge documents by vendor and query text."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
                mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
        ), ns.handleSearchVendorDocs)

        // Tool: negotiate_vendor_knowledge_report
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_vendor_knowledge_report",
                mcp.WithDescription("Get a knowledge report summary for a vendor, including total docs and doc type breakdown."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
        ), ns.handleVendorKnowledgeReport)

	// Tool: negotiate_ai_performance
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_ai_performance",
		mcp.WithDescription("Log an AI agent performance call. Use this to track latency, token usage, success rate, and negotiation type for any AI model used during a negotiation workflow."),
		mcp.WithString("model", mcp.Required(), mcp.Description("AI model name (e.g. gpt-4, claude-3)")),
		mcp.WithString("tool_name", mcp.Required(), mcp.Description("Name of the tool invoked by the AI agent")),
		mcp.WithNumber("latency_ms", mcp.Required(), mcp.Description("Response latency in milliseconds")),
		mcp.WithNumber("tokens_used", mcp.Required(), mcp.Description("Number of tokens consumed")),
		mcp.WithBoolean("success", mcp.Required(), mcp.Description("Whether the call was successful")),
		mcp.WithString("negotiation_type", mcp.Description("Type of negotiation (e.g. price_query, create_session, strategy)")),
	), ns.handleAIPerformance)

        // Tool: negotiate_summarize_session
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_summarize_session",
                mcp.WithDescription("Generate a concise summary of a completed negotiation session. Supports brief, detailed, and bullet_points styles."),
                mcp.WithString("session_id", mcp.Required(), mcp.Description("Session ID from negotiate_create_session")),
                mcp.WithString("style", mcp.Description("Summary style: brief, detailed, or bullet_points (default: bullet_points)")),
        ), ns.handleSummarizeSession)

	// Tool: negotiate_sentiment
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_sentiment",
			mcp.WithDescription("Analyze the sentiment of vendor communication text. Returns a score from -1.0 (negative) to 1.0 (positive), confidence level, label, and key phrases found."),
			mcp.WithString("text", mcp.Required(), mcp.Description("The vendor communication text to analyze (max 10,000 characters)")),
	), ns.handleSentiment)

		// Tool: negotiate_translate
		ns.mcpServer.AddTool(mcp.NewTool("negotiate_translate",
			mcp.WithDescription("Translate negotiation text between supported languages. Supports en, es, fr, de, zh, ja, ar."),
			mcp.WithString("text", mcp.Required(), mcp.Description("The text to translate")),
			mcp.WithString("target_language", mcp.Required(), mcp.Description("Target language code (e.g. en, es, fr, de, zh, ja, ar)")),
		), ns.handleTranslate)

		// Tool: negotiate_set_language_preference
		ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_language_preference",
			mcp.WithDescription("Set a vendor's preferred language for negotiation communications."),
			mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
			mcp.WithString("language", mcp.Required(), mcp.Description("Language code (e.g. en, es, fr, de, zh, ja, ar)")),
		), ns.handleSetLanguagePreference)

		// Tool: negotiate_language_glossary
		ns.mcpServer.AddTool(mcp.NewTool("negotiate_language_glossary",
			mcp.WithDescription("Create or retrieve a glossary of terms for a language pair. Terms should be provided as a JSON array of strings."),
			mcp.WithString("terms", mcp.Required(), mcp.Description("JSON array of terms to translate (e.g. [\"hello\",\"world\"])")),
			mcp.WithString("from", mcp.Required(), mcp.Description("Source language code")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target language code")),
		), ns.handleLanguageGlossary)

                // Tool: negotiate_compliance_check
                ns.mcpServer.AddTool(mcp.NewTool("negotiate_compliance_check",
                        mcp.WithDescription("Check negotiation terms against regulatory compliance requirements for a given jurisdiction (gdpr, soc2, hipaa, ccpa). Returns pass/fail per rule and an overall compliance status."),
                        mcp.WithString("terms", mcp.Required(), mcp.Description("The negotiation terms / agreement text to check")),
                        mcp.WithString("jurisdiction", mcp.Required(), mcp.Description("Regulatory jurisdiction: gdpr, soc2, hipaa, or ccpa")),
                ), ns.handleComplianceCheck)
        // Tool: negotiate_list_clauses
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_clauses",
                mcp.WithDescription("List standard contract clauses, optionally filtered by category."),
                mcp.WithString("category", mcp.Description("Optional category filter (e.g. payment, termination, confidentiality)")),
        ), ns.handleListClauses)

        // Tool: negotiate_get_clause
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_get_clause",
                mcp.WithDescription("Get a contract clause by its ID."),
                mcp.WithInteger("clause_id", mcp.Required(), mcp.Description("Clause ID")),
        ), ns.handleGetClause)

        // Tool: negotiate_search_clauses
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_search_clauses",
                mcp.WithDescription("Search contract clauses by keyword across title, content, and description."),
                mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
        ), ns.handleSearchClauses)

        // Tool: negotiate_add_clause
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_add_clause",
                mcp.WithDescription("Add a new contract clause to the library."),
                mcp.WithString("category", mcp.Required(), mcp.Description("Clause category")),
                mcp.WithString("title", mcp.Required(), mcp.Description("Clause title")),
                mcp.WithString("content", mcp.Required(), mcp.Description("Clause content / legal text")),
                mcp.WithString("description", mcp.Description("Optional description")),
        ), ns.handleAddClause)

        // Tool: negotiate_send_for_signature
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_send_for_signature",
                mcp.WithDescription("Send a contract for e-signature to a signer."),
                mcp.WithString("contract_id", mcp.Required(), mcp.Description("Contract ID to send for signing")),
                mcp.WithString("signer_email", mcp.Required(), mcp.Description("Email address of the signer")),
        ), ns.handleSendForSignature)

        // Tool: negotiate_signature_status
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_signature_status",
                mcp.WithDescription("Check the current status of an e-signature envelope."),
                mcp.WithString("envelope_id", mcp.Required(), mcp.Description("E-signature envelope ID")),
        ), ns.handleSignatureStatus)

        // Tool: negotiate_signed_document
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_signed_document",
                mcp.WithDescription("Retrieve a signed document from an e-signature envelope."),
                mcp.WithString("envelope_id", mcp.Required(), mcp.Description("E-signature envelope ID")),
        ), ns.handleSignedDocument)

        // Tool: negotiate_set_data_residency
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_set_data_residency",
                mcp.WithDescription("Set data residency rule for a region. Controls whether data in a specific geographic region is allowed or blocked."),
                mcp.WithString("region", mcp.Required(), mcp.Description("Geographic region (e.g. eu, us, cn)")),
                mcp.WithBoolean("allowed", mcp.Required(), mcp.Description("Whether data residency is allowed in this region")),
        ), ns.handleSetDataResidency)

        // Tool: negotiate_check_residency
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_check_residency",
                mcp.WithDescription("Check if a vendor's data storage region is compliant with configured data residency rules."),
                mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name (e.g. Slack, GitHub)")),
        ), ns.handleCheckResidency)

        // Tool: negotiate_residency_report
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_residency_report",
		mcp.WithDescription("Generate a data residency compliance report showing all configured rules and their status."),
	), ns.handleResidencyReport)

	// P95: Dashboard Widget Tools
	ns.mcpServer.AddTool(mcp.NewTool("negotiate_create_widget",
		mcp.WithDescription("Create a new dashboard widget for negotiation metrics."),
		mcp.WithString("widget_type", mcp.Required(), mcp.Description("Widget type (e.g. price_chart, metric, table)")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Widget display title")),
		mcp.WithString("config", mcp.Description("Widget configuration JSON (default: {})")),
	), ns.handleCreateWidget)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_list_widgets",
		mcp.WithDescription("List all dashboard widgets ordered by creation date."),
	), ns.handleListWidgets)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_render_dashboard",
		mcp.WithDescription("Render a dashboard containing specific widgets by their IDs."),
		mcp.WithString("widget_ids", mcp.Description("JSON array of widget IDs to include (optional — returns all if omitted)")),
	), ns.handleRenderDashboard)

	ns.mcpServer.AddTool(mcp.NewTool("negotiate_export_dashboard",
		mcp.WithDescription("Export all dashboard widgets as a JSON array."),
		mcp.WithString("format", mcp.Description("Export format: \"json\" for pretty-printed JSON, or any other value for compact JSON (default: json)")),
	), ns.handleExportDashboard)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_export_chart",
                mcp.WithDescription("Export negotiation data as a chart image (PNG or SVG)."),
                mcp.WithString("data_source", mcp.Required(), mcp.Description("Name of the data source to chart")),
                mcp.WithString("chart_type", mcp.Required(), mcp.Description("Chart type: bar, line, pie, area, scatter")),
                mcp.WithString("format", mcp.Required(), mcp.Description("Export format: png or svg")),
        ), ns.handleExportChart)

        ns.mcpServer.AddTool(mcp.NewTool("negotiate_chart_templates",
                mcp.WithDescription("List predefined chart templates available for export."),
        ), ns.handleChartTemplates)

        // P97: Real-Time Monitoring Dashboard
        ns.mcpServer.AddTool(mcp.NewTool("negotiate_live_dashboard",
                mcp.WithDescription("Return real-time monitoring dashboard data including active negotiations, system health, recent tool calls, and error rate."),
        ), ns.handleLiveDashboard)

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
		"api_key":     key,
		"owner":       owner,
		"note":        "This key will not be shown again. Store it securely.",
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
	"gpt-4o":            {"chat", "vision", "code"},
	"gpt-4o-mini":       {"chat", "code"},
	"gpt-4o-audio":      {"audio"},
	"o1":                {"reasoning", "code"},
	"o1-mini":           {"reasoning", "code"},
	"dall-e-3":          {"image_generation"},
	"whisper-1":         {"audio"},
	"tts-1":             {"audio"},
	"claude-3.5-sonnet": {"chat", "vision", "code"},
	"claude-3-opus":     {"chat", "reasoning", "code"},
	"claude-3-haiku":    {"chat", "code"},
	"gemini-2.5-flash":  {"chat", "vision", "code"},
	"gemini-2.5-pro":    {"chat", "reasoning", "vision", "code"},
	"gemini-2.0-flash":  {"chat", "vision", "code"},
	"deepseek-v4":       {"chat", "reasoning", "code"},
	"deepseek-r1":       {"reasoning", "code"},
	"deepseek-chat":     {"chat", "code"},
	"mistral-large":     {"chat", "code"},
	"mistral-small":     {"chat", "code"},
	"command-r-plus":    {"chat", "code"},
	"command-r":         {"chat", "code"},
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
		"vendor":           rep.Vendor,
		"deal_count":       rep.DealCount,
		"avg_discount_pct": rep.AvgDiscountPct,
		"max_discount_pct": rep.MaxDiscountPct,
		"negotiability":    rep.Negotiability,
		"win_rate":         rep.WinRate,
		"duration_ms":      time.Since(start).Milliseconds(),
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
		"vendor":               rec.Vendor,
		"recommended_strategy": rec.RecommendedStrategy,
		"confidence":           rec.Confidence,
		"avg_discount_pct":     rec.AvgDiscount,
		"total_deals":          rec.TotalDeals,
		"breakdown":            rec.Breakdown,
		"duration_ms":          time.Since(start).Milliseconds(),
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
		"vendor":      vendor,
		"patterns":    patterns,
		"count":       len(patterns),
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
		"patterns":    patterns,
		"count":       len(patterns),
		"limit":       limit,
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
		"vendor": leverage.Vendor,
		"health": map[string]any{
			"score":        leverage.Health.Score,
			"category":     leverage.Health.Category,
			"last_updated": leverage.Health.LastUpdated.Format(time.RFC3339),
			"signals":      leverage.Health.Signals,
		},
		"leverage":    leverage.Leverage,
		"suggestion":  leverage.Suggestion,
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
		"vendor":   leverage.Vendor,
		"recorded": true,
		"health": map[string]any{
			"score":    leverage.Health.Score,
			"category": leverage.Health.Category,
		},
		"leverage":    leverage.Leverage,
		"suggestion":  leverage.Suggestion,
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
		"vendors":      vendors,
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
		"sla_id":         contract.ID,
		"vendor":         contract.Vendor,
		"service":        contract.Service,
		"uptime_pct":     contract.UptimePct,
		"credit_pct":     contract.CreditPct,
		"max_credit_pct": contract.MaxCreditPct,
		"monthly_spend":  contract.MonthlySpend,
		"status":         contract.Status,
		"duration_ms":    time.Since(start).Milliseconds(),
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
			"vendor":           analysis.Quote.Vendor,
			"sku":              analysis.Quote.SKU,
			"description":      analysis.Quote.Description,
			"seats":            analysis.Quote.Seats,
			"term_months":      analysis.Quote.TermMonths,
			"price_per_unit":   analysis.Quote.PricePerUnit,
			"total_price":      analysis.Quote.TotalPrice,
			"list_price":       analysis.Quote.ListPrice,
			"discount_offered": analysis.Quote.DiscountOffered,
		},
		"market_range":      analysis.MarketRange,
		"counter_offer_min": analysis.CounterOfferMin,
		"counter_offer_max": analysis.CounterOfferMax,
		"potential_savings": analysis.PotentialSavings,
		"confidence":        analysis.Confidence,
		"duration_ms":       time.Since(start).Milliseconds(),
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
		"user_id":     userID,
		"badges":      badges,
		"count":       len(badges),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── ROI Handlers ───

func (ns *NegotiationServer) handleCalculateROI(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	currentSpend := req.GetFloat("current_spend", 0)
	negotiatedPrice := req.GetFloat("negotiated_price", 0)
	implementationCosts := req.GetFloat("implementation_costs", 0)
	annualOverhead := req.GetFloat("annual_overhead", 0)

	ns.logger.Debug("calculate_roi called", "vendor", vendor)

	calc, err := ns.roiEng.Calculate(ctx, currentSpend, negotiatedPrice, implementationCosts, annualOverhead)
	if err != nil {
		ns.logger.Warn("calculate_roi failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("ROI calculation failed: %s", err.Error())), nil
	}

	calc.Vendor = vendor

	resp := map[string]any{
		"vendor":               calc.Vendor,
		"current_spend":        calc.CurrentSpend,
		"negotiated_price":     calc.NegotiatedPrice,
		"implementation_costs": calc.ImplementationCosts,
		"annual_overhead":      calc.AnnualOverhead,
		"annual_savings":       calc.AnnualSavings,
		"roi_pct":              calc.ROIPct,
		"payback_months":       calc.PaybackMonths,
		"savings_1y":           calc.Savings1Y,
		"savings_3y":           calc.Savings3Y,
		"savings_5y":           calc.Savings5Y,
		"npv":                  calc.NPV,
		"duration_ms":          time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Benchmark Handlers ───

func (ns *NegotiationServer) handleBenchmarkReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	category, _ := req.RequireString("category")
	period, _ := req.RequireString("period")
	if period == "" {
		period = "90d"
	}

	ns.logger.Debug("benchmark_report called", "vendor", vendor, "category", category, "period", period)

	report, err := ns.benchmarkEng.GenerateReport(ctx, "", vendor, category, period)
	if err != nil {
		ns.logger.Warn("benchmark_report failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Benchmark report failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"total_savings":    report.TotalSavings,
		"avg_discount_pct": report.AvgDiscountPct,
		"deal_count":       report.DealCount,
		"percentile":       report.Percentile,
		"by_vendor":        report.ByVendor,
		"period":           report.Period,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Price Trend Handlers ───

func (ns *NegotiationServer) handlePriceTrends(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	period, _ := req.RequireString("period")
	if period == "" {
		period = "1y"
	}

	ns.logger.Debug("price_trends called", "vendor", vendor, "sku", sku, "period", period)

	analysis, err := ns.trendsEng.Analyze(ctx, vendor, sku, period)
	if err != nil {
		ns.logger.Warn("price_trends failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Price trend analysis failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":          analysis.Vendor,
		"sku":             analysis.SKU,
		"period":          analysis.Period,
		"direction":       analysis.Direction,
		"slope":           analysis.Slope,
		"volatility":      analysis.Volatility,
		"price_change_6m": analysis.PriceChange6M,
		"forecast_3m":     analysis.Forecast3M,
		"forecast_6m":     analysis.Forecast6M,
		"seasonal":        analysis.Seasonal,
		"data_points":     analysis.DataPoints,
		"snapshots":       analysis.Snapshots,
		"duration_ms":     time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Win/Loss Handlers ───

func (ns *NegotiationServer) handleWinLossAnalysis(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor := req.GetString("vendor", "")
	strategy := req.GetString("strategy", "")
	period := req.GetString("period", "all")

	ns.logger.Debug("win_loss_analysis called", "vendor", vendor, "strategy", strategy, "period", period)

	report, err := ns.winlossEng.Analyze(ctx, vendor, strategy, period)
	if err != nil {
		ns.logger.Warn("win_loss_analysis failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Win/loss analysis failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"period":        report.Period,
		"total_deals":   report.TotalDeals,
		"won":           report.Won,
		"lost":          report.Lost,
		"pending":       report.Pending,
		"win_rate_pct":  report.WinRate,
		"by_strategy":   report.ByStrategy,
		"by_vendor":     report.ByVendor,
		"monthly_trend": report.MonthlyTrend,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Export Handlers ───

func (ns *NegotiationServer) handleExportData(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	format := req.GetString("format", "csv")
	exportType := req.GetString("type", "deals")
	vendor := req.GetString("vendor", "")
	dateFrom := req.GetString("date_from", "")
	dateTo := req.GetString("date_to", "")

	ns.logger.Debug("export_data called", "format", format, "type", exportType, "vendor", vendor)

	reqData := export.ExportRequest{
		Format:   format,
		Type:     exportType,
		Vendor:   vendor,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}

	result, err := ns.exportEng.Export(ctx, reqData)
	if err != nil {
		ns.logger.Warn("export_data failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Export failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"export_id":    result.ExportID,
		"format":       result.Format,
		"export_type":  result.ExportType,
		"record_count": result.RecordCount,
		"data":         result.Data,
		"filename":     result.Filename,
		"generated_at": result.GeneratedAt,
		"duration_ms":  time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Notification Handlers ───

func (ns *NegotiationServer) handleSetPreferences(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	channel, _ := req.RequireString("channel")
	rawTypes, _ := req.GetArguments()["enabled_types"]
	enabledTypesRaw, _ := rawTypes.([]any)
	digestFreq := req.GetString("digest_frequency", "never")
	webhookURL := req.GetString("webhook_url", "")

	ns.logger.Debug("set_preferences called", "channel", channel, "digest_freq", digestFreq)

	enabledTypes := make([]string, len(enabledTypesRaw))
	for i, v := range enabledTypesRaw {
		enabledTypes[i], _ = v.(string)
	}

	prefs, err := ns.notifyEng.SetPreferences(ctx, channel, enabledTypes, digestFreq, webhookURL)
	if err != nil {
		ns.logger.Warn("set_preferences failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Set preferences failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"user_id":          prefs.UserID,
		"channel":          prefs.Channel,
		"enabled_types":    prefs.EnabledTypes,
		"digest_frequency": prefs.DigestFreq,
		"webhook_url":      prefs.WebhookURL,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGetPreferences(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("get_preferences called")

	prefs, err := ns.notifyEng.GetPreferences(ctx)
	if err != nil {
		ns.logger.Warn("get_preferences failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Get preferences failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"user_id":          prefs.UserID,
		"channel":          prefs.Channel,
		"enabled_types":    prefs.EnabledTypes,
		"digest_frequency": prefs.DigestFreq,
		"webhook_url":      prefs.WebhookURL,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSendNotification(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	notifType, _ := req.RequireString("type")
	message, _ := req.RequireString("message")
	priority := req.GetString("priority", "normal")

	ns.logger.Debug("send_notification called", "type", notifType, "priority", priority)

	n, err := ns.notifyEng.SendNotification(ctx, notifType, message, priority)
	if err != nil {
		ns.logger.Warn("send_notification failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Send notification failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"id":          n.ID,
		"type":        n.Type,
		"channel":     n.Channel,
		"message":     n.Message,
		"priority":    n.Priority,
		"status":      n.Status,
		"created_at":  n.CreatedAt.Format(time.RFC3339),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Budget Dashboard Handlers ───

func (ns *NegotiationServer) handleBudgetDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	period := req.GetString("period", "monthly")

	ns.logger.Debug("budget_dashboard called", "period", period)

	dash, err := ns.budgetEng.Dashboard(ctx, period)
	if err != nil {
		ns.logger.Warn("budget_dashboard failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Budget dashboard failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"period":        dash.Period,
		"total_budget":  dash.TotalBudget,
		"total_actual":  dash.TotalActual,
		"variance":      dash.Variance,
		"variance_pct":  dash.VariancePct,
		"by_vendor":     dash.ByVendor,
		"monthly_trend": dash.MonthlyTrend,
		"warnings":      dash.Warnings,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSetBudget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	budgetAmount := req.GetFloat("budget_amount", 0)
	periodMonth := req.GetString("period_month", "")

	ns.logger.Debug("set_budget called", "vendor", vendor, "amount", budgetAmount)

	if err := ns.budgetEng.Store().SetBudget(ctx, vendor, budgetAmount, periodMonth); err != nil {
		ns.logger.Warn("set_budget failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Set budget failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"budget":      budgetAmount,
		"status":      "set",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDeleteBudget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("delete_budget called", "vendor", vendor)

	if err := ns.budgetEng.Store().DeleteBudget(ctx, vendor); err != nil {
		ns.logger.Warn("delete_budget failed", "vendor", vendor, "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Delete budget failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"status":      "deleted",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Budget Management Handlers ───

func (ns *NegotiationServer) handleSetMonthlyBudget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	month, _ := req.RequireString("month")
	budgetAmount := req.GetFloat("budget_amount", 0)

	ns.logger.Debug("set_monthly_budget called", "vendor", vendor, "month", month, "amount", budgetAmount)

	if err := ns.budgetMgmtEng.SetBudget(ctx, vendor, month, budgetAmount); err != nil {
		ns.logger.Warn("set_monthly_budget failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Set monthly budget failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"month":       month,
		"amount":      budgetAmount,
		"status":      "set",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleBudgetForecast(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("budget_forecast called", "vendor", vendor)

	forecast, err := ns.budgetMgmtEng.GetForecast(ctx, vendor)
	if err != nil {
		ns.logger.Warn("budget_forecast failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Budget forecast failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":           forecast.Vendor,
		"ytd_budget":       forecast.YTDBudget,
		"ytd_spent":        forecast.YTDSpent,
		"projected_annual": forecast.ProjectedAnnual,
		"remaining_months": forecast.RemainingMonths,
		"status":           forecast.Status,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Spending Caps Handlers ───

func (ns *NegotiationServer) handleSetSpendingCap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	softCap := req.GetFloat("soft_cap", 0)
	hardCap := req.GetFloat("hard_cap", 0)
	period := req.GetString("period", "monthly")

	ns.logger.Debug("set_spending_cap called", "vendor", vendor, "soft_cap", softCap, "hard_cap", hardCap, "period", period)

	if err := ns.spendingCapsEng.SetCap(ctx, vendor, softCap, hardCap, period); err != nil {
		ns.logger.Warn("set_spending_cap failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Set spending cap failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"soft_cap":    softCap,
		"hard_cap":    hardCap,
		"period":      period,
		"status":      "set",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCheckSpendingCaps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("check_spending_caps called")

	results, err := ns.spendingCapsEng.CheckCaps(ctx)
	if err != nil {
		ns.logger.Warn("check_spending_caps failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Check spending caps failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"results":     results,
		"count":       len(results),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDeleteSpendingCap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")

	ns.logger.Debug("delete_spending_cap called", "vendor", vendor)

	if err := ns.spendingCapsEng.Store().DeleteCap(ctx, vendor); err != nil {
		ns.logger.Warn("delete_spending_cap failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Delete spending cap failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"status":      "deleted",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Savings Realization Handlers ───

func (ns *NegotiationServer) handleRecordRealization(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	sessionID, _ := req.RequireString("session_id")
	vendor, _ := req.RequireString("vendor")
	projectedAmount := req.GetFloat("projected_amount", 0)
	actualAmount := req.GetFloat("actual_amount", 0)
	period := req.GetString("period", "monthly")

	ns.logger.Debug("record_realization called", "vendor", vendor, "session_id", sessionID)

	result, err := ns.savingsRealizationEng.Record(ctx, sessionID, vendor, projectedAmount, actualAmount, period)
	if err != nil {
		ns.logger.Warn("record_realization failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Record realization failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"id":               result.ID,
		"session_id":       result.SessionID,
		"vendor":           result.Vendor,
		"projected_amount": result.ProjectedAmount,
		"actual_amount":    result.ActualAmount,
		"period":           result.Period,
		"status":           "recorded",
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRealizationReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	period := req.GetString("period", "90d")

	ns.logger.Debug("realization_report called", "period", period)

	report, err := ns.savingsRealizationEng.GetReport(ctx, period)
	if err != nil {
		ns.logger.Warn("realization_report failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Realization report failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"total_projected":  report.TotalProjected,
		"total_realized":   report.TotalRealized,
		"realization_rate": report.RealizationRate,
		"by_vendor":        report.ByVendor,
		"top_shortfalls":   report.TopShortfalls,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Vendor Spend Handler ───

func (ns *NegotiationServer) handleVendorSpend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor := req.GetString("vendor", "")
	period := req.GetString("period", "1y")

	ns.logger.Debug("vendor_spend called", "vendor", vendor, "period", period)

	report, err := ns.vendorspendEng.Report(ctx, vendor, period)
	if err != nil {
		ns.logger.Warn("vendor_spend failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Vendor spend report failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"period":         report.Period,
		"total_spend":    report.TotalSpend,
		"vendors":        report.Vendors,
		"subscriptions":  report.Subscriptions,
		"by_vendor":      report.ByVendor,
		"monthly_trend":  report.MonthlyTrend,
		"yoy_change_pct": report.YoYChangePct,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Effectiveness Score Handler ───

func (ns *NegotiationServer) handleEffectivenessScore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	userID := req.GetString("user_id", "")
	period := req.GetString("period", "90d")

	ns.logger.Debug("effectiveness_score called", "user_id", userID, "period", period)

	score, err := ns.effectivenessEng.Score(ctx, userID, period)
	if err != nil {
		ns.logger.Warn("effectiveness_score failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Effectiveness score failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"user_id":       score.UserID,
		"period":        score.Period,
		"overall_score": score.OverallScore,
		"components":    score.Components,
		"trend":         score.Trend,
		"vs_average":    score.VsAverage,
		"tips":          score.Tips,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Price Alert Handlers ───

func (ns *NegotiationServer) handleEnablePriceAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	thresholdPct := req.GetFloat("threshold_pct", 10)
	channel := req.GetString("channel", "webhook")

	ns.logger.Debug("enable_price_alert", "vendor", vendor, "sku", sku, "threshold", thresholdPct)

	rule, err := ns.priceAlertEng.EnableAlert(ctx, vendor, sku, thresholdPct, channel)
	if err != nil {
		ns.logger.Warn("enable_price_alert failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Enable price alert failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":        rule.Vendor,
		"sku":           rule.SKU,
		"threshold_pct": rule.ThresholdPct,
		"channel":       rule.Channel,
		"enabled":       rule.Enabled,
		"created_at":    rule.CreatedAt,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCheckPriceAlerts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("check_price_alerts")

	results, err := ns.priceAlertEng.CheckAlerts(ctx)
	if err != nil {
		ns.logger.Warn("check_price_alerts failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Check price alerts failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"results":     results,
		"count":       len(results),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDisablePriceAlert(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")

	ns.logger.Debug("disable_price_alert", "vendor", vendor, "sku", sku)

	if err := ns.priceAlertEng.DisableAlert(ctx, vendor, sku); err != nil {
		ns.logger.Warn("disable_price_alert failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Disable price alert failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"status":      "disabled",
		"vendor":      vendor,
		"sku":         sku,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Renewal Reminder Handlers ───

func (ns *NegotiationServer) handleCheckRenewalReminders(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("check_renewal_reminders")

	result, err := ns.reminderEng.CheckRenewals(ctx)
	if err != nil {
		ns.logger.Warn("check_renewal_reminders failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Check renewal reminders failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"critical":    result.Critical,
		"soon":        result.Soon,
		"upcoming":    result.Upcoming,
		"total":       len(result.Critical) + len(result.Soon) + len(result.Upcoming),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Budget Alert Handlers ───

func (ns *NegotiationServer) handleCheckBudgetAlerts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("check_budget_alerts")

	alerts, err := ns.budgetAlertEng.CheckBudgets(ctx)
	if err != nil {
		ns.logger.Warn("check_budget_alerts failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Check budget alerts failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"alerts":      alerts,
		"count":       len(alerts),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleBudgetAlertHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	limit := int(req.GetInt("limit", 10))

	ns.logger.Debug("budget_alert_history", "vendor", vendor, "limit", limit)

	history, err := ns.budgetAlertEng.Store().List(ctx, vendor, limit)
	if err != nil {
		ns.logger.Warn("budget_alert_history failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Budget alert history failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"vendor":      vendor,
		"history":     history,
		"count":       len(history),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Report Builder Handler ---

func (ns *NegotiationServer) handleBuildReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	rawSections, _ := req.GetArguments()["sections"]
	sectionsRaw, _ := rawSections.([]any)
	sections := make([]string, len(sectionsRaw))
	for i, v := range sectionsRaw {
		sections[i], _ = v.(string)
	}
	if len(sections) == 0 {
		return mcp.NewToolResultError("sections is required"), nil
	}

	period := req.GetString("period", "all")
	vendor := req.GetString("vendor", "")

	ns.logger.Debug("build_report called", "sections", sections, "period", period, "vendor", vendor)

	reqData := reports.ReportRequest{
		Sections: sections,
		Period:   period,
		Vendor:   vendor,
	}
	result, err := ns.reportsEng.Build(ctx, reqData)
	if err != nil {
		ns.logger.Warn("build_report failed", "error", err.Error())
		return mcp.NewToolResultError("Build report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"sections":      result.Sections,
		"generated_at":  result.GeneratedAt,
		"section_count": result.SectionCount,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Pricing Index Handler ---

func (ns *NegotiationServer) handlePricingIndex(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category := req.GetString("category", "")
	vendor := req.GetString("vendor", "")

	ns.logger.Debug("pricing_index called", "category", category, "vendor", vendor)

	result, err := ns.pricingIndexEng.Index(ctx, category, vendor)
	if err != nil {
		ns.logger.Warn("pricing_index failed", "error", err.Error())
		return mcp.NewToolResultError("Pricing index failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"category":         result.Category,
		"period":           result.Period,
		"avg_price":        result.AvgPrice,
		"price_range":      result.PriceRange,
		"vendors":          result.Vendors,
		"mom_change_pct":   result.MoMChangePct,
		"volatility_index": result.VolatilityIdx,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Price Chart Handler ---

func (ns *NegotiationServer) handlePriceChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku := req.GetString("sku", "")
	period := req.GetString("period", "1y")

	ns.logger.Debug("price_chart called", "vendor", vendor, "sku", sku, "period", period)

	result, err := ns.priceChartEng.Chart(ctx, vendor, sku, period)
	if err != nil {
		ns.logger.Warn("price_chart failed", "error", err.Error())
		return mcp.NewToolResultError("Price chart failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"labels":      result.Labels,
		"datasets":    result.Datasets,
		"summary":     result.Summary,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Vendor Comparison Handler ───

func (ns *NegotiationServer) handleCompareVendors(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category, _ := req.RequireString("category")
	rawFeatures, _ := req.GetArguments()["features"]
	var features []string
	if rawFeatures != nil {
		if arr, ok := rawFeatures.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					features = append(features, s)
				}
			}
		}
	}
	seats := int(req.GetInt("seats", 50))

	ns.logger.Debug("compare_vendors called", "category", category, "features", features, "seats", seats)

	compReq := vendorcomparison.ComparisonRequest{
		Category: category,
		Features: features,
		Seats:    seats,
	}
	result, err := ns.vendorComparisonEng.Compare(ctx, compReq)
	if err != nil {
		ns.logger.Warn("compare_vendors failed", "error", err.Error())
		return mcp.NewToolResultError("Compare vendors failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"category":    result.Category,
		"comparisons": result.Comparisons,
		"top_pick":    result.TopPick,
		"avg_price":   result.AvgPrice,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Batch Negotiation Handler ───

func (ns *NegotiationServer) handleBatchNegotiate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendorsJSON, _ := req.RequireString("vendors_json")

	ns.logger.Debug("batch_negotiate called")

	var items []batchnegotiation.BatchItem
	if err := json.Unmarshal([]byte(vendorsJSON), &items); err != nil {
		return mcp.NewToolResultError("Invalid vendors_json: " + err.Error()), nil
	}

	batchReq := batchnegotiation.BatchRequest{Items: items}
	result, err := ns.batchNegotiationEng.Run(ctx, batchReq)
	if err != nil {
		ns.logger.Warn("batch_negotiate failed", "error", err.Error())
		return mcp.NewToolResultError("Batch negotiate failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"batch_id":      result.BatchID,
		"results":       result.Results,
		"total_savings": result.TotalSavings,
		"duration_ms":   time.Since(start).Milliseconds(),
		"created_at":    result.CreatedAt,
	}
	return ns.jsonResult(resp)
}

// ─── Strategy Comparison Handler ───

func (ns *NegotiationServer) handleCompareStrategies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku := req.GetString("sku", "")
	rawStrategies, _ := req.GetArguments()["strategies"]
	var strategies []string
	if rawStrategies != nil {
		if arr, ok := rawStrategies.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					strategies = append(strategies, s)
				}
			}
		}
	}
	budget := req.GetFloat("budget", 0)

	ns.logger.Debug("compare_strategies called", "vendor", vendor, "sku", sku, "budget", budget)

	compReq := strategycomparison.StrategyComparisonRequest{
		Vendor:     vendor,
		SKU:        sku,
		Strategies: strategies,
		Budget:     budget,
	}
	result, err := ns.strategyComparisonEng.Compare(ctx, compReq)
	if err != nil {
		ns.logger.Warn("compare_strategies failed", "error", err.Error())
		return mcp.NewToolResultError("Compare strategies failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"vendor":        result.Vendor,
		"sku":           result.SKU,
		"budget":        result.Budget,
		"results":       result.Results,
		"best_strategy": result.BestStrategy,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Workspace Handlers ---

func (ns *NegotiationServer) handleCreateWorkspace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	name, _ := req.RequireString("name")
	description := req.GetString("description", "")

	ns.logger.Debug("create_workspace called", "name", name)

	ws, err := ns.workspacesEng.Create(ctx, name, description)
	if err != nil {
		ns.logger.Warn("create_workspace failed", "error", err.Error())
		return mcp.NewToolResultError("Create workspace failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          ws.ID,
		"name":        ws.Name,
		"description": ws.Description,
		"created_at":  ws.CreatedAt.Format(time.RFC3339),
		"updated_at":  ws.UpdatedAt.Format(time.RFC3339),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListWorkspaces(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("list_workspaces called")

	workspaces, err := ns.workspacesEng.List(ctx)
	if err != nil {
		ns.logger.Warn("list_workspaces failed", "error", err.Error())
		return mcp.NewToolResultError("List workspaces failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"workspaces":  workspaces,
		"count":       len(workspaces),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleWorkspaceSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	workspaceID, _ := req.RequireString("workspace_id")

	ns.logger.Debug("workspace_summary called", "workspace_id", workspaceID)

	summary, err := ns.workspacesEng.Summary(ctx, workspaceID)
	if err != nil {
		ns.logger.Warn("workspace_summary failed", "error", err.Error())
		return mcp.NewToolResultError("Workspace summary failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"workspace_id":  summary.ID,
		"name":          summary.Name,
		"vendor_count":  summary.VendorCount,
		"deal_count":    summary.DealCount,
		"total_savings": summary.TotalSavings,
		"member_count":  summary.MemberCount,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Audit Log Handlers ---

func (ns *NegotiationServer) handleAuditLog(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	action := req.GetString("action", "")
	userID := req.GetString("user_id", "")
	limit := int(req.GetInt("limit", 50))
	since := req.GetString("since", "")

	ns.logger.Debug("audit_log called", "action", action, "user_id", userID, "limit", limit)

	entries, err := ns.auditLogEng.Search(ctx, action, userID, limit, since)
	if err != nil {
		ns.logger.Warn("audit_log failed", "error", err.Error())
		return mcp.NewToolResultError("Audit log failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"entries":     entries,
		"count":       len(entries),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAuditSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("audit_summary called")

	summary, err := ns.auditLogEng.Summary(ctx)
	if err != nil {
		ns.logger.Warn("audit_summary failed", "error", err.Error())
		return mcp.NewToolResultError("Audit summary failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"total_actions": summary.TotalActions,
		"by_action":     summary.ByAction,
		"by_day":        summary.ByDay,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- User Activity Handlers ---

func (ns *NegotiationServer) handleUserActivity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	userID := req.GetString("user_id", "")
	period := req.GetString("period", "30d")

	ns.logger.Debug("user_activity called", "user_id", userID, "period", period)

	report, err := ns.userActivityEng.Report(ctx, userID, period)
	if err != nil {
		ns.logger.Warn("user_activity failed", "error", err.Error())
		return mcp.NewToolResultError("User activity failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"user_id":                report.UserID,
		"period":                 report.Period,
		"total_sessions":         report.TotalSessions,
		"completed_negotiations": report.CompletedNegotiations,
		"total_savings":          report.TotalSavings,
		"active_days":            report.ActiveDays,
		"last_active":            report.LastActive,
		"favorite_strategies":    report.FavoriteStrategies,
		"top_vendors":            report.TopVendors,
		"duration_ms":            time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Contract Template Handlers ---

func (ns *NegotiationServer) handleListContractTemplates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category := req.GetString("category", "")

	ns.logger.Debug("list_contract_templates called", "category", category)

	templates, err := ns.contractTemplatesEng.ListTemplates(ctx, category)
	if err != nil {
		ns.logger.Warn("list_contract_templates failed", "error", err.Error())
		return mcp.NewToolResultError("List contract templates failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"templates":   templates,
		"count":       len(templates),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGenerateContract(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	templateID, _ := req.RequireString("template_id")
	vendor, _ := req.RequireString("vendor")

	// Parse params map
	params := map[string]string{}
	if rawParams, ok := req.GetArguments()["params"]; ok {
		if paramsMap, ok := rawParams.(map[string]any); ok {
			for k, v := range paramsMap {
				if s, ok := v.(string); ok {
					params[k] = s
				}
			}
		}
	}

	ns.logger.Debug("generate_contract called", "template_id", templateID, "vendor", vendor)

	contract, err := ns.contractTemplatesEng.GenerateContract(ctx, templateID, vendor, params)
	if err != nil {
		ns.logger.Warn("generate_contract failed", "error", err.Error())
		return mcp.NewToolResultError("Generate contract failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"template_id":    contract.TemplateID,
		"vendor_name":    contract.VendorName,
		"content":        contract.Content,
		"variables_used": contract.VariablesUsed,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Contract Risk Handler ---

func (ns *NegotiationServer) handleContractRisk(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	contractText, _ := req.RequireString("contract_text")

	ns.logger.Debug("contract_risk called", "text_length", len(contractText))

	report := ns.contractRiskEng.Analyze(contractText)

	resp := map[string]any{
		"overall_score":   report.OverallScore,
		"risk_level":      report.RiskLevel,
		"clauses":         report.Clauses,
		"recommendations": report.Recommendations,
		"duration_ms":     time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Shared Strategy Handlers ---

func (ns *NegotiationServer) handleShareStrategy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	name, _ := req.RequireString("name")
	notes := req.GetString("notes", "")
	strategyType := req.GetString("strategy_type", "balanced")

	ns.logger.Debug("share_strategy called", "name", name, "strategy_type", strategyType)

	st, err := ns.sharedStrategiesEng.ShareStrategy(ctx, name, notes, strategyType)
	if err != nil {
		ns.logger.Warn("share_strategy failed", "error", err.Error())
		return mcp.NewToolResultError("Share strategy failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":            st.ID,
		"name":          st.Name,
		"notes":         st.Notes,
		"strategy_type": st.StrategyType,
		"usage_count":   st.UsageCount,
		"created_at":    st.CreatedAt,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListSharedStrategies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("list_shared_strategies called")

	strategies, err := ns.sharedStrategiesEng.List(ctx)
	if err != nil {
		ns.logger.Warn("list_shared_strategies failed", "error", err.Error())
		return mcp.NewToolResultError("List shared strategies failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"strategies":  strategies,
		"count":       len(strategies),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleImportStrategy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	strategyID, _ := req.RequireString("strategy_id")

	ns.logger.Debug("import_strategy called", "strategy_id", strategyID)

	st, err := ns.sharedStrategiesEng.ImportStrategy(ctx, strategyID)
	if err != nil {
		ns.logger.Warn("import_strategy failed", "error", err.Error())
		return mcp.NewToolResultError("Import strategy failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":            st.ID,
		"name":          st.Name,
		"notes":         st.Notes,
		"strategy_type": st.StrategyType,
		"usage_count":   st.UsageCount,
		"created_at":    st.CreatedAt,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Notes Handlers ---

func (ns *NegotiationServer) handleAddNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	sessionID, _ := req.RequireString("session_id")
	content, _ := req.RequireString("content")

	ns.logger.Debug("add_note called", "session_id", sessionID)

	note, err := ns.notesEng.AddNote(ctx, sessionID, content)
	if err != nil {
		ns.logger.Warn("add_note failed", "error", err.Error())
		return mcp.NewToolResultError("Add note failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          note.ID,
		"session_id":  note.SessionID,
		"content":     note.Content,
		"created_at":  note.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	sessionID, _ := req.RequireString("session_id")

	ns.logger.Debug("list_notes called", "session_id", sessionID)

	notes, err := ns.notesEng.ListNotes(ctx, sessionID)
	if err != nil {
		ns.logger.Warn("list_notes failed", "error", err.Error())
		return mcp.NewToolResultError("List notes failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"notes":       notes,
		"count":       len(notes),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDeleteNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	noteID := req.GetInt("note_id", 0)
	if noteID == 0 {
		return mcp.NewToolResultError("note_id is required"), nil
	}

	ns.logger.Debug("delete_note called", "note_id", noteID)

	if err := ns.notesEng.DeleteNote(ctx, int64(noteID)); err != nil {
		ns.logger.Warn("delete_note failed", "error", err.Error())
		return mcp.NewToolResultError("Delete note failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"deleted":     true,
		"note_id":     noteID,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Approval Handlers ---

func (ns *NegotiationServer) handleRequestApproval(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	sessionID, _ := req.RequireString("session_id")
	reason, _ := req.RequireString("reason")
	threshold := req.GetFloat("threshold", 0)

	ns.logger.Debug("request_approval called", "session_id", sessionID, "reason", reason, "threshold", threshold)

	approval, err := ns.approvalsEng.RequestApproval(ctx, sessionID, reason, threshold)
	if err != nil {
		ns.logger.Warn("request_approval failed", "error", err.Error())
		return mcp.NewToolResultError("Request approval failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          approval.ID,
		"session_id":  approval.SessionID,
		"reason":      approval.Reason,
		"threshold":   approval.Threshold,
		"status":      approval.Status,
		"created_at":  approval.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleApprove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	approvalID, _ := req.RequireString("approval_id")

	ns.logger.Debug("approve called", "approval_id", approvalID)

	approval, err := ns.approvalsEng.Approve(ctx, approvalID)
	if err != nil {
		ns.logger.Warn("approve failed", "error", err.Error())
		return mcp.NewToolResultError("Approve failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          approval.ID,
		"session_id":  approval.SessionID,
		"reason":      approval.Reason,
		"threshold":   approval.Threshold,
		"status":      approval.Status,
		"created_at":  approval.CreatedAt,
		"resolved_at": approval.ResolvedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleReject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	approvalID, _ := req.RequireString("approval_id")

	ns.logger.Debug("reject called", "approval_id", approvalID)

	approval, err := ns.approvalsEng.Reject(ctx, approvalID)
	if err != nil {
		ns.logger.Warn("reject failed", "error", err.Error())
		return mcp.NewToolResultError("Reject failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          approval.ID,
		"session_id":  approval.SessionID,
		"reason":      approval.Reason,
		"threshold":   approval.Threshold,
		"status":      approval.Status,
		"created_at":  approval.CreatedAt,
		"resolved_at": approval.ResolvedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handlePendingApprovals(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("pending_approvals called")

	approvals, err := ns.approvalsEng.Pending(ctx)
	if err != nil {
		ns.logger.Warn("pending_approvals failed", "error", err.Error())
		return mcp.NewToolResultError("Pending approvals failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"approvals":   approvals,
		"count":       len(approvals),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Vendor Scorecard Handler ---

func (ns *NegotiationServer) handleVendorScorecard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	period := req.GetString("period", "1y")

	ns.logger.Debug("vendor_scorecard called", "vendor", vendor, "period", period)

	scorecard, err := ns.scorecardsEng.Scorecard(ctx, vendor, period)
	if err != nil {
		ns.logger.Warn("vendor_scorecard failed", "error", err.Error())
		return mcp.NewToolResultError("Vendor scorecard failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"vendor":             scorecard.Vendor,
		"period":             scorecard.Period,
		"overall_score":      scorecard.OverallScore,
		"pricing_score":      scorecard.PricingScore,
		"reliability_score":  scorecard.ReliabilityScore,
		"support_score":      scorecard.SupportScore,
		"relationship_score": scorecard.RelationshipScore,
		"trend":              scorecard.Trend,
		"details":            scorecard.Details,
		"duration_ms":        time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- TCO Handler ---

func (ns *NegotiationServer) handleTCO(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	seats := req.GetInt("seats", 50)
	termMonths := req.GetInt("term_months", 12)
	implCosts := req.GetFloat("implementation_costs", 0)
	trainingCosts := req.GetFloat("training_costs", 0)
	supportCosts := req.GetFloat("support_costs", 0)

	ns.logger.Debug("tco called", "vendor", vendor, "sku", sku, "seats", seats)

	input := tco.TCOInput{
		Vendor:              vendor,
		SKU:                 sku,
		Seats:               seats,
		TermMonths:          termMonths,
		ImplementationCosts: implCosts,
		TrainingCosts:       trainingCosts,
		SupportCosts:        supportCosts,
	}

	output, err := ns.tcoEng.Calculate(ctx, input)
	if err != nil {
		ns.logger.Warn("tco failed", "error", err.Error())
		return mcp.NewToolResultError("TCO calculation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"vendor":                  output.Vendor,
		"sku":                     output.SKU,
		"seats":                   output.Seats,
		"term_months":             output.TermMonths,
		"per_unit_cost":           output.PerUnitCost,
		"annual_subscription":     output.AnnualSubscription,
		"total_1y_tco":            output.Total1YTCO,
		"total_3y_tco":            output.Total3YTCO,
		"cost_per_user_per_month": output.CostPerUserPerMonth,
		"market_avg_cupm":         output.MarketAvgCUPM,
		"savings_vs_market_pct":   output.SavingsVsMarketPct,
		"hidden_costs_flagged":    output.HiddenCostsFlagged,
		"duration_ms":             time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Data Import Handler ---

func (ns *NegotiationServer) handleImportData(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	typeStr, _ := req.RequireString("type")
	data, _ := req.RequireString("data")
	mode := req.GetString("mode", "import")
	dryRun := req.GetBool("dry_run", false)

	ns.logger.Debug("import_data called", "type", typeStr, "mode", mode, "dry_run", dryRun)

	importReq := dataimport.ImportRequest{
		Type:   dataimport.ImportType(typeStr),
		Data:   data,
		Mode:   dataimport.ImportMode(mode),
		DryRun: dryRun,
	}

	var result *dataimport.ImportResult
	var err error

	if mode == "validate" {
		result, err = ns.dataImportEng.Validate(ctx, importReq)
	} else {
		result, err = ns.dataImportEng.Import(ctx, importReq)
	}

	if err != nil {
		ns.logger.Warn("import_data failed", "error", err.Error())
		return mcp.NewToolResultError("Data import failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"valid_count":    result.ValidCount,
		"imported_count": result.ImportedCount,
		"skipped_count":  result.SkippedCount,
		"errors":         result.Errors,
		"summary":        result.Summary,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- Cost Allocation Handlers ---

func (ns *NegotiationServer) handleSetAllocation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	department, _ := req.RequireString("department")
	allocationPct := req.GetFloat("allocation_pct", 0)

	ns.logger.Debug("set_allocation called", "vendor", vendor, "department", department, "pct", allocationPct)

	allocation, err := ns.costAllocationEng.SetAllocation(ctx, vendor, department, allocationPct)
	if err != nil {
		ns.logger.Warn("set_allocation failed", "error", err.Error())
		return mcp.NewToolResultError("Set allocation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":             allocation.ID,
		"vendor":         allocation.Vendor,
		"department":     allocation.Department,
		"allocation_pct": allocation.AllocationPct,
		"created_at":     allocation.CreatedAt,
		"updated_at":     allocation.UpdatedAt,
		"status":         "saved",
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCostAllocationReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	period := req.GetString("period", "90d")

	ns.logger.Debug("cost_allocation_report called", "period", period)

	report, err := ns.costAllocationEng.Report(ctx, period)
	if err != nil {
		ns.logger.Warn("cost_allocation_report failed", "error", err.Error())
		return mcp.NewToolResultError("Cost allocation report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"period":             report.Period,
		"total_spend":        report.TotalSpend,
		"by_department":      report.ByDepartment,
		"by_vendor_per_dept": report.ByVendorDept,
		"duration_ms":        time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAlertHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	typeFilter := req.GetString("type", "all")
	vendorFilter := req.GetString("vendor", "")
	limit := int(req.GetInt("limit", 50))

	ns.logger.Debug("alert_history called", "type", typeFilter, "vendor", vendorFilter, "limit", limit)

	feed, err := ns.alertHistoryEng.GetAlerts(ctx, typeFilter, vendorFilter, limit)
	if err != nil {
		ns.logger.Warn("alert_history failed", "error", err.Error())
		return mcp.NewToolResultError("Alert history failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"entries":     feed.Entries,
		"grouped":     feed.Grouped,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSLACredit(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	service, _ := req.RequireString("service")
	monthlySpend := req.GetFloat("monthly_spend", 0)
	uptimePct := req.GetFloat("uptime_pct", 0)
	guaranteedUptime := req.GetFloat("guaranteed_uptime", 99.9)
	creditRate := req.GetFloat("credit_rate", 5)

	ns.logger.Debug("sla_credit called", "vendor", vendor, "service", service)

	if vendor == "" || service == "" {
		return mcp.NewToolResultError("vendor and service are required"), nil
	}

	input := &slacredit.SLACreditInput{
		Vendor:           vendor,
		Service:          service,
		MonthlySpend:     monthlySpend,
		UptimePct:        uptimePct,
		GuaranteedUptime: guaranteedUptime,
		CreditRate:       creditRate,
	}

	output := ns.slaCreditEng.Calculate(input)

	resp := map[string]any{
		"vendor":            output.Vendor,
		"service":           output.Service,
		"monthly_spend":     output.MonthlySpend,
		"actual_uptime":     output.ActualUptime,
		"guaranteed_uptime": output.GuaranteedUptime,
		"credit_rate":       output.CreditRate,
		"credit_amount":     output.CreditAmount,
		"eligible":          output.Eligible,
		"duration_ms":       time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleLogCommunication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	commType, _ := req.RequireString("type")
	summary, _ := req.RequireString("summary")
	detail := req.GetString("detail", "")

	ns.logger.Debug("log_communication called", "vendor", vendor, "type", commType)

	entry, err := ns.commLogEng.Log(ctx, vendor, commType, summary, detail)
	if err != nil {
		ns.logger.Warn("log_communication failed", "error", err.Error())
		return mcp.NewToolResultError("Log communication failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          entry.ID,
		"vendor":      entry.Vendor,
		"comm_type":   entry.CommType,
		"summary":     entry.Summary,
		"detail":      entry.Detail,
		"created_at":  entry.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCommunicationHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	limit := int(req.GetInt("limit", 20))

	ns.logger.Debug("communication_history called", "vendor", vendor, "limit", limit)

	result, err := ns.commLogEng.History(ctx, vendor, limit)
	if err != nil {
		ns.logger.Warn("communication_history failed", "error", err.Error())
		return mcp.NewToolResultError("Communication history failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"entries":     result.Entries,
		"total_count": result.TotalCount,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P66: Time-Limited Offer Handler ───

func (ns *NegotiationServer) handleAnalyzeOffer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, _ := req.RequireString("vendor")
	sku, _ := req.RequireString("sku")
	offerPrice := req.GetFloat("offer_price", 0)
	expiresAtStr, _ := req.RequireString("expires_at")
	currentPrice := req.GetFloat("current_price", 0)
	currentSpend := req.GetFloat("current_spend", 0)

	ns.logger.Debug("negotiate_analyze_offer called", "vendor", vendor, "sku", sku)

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return mcp.NewToolResultError("Invalid expires_at format, expected RFC3339: " + err.Error()), nil
	}

	input := &limitedoffer.OfferInput{
		Vendor:       vendor,
		SKU:          sku,
		OfferPrice:   offerPrice,
		ExpiresAt:    expiresAt,
		CurrentPrice: currentPrice,
		CurrentSpend: currentSpend,
	}

	result, err := ns.limitedOfferEng.Analyze(ctx, input)
	if err != nil {
		ns.logger.Warn("negotiate_analyze_offer failed", "error", err.Error())
		return mcp.NewToolResultError("Analyze offer failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"savings":           result.Savings,
		"days_remaining":    result.DaysRemaining,
		"urgency":           result.Urgency,
		"recommendation":    result.Recommendation,
		"vs_best_price_pct": result.VsBestPricePct,
		"duration_ms":       time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P67: Pricing Refresh Handler ───

func (ns *NegotiationServer) handleRefreshPricing(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawVendors, _ := req.GetArguments()["vendors"]
	var vendors []string
	if rawVendors != nil {
		if vl, ok := rawVendors.([]any); ok {
			for _, v := range vl {
				if s, ok := v.(string); ok {
					vendors = append(vendors, s)
				}
			}
		}
	}

	ns.logger.Debug("negotiate_refresh_pricing called", "vendors", vendors)

	result, err := ns.pricingRefreshEng.Refresh(ctx, vendors, ns.trendsEng.Store())
	if err != nil {
		ns.logger.Warn("negotiate_refresh_pricing failed", "error", err.Error())
		return mcp.NewToolResultError("Refresh pricing failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"vendors_refreshed": result.VendorsRefreshed,
		"records_updated":   result.RecordsUpdated,
		"duration_ms":       result.DurationMs,
	}
	return ns.jsonResult(resp)
}

// ─── P68: Rate Limit Dashboard Handlers ───

func (ns *NegotiationServer) handleRateLimitDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_rate_limit_dashboard called")

	status, err := ns.rateLimitDashEng.GetStatus(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_rate_limit_dashboard failed", "error", err.Error())
		return mcp.NewToolResultError("Rate limit dashboard failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"requests_this_minute": status.RequestsThisMinute,
		"requests_this_hour":   status.RequestsThisHour,
		"requests_today":       status.RequestsToday,
		"remaining_budget":     status.RemainingBudget,
		"status":               status.Status,
		"duration_ms":          time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleLogAPIRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	apiKeyID, _ := req.RequireString("api_key_id")
	endpoint, _ := req.RequireString("endpoint")

	ns.logger.Debug("negotiate_log_api_request called", "api_key_id", apiKeyID, "endpoint", endpoint)

	entry, err := ns.rateLimitDashEng.Store().LogRequest(ctx, apiKeyID, endpoint)
	if err != nil {
		ns.logger.Warn("negotiate_log_api_request failed", "error", err.Error())
		return mcp.NewToolResultError("Log API request failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          entry.ID,
		"api_key_id":  entry.APIKeyID,
		"endpoint":    entry.Endpoint,
		"timestamp":   entry.Timestamp,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P69-P71 Handlers ───

func (ns *NegotiationServer) handleAPIDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	format := req.GetString("format", "markdown")
	toolFilter := req.GetString("tool", "")

	ns.logger.Debug("negotiate_api_docs called", "format", format, "tool", toolFilter)

	if format == "json" {
		data, err := ns.apiDocsEng.GenerateJSON()
		if err != nil {
			ns.logger.Warn("negotiate_api_docs failed", "error", err.Error())
			return mcp.NewToolResultError("API docs generation failed: " + err.Error()), nil
		}
		resp := map[string]any{
			"format":        "json",
			"documentation": string(data),
			"duration_ms":   time.Since(start).Milliseconds(),
		}
		return ns.jsonResult(resp)
	}

	markdown := ns.apiDocsEng.GenerateMarkdown()
	resp := map[string]any{
		"format":        "markdown",
		"documentation": markdown,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleToolStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	period := req.GetString("period", "7d")

	ns.logger.Debug("negotiate_tool_stats called", "period", period)

	report, err := ns.toolStatsEng.GetReport(ctx, period)
	if err != nil {
		ns.logger.Warn("negotiate_tool_stats failed", "error", err.Error())
		return mcp.NewToolResultError("Tool stats failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"period":       report.Period,
		"total_calls":  report.TotalCalls,
		"unique_tools": report.UniqueTools,
		"top_tools":    report.TopTools,
		"bottom_tools": report.BottomTools,
		"duration_ms":  time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_health called")

	result := ns.healthCheckEng.Check(ctx)

	resp := map[string]any{
		"status":         result.Status,
		"database_ok":    result.DatabaseOK,
		"tool_count":     result.ToolCount,
		"db_size_bytes":  result.DBSizeBytes,
		"uptime_seconds": result.UptimeSecs,
		"started_at":     result.StartedAt,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCLIAutocomplete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	shell := req.GetString("shell", "bash")

	ns.logger.Debug("negotiate_cli_autocomplete called", "shell", shell)

	script := ns.autocompleteEng.Generate(shell)

	resp := map[string]any{
		"content":     script.Content,
		"shell":       script.Shell,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_metrics called")

	payload, err := ns.metricsEng.Generate(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_metrics failed", "error", err.Error())
		return mcp.NewToolResultError("Metrics generation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"content":     payload.Content,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleShutdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns.logger.Debug("negotiate_shutdown called")

	// Collect closable stores
	var db *sql.DB
	var stores []shutdown.Closable
	if ns.pricingStore != nil {
		db = ns.pricingStore.DB()
		stores = append(stores, ns.pricingStore)
	}

	result := ns.shutdownEng.Shutdown(db, stores)

	resp := map[string]any{
		"status":            result.Status,
		"resources_cleaned": result.ResourcesCleaned,
		"duration_ms":       result.DurationMs,
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCoverage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Debug("negotiate_coverage called")

	if ns.coverageEng == nil {
		return mcp.NewToolResultError("Coverage engine is not available"), nil
	}

	report, err := ns.coverageEng.Run()
	if err != nil {
		ns.logger.Warn("negotiate_coverage failed", "error", err.Error())
		return mcp.NewToolResultError("Coverage report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"overall_pct":       report.OverallPct,
		"packages":          report.Packages,
		"total_tests":       report.TotalTests,
		"untested_packages": report.UntestedPackages,
		"recommendation":    report.Recommendation,
		"duration_ms":       time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleDependencies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Debug("negotiate_dependencies called")

	if ns.dependencyEng == nil {
		return mcp.NewToolResultError("Dependency engine is not available"), nil
	}

	modData, err := os.ReadFile("go.mod")
	if err != nil {
		ns.logger.Warn("negotiate_dependencies failed", "error", err.Error())
		return mcp.NewToolResultError("Failed to read go.mod: " + err.Error()), nil
	}

	report, err := ns.dependencyEng.Parse(modData)
	if err != nil {
		ns.logger.Warn("negotiate_dependencies failed", "error", err.Error())
		return mcp.NewToolResultError("Dependency report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"direct":      report.Direct,
		"indirect":    report.Indirect,
		"total_count": report.TotalCount,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleContributionGuide(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Debug("negotiate_contribution_guide called")

	if ns.contribguideEng == nil {
		return mcp.NewToolResultError("Contribution guide engine is not available"), nil
	}

	// Determine project root from the working directory
	guide, err := ns.contribguideEng.Generate(".")
	if err != nil {
		ns.logger.Warn("negotiate_contribution_guide failed", "error", err.Error())
		return mcp.NewToolResultError("Contribution guide generation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"content":     guide.Content,
		"sections":    guide.Sections,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRotateKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	keyID, err := req.RequireString("key_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: key_id"), nil
	}

	ns.logger.Info("negotiate_rotate_key called", "key_id", keyID)

	if ns.apiKeyRotateEng == nil {
		return mcp.NewToolResultError("API key rotation engine is not available"), nil
	}

	result, err := ns.apiKeyRotateEng.RotateKey(keyID)
	if err != nil {
		ns.logger.Warn("negotiate_rotate_key failed", "error", err.Error())
		return mcp.NewToolResultError("Key rotation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"old_key_id":  result.OldKeyID,
		"new_key_id":  result.NewKeyID,
		"status":      result.Status,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleKeyHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Info("negotiate_key_health called")

	if ns.apiKeyRotateEng == nil {
		return mcp.NewToolResultError("API key rotation engine is not available"), nil
	}

	keys, err := ns.apiKeyRotateEng.KeyHealth()
	if err != nil {
		ns.logger.Warn("negotiate_key_health failed", "error", err.Error())
		return mcp.NewToolResultError("Key health check failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"keys":        keys,
		"total":       len(keys),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAddIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ip, _ := req.RequireString("ip_address")
	label, _ := req.RequireString("label")

	ns.logger.Info("negotiate_add_ip called", "ip_address", ip, "label", label)

	if ns.ipWhitelistEng == nil {
		return mcp.NewToolResultError("IP whitelist engine is not available"), nil
	}

	if err := ns.ipWhitelistEng.AddIP(ip, label); err != nil {
		ns.logger.Warn("negotiate_add_ip failed", "error", err.Error())
		return mcp.NewToolResultError("Add IP failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"ip_address":  ip,
		"label":       label,
		"status":      "added",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRemoveIP(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ip, _ := req.RequireString("ip_address")

	ns.logger.Info("negotiate_remove_ip called", "ip_address", ip)

	if ns.ipWhitelistEng == nil {
		return mcp.NewToolResultError("IP whitelist engine is not available"), nil
	}

	if err := ns.ipWhitelistEng.RemoveIP(ip); err != nil {
		ns.logger.Warn("negotiate_remove_ip failed", "error", err.Error())
		return mcp.NewToolResultError("Remove IP failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"ip_address":  ip,
		"status":      "removed",
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListWhitelist(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Info("negotiate_list_whitelist called")

	if ns.ipWhitelistEng == nil {
		return mcp.NewToolResultError("IP whitelist engine is not available"), nil
	}

	entries := ns.ipWhitelistEng.List()

	resp := map[string]any{
		"entries":     entries,
		"total":       len(entries),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSetRetention(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	dataType, _ := req.RequireString("data_type")
	retentionDays := req.GetInt("retention_days", 0)
	if retentionDays < 1 {
		return mcp.NewToolResultError("Missing or invalid required parameter: retention_days"), nil
	}
	action, _ := req.RequireString("action")

	ns.logger.Info("negotiate_set_retention called", "data_type", dataType, "retention_days", retentionDays, "action", action)

	if ns.dataRetentionEng == nil {
		return mcp.NewToolResultError("Data retention engine is not available"), nil
	}

	if err := ns.dataRetentionEng.SetPolicy(dataType, int(retentionDays), action); err != nil {
		ns.logger.Warn("negotiate_set_retention failed", "error", err.Error())
		return mcp.NewToolResultError("Set retention failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"data_type":      dataType,
		"retention_days": retentionDays,
		"action":         action,
		"status":         "set",
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGetRetention(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Info("negotiate_get_retention called")

	if ns.dataRetentionEng == nil {
		return mcp.NewToolResultError("Data retention engine is not available"), nil
	}

	policies := ns.dataRetentionEng.GetPolicies()

	resp := map[string]any{
		"policies":    policies,
		"total":       len(policies),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handlePurgeOldData(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	dryRun := req.GetBool("dry_run", true)

	ns.logger.Info("negotiate_purge_old_data called", "dry_run", dryRun)

	if ns.dataRetentionEng == nil {
		return mcp.NewToolResultError("Data retention engine is not available"), nil
	}

	results := ns.dataRetentionEng.PurgeOldData(dryRun)

	resp := map[string]any{
		"results":     results,
		"total_types": len(results),
		"dry_run":     dryRun,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
func (ns *NegotiationServer) handleSimulate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, err := req.RequireString("vendor")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: vendor"), nil
	}
	strategy, err := req.RequireString("strategy")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: strategy"), nil
	}
	budget, err := req.RequireFloat("budget")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: budget"), nil
	}
	rounds := req.GetInt("rounds", 3)

	ns.logger.Debug("negotiate_simulate called", "vendor", vendor, "strategy", strategy, "budget", budget, "rounds", rounds)

	result, err := ns.trainingEng.Simulate(vendor, strategy, budget, rounds)
	if err != nil {
		ns.logger.Warn("negotiate_simulate failed", "error", err.Error())
		return mcp.NewToolResultError("Simulation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":                 result.ID,
		"vendor":             result.Vendor,
		"strategy":           result.Strategy,
		"budget":             result.Budget,
		"total_rounds":       result.TotalRounds,
		"rounds":             result.Rounds,
		"final_outcome":      result.FinalOutcome,
		"total_discount_pct": result.TotalDiscount,
		"lessons":            result.Lessons,
		"duration_ms":        time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handlePlaybook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_playbook called")

	pb, err := ns.playbookEng.Generate()
	if err != nil {
		ns.logger.Warn("negotiate_playbook failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate playbook: %s", err.Error())), nil
	}

	resp := map[string]any{
		"content":     pb.Content,
		"sections":    pb.Sections,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P83: Industry Report Handlers ───

func (ns *NegotiationServer) handleSaveReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: title"), nil
	}
	category, err := req.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: category"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: content"), nil
	}
	source, err := req.RequireString("source")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: source"), nil
	}

	ns.logger.Debug("negotiate_save_report called", "title", title, "category", category, "source", source)

	report, err := ns.industryReportsStore.SaveReport(ctx, title, category, content, source)
	if err != nil {
		ns.logger.Warn("negotiate_save_report failed", "error", err.Error())
		return mcp.NewToolResultError("Save report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          report.ID,
		"title":       report.Title,
		"category":    report.Category,
		"source":      report.Source,
		"created_at":  report.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListReports(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category := req.GetString("category", "")

	ns.logger.Debug("negotiate_list_reports called", "category", category)

	reports, err := ns.industryReportsStore.ListReports(ctx, category)
	if err != nil {
		ns.logger.Warn("negotiate_list_reports failed", "error", err.Error())
		return mcp.NewToolResultError("List reports failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"reports":     reports,
		"total":       len(reports),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGetReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	reportID, err := req.RequireInt("report_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: report_id"), nil
	}

	ns.logger.Debug("negotiate_get_report called", "report_id", reportID)

	report, err := ns.industryReportsStore.GetReport(ctx, reportID)
	if err != nil {
		ns.logger.Warn("negotiate_get_report failed", "error", err.Error())
		return mcp.NewToolResultError("Get report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          report.ID,
		"title":       report.Title,
		"category":    report.Category,
		"content":     report.Content,
		"source":      report.Source,
		"created_at":  report.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P84: AI Agent Performance Handler ───

// ─── P98: Webhook Event Log Handlers ───

func (ns *NegotiationServer) handleListWebhookEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	status := req.GetString("status", "")
	limit := int(req.GetInt("limit", 50))

	ns.logger.Debug("negotiate_list_webhook_events called", "status", status, "limit", limit)

	events, err := ns.webhookLogStore.ListEvents(ctx, status, limit)
	if err != nil {
		ns.logger.Warn("negotiate_list_webhook_events failed", "error", err.Error())
		return mcp.NewToolResultError("List webhook events failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"events":      events,
		"total":       len(events),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleWebhookEventDetail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	eventID, err := req.RequireInt("event_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: event_id"), nil
	}

	ns.logger.Debug("negotiate_webhook_event_detail called", "event_id", eventID)

	event, err := ns.webhookLogStore.GetEvent(ctx, eventID)
	if err != nil {
		ns.logger.Warn("negotiate_webhook_event_detail failed", "error", err.Error())
		return mcp.NewToolResultError("Get webhook event failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          event.ID,
		"event_type":  event.EventType,
		"payload":     event.Payload,
		"status":      event.Status,
		"attempts":    event.Attempts,
		"created_at":  event.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleReplayWebhookEvent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	eventID, err := req.RequireInt("event_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: event_id"), nil
	}

	ns.logger.Debug("negotiate_replay_webhook_event called", "event_id", eventID)

	event, err := ns.webhookLogStore.ReplayEvent(ctx, eventID)
	if err != nil {
		ns.logger.Warn("negotiate_replay_webhook_event failed", "error", err.Error())
		return mcp.NewToolResultError("Replay webhook event failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          event.ID,
		"event_type":  event.EventType,
		"payload":     event.Payload,
		"status":      event.Status,
		"attempts":    event.Attempts,
		"created_at":  event.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleWebhookStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_webhook_stats called")

	stats, err := ns.webhookLogStore.GetStats(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_webhook_stats failed", "error", err.Error())
		return mcp.NewToolResultError("Get webhook stats failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"total_events":     stats.TotalEvents,
		"success_rate":     stats.SuccessRate,
		"avg_attempts":     stats.AvgAttempts,
		"status_breakdown": stats.StatusBreakdown,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
func (ns *NegotiationServer) handleAIPerformance(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	model, err := req.RequireString("model")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: model"), nil
	}
	toolName, err := req.RequireString("tool_name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: tool_name"), nil
	}
	latencyMs, err := req.RequireFloat("latency_ms")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: latency_ms"), nil
	}
	tokensUsed, err := req.RequireFloat("tokens_used")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: tokens_used"), nil
	}
	success, err := req.RequireBool("success")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: success"), nil
	}
	negotiationType := req.GetString("negotiation_type", "")

	ns.logger.Debug("negotiate_ai_performance called",
		"model", model,
		"tool_name", toolName,
		"latency_ms", latencyMs,
		"tokens_used", tokensUsed,
		"success", success,
		"negotiation_type", negotiationType,
	)

	entry, err := ns.aiPerfStore.LogCall(ctx, model, toolName, int(latencyMs), int(tokensUsed), success, negotiationType)
	if err != nil {
		ns.logger.Warn("negotiate_ai_performance failed", "error", err.Error())
		return mcp.NewToolResultError("Log AI performance failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":               entry.ID,
		"model":            entry.Model,
		"tool_name":        entry.ToolName,
		"latency_ms":       entry.LatencyMs,
		"tokens_used":      entry.TokensUsed,
		"success":          entry.Success,
		"negotiation_type": entry.NegotiationType,
		"created_at":       entry.CreatedAt,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P85: Prompt Template Handlers ───

func (ns *NegotiationServer) handleSavePrompt(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: name"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: content"), nil
	}
	tags := req.GetString("tags", "")

	ns.logger.Debug("negotiate_save_prompt called", "name", name, "tags", tags)

	prompt, err := ns.promptsStore.SavePrompt(ctx, name, content, tags)
	if err != nil {
		ns.logger.Warn("negotiate_save_prompt failed", "error", err.Error())
		return mcp.NewToolResultError("Save prompt failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          prompt.ID,
		"name":        prompt.Name,
		"tags":        prompt.Tags,
		"created_at":  prompt.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListPrompts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	tag := req.GetString("tag", "")

	ns.logger.Debug("negotiate_list_prompts called", "tag", tag)

	prompts, err := ns.promptsStore.ListPrompts(ctx, tag)
	if err != nil {
		ns.logger.Warn("negotiate_list_prompts failed", "error", err.Error())
		return mcp.NewToolResultError("List prompts failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"prompts":     prompts,
		"total":       len(prompts),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGetPrompt(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	promptID, err := req.RequireInt("prompt_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: prompt_id"), nil
	}

	ns.logger.Debug("negotiate_get_prompt called", "prompt_id", promptID)

	prompt, err := ns.promptsStore.GetPrompt(ctx, promptID)
	if err != nil {
		ns.logger.Warn("negotiate_get_prompt failed", "error", err.Error())
		return mcp.NewToolResultError("Get prompt failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          prompt.ID,
		"name":        prompt.Name,
		"content":     prompt.Content,
		"tags":        prompt.Tags,
		"created_at":  prompt.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRenderPrompt(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	promptID, err := req.RequireInt("prompt_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: prompt_id"), nil
	}
	variablesJSON := req.GetString("variables", "")

	ns.logger.Debug("negotiate_render_prompt called", "prompt_id", promptID)

	variables := map[string]string{}
	if variablesJSON != "" {
		if err := json.Unmarshal([]byte(variablesJSON), &variables); err != nil {
			return mcp.NewToolResultError("Invalid variables JSON: " + err.Error()), nil
		}
	}

	rendered, err := ns.promptsStore.RenderPrompt(ctx, promptID, variables)
	if err != nil {
		ns.logger.Warn("negotiate_render_prompt failed", "error", err.Error())
		return mcp.NewToolResultError("Render prompt failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"prompt_id":   promptID,
		"rendered":    rendered,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleModelABTest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	modelA, err := req.RequireString("model_a")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: model_a"), nil
	}
	modelB, err := req.RequireString("model_b")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: model_b"), nil
	}
	scenarioID, err := req.RequireString("scenario_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: scenario_id"), nil
	}

	ns.logger.Debug("negotiate_model_ab_test called", "model_a", modelA, "model_b", modelB, "scenario_id", scenarioID)

	result, err := ns.modelABTestEng.RunABTest(ctx, modelabtesting.ABTestInput{
		ModelA:     modelA,
		ModelB:     modelB,
		ScenarioID: scenarioID,
	})
	if err != nil {
		ns.logger.Warn("negotiate_model_ab_test failed", "error", err.Error())
		return mcp.NewToolResultError("Model A/B test failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"model_a_result":          result.ModelAResult,
		"model_b_result":          result.ModelBResult,
		"winner":                 result.Winner,
		"savings_difference_pct": result.SavingsDiff,
		"recommendation":         result.Recommendation,
		"duration_ms":            time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P87: Vendor Knowledge Base Handlers ───

func (ns *NegotiationServer) handleIngestDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, err := req.RequireString("vendor")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: vendor"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: content"), nil
	}
	docType, err := req.RequireString("doc_type")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: doc_type"), nil
	}

	ns.logger.Debug("negotiate_ingest_document called", "vendor", vendor, "doc_type", docType)

	doc, err := ns.vendorKnowledgeStore.IngestDocument(ctx, vendor, content, docType)
	if err != nil {
		ns.logger.Warn("negotiate_ingest_document failed", "error", err.Error())
		return mcp.NewToolResultError("Ingest document failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          doc.ID,
		"vendor":      doc.Vendor,
		"doc_type":    doc.DocType,
		"created_at":  doc.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSearchVendorDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, err := req.RequireString("vendor")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: vendor"), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: query"), nil
	}

	ns.logger.Debug("negotiate_search_vendor_docs called", "vendor", vendor, "query", query)

	docs, err := ns.vendorKnowledgeStore.SearchDocs(ctx, vendor, query)
	if err != nil {
		ns.logger.Warn("negotiate_search_vendor_docs failed", "error", err.Error())
		return mcp.NewToolResultError("Search vendor docs failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"docs":        docs,
		"total":       len(docs),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleVendorKnowledgeReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, err := req.RequireString("vendor")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: vendor"), nil
	}

	ns.logger.Debug("negotiate_vendor_knowledge_report called", "vendor", vendor)

	report, err := ns.vendorKnowledgeStore.GetKnowledgeReport(ctx, vendor)
	if err != nil {
		ns.logger.Warn("negotiate_vendor_knowledge_report failed", "error", err.Error())
		return mcp.NewToolResultError("Vendor knowledge report failed: " + err.Error()), nil
	}

	report["duration_ms"] = time.Since(start).Milliseconds()
	return ns.jsonResult(report)
}

// ─── P89: Sentiment Analysis Handler ───

func (ns *NegotiationServer) handleSentiment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: text"), nil
	}

	ns.logger.Debug("negotiate_sentiment called", "text_length", len(text))

	if ns.sentimentEng == nil {
		return mcp.NewToolResultError("Sentiment engine is not available"), nil
	}

	result, err := ns.sentimentEng.Analyze(ctx, text)
	if err != nil {
		ns.logger.Warn("negotiate_sentiment failed", "error", err.Error())
		return mcp.NewToolResultError("Sentiment analysis failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"score":       result.Score,
		"confidence":  result.Confidence,
		"label":       result.Label,
		"key_phrases": result.KeyPhrases,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
// ─── P88: Summarize Session Handler ───

func (ns *NegotiationServer) handleSummarizeSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: session_id"), nil
	}
	style := req.GetString("style", "bullet_points")

	ns.logger.Debug("negotiate_summarize_session called", "session_id", sessionID, "style", style)

	if ns.summarizerEng == nil {
		return mcp.NewToolResultError("Summarizer engine is not available"), nil
	}

	result, err := ns.summarizerEng.Summarize(ctx, sessionID, style)
	if err != nil {
		ns.logger.Warn("negotiate_summarize_session failed", "error", err.Error())
		return mcp.NewToolResultError("Summarize session failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"session_id": result.SessionID,
		"summary":    result.Summary,
		"word_count": result.WordCount,
		"style":      result.Style,
		"key_points": result.KeyPoints,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P90: Translation Handlers ───

func (ns *NegotiationServer) handleTranslate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: text"), nil
	}
	targetLang, err := req.RequireString("target_language")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: target_language"), nil
	}

	ns.logger.Debug("negotiate_translate called", "text_length", len(text), "target_language", targetLang)

	if ns.translationEng == nil {
		return mcp.NewToolResultError("Translation engine is not available"), nil
	}

	result, err := ns.translationEng.Translate(ctx, text, targetLang)
	if err != nil {
		ns.logger.Warn("negotiate_translate failed", "error", err.Error())
		return mcp.NewToolResultError("Translation failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"original_text":     result.OriginalText,
		"translated_text":   result.TranslatedText,
		"target_language":   result.TargetLanguage,
		"detected_language": result.DetectedLanguage,
		"duration_ms":       time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSetLanguagePreference(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	vendor, err := req.RequireString("vendor")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: vendor"), nil
	}
	language, err := req.RequireString("language")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: language"), nil
	}

	ns.logger.Debug("negotiate_set_language_preference called", "vendor", vendor, "language", language)

	if ns.translationStore == nil {
		return mcp.NewToolResultError("Translation store is not available"), nil
	}

	pref, err := ns.translationStore.SetPreference(ctx, vendor, language)
	if err != nil {
		ns.logger.Warn("negotiate_set_language_preference failed", "error", err.Error())
		return mcp.NewToolResultError("Failed to set language preference: " + err.Error()), nil
	}

	resp := map[string]any{
		"vendor":   pref.Vendor,
		"language": pref.Language,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleLanguageGlossary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	termsRaw, err := req.RequireString("terms")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: terms"), nil
	}
	fromLang, err := req.RequireString("from")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: from"), nil
	}
	toLang, err := req.RequireString("to")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: to"), nil
	}

	ns.logger.Debug("negotiate_language_glossary called", "from", fromLang, "to", toLang)

	if ns.translationEng == nil {
		return mcp.NewToolResultError("Translation engine is not available"), nil
	}

	var terms []string
	if err := json.Unmarshal([]byte(termsRaw), &terms); err != nil {
		return mcp.NewToolResultError("Invalid terms JSON: " + err.Error()), nil
	}

	glossary, err := ns.translationEng.BuildGlossary(ctx, terms, fromLang, toLang)
	if err != nil {
		ns.logger.Warn("negotiate_language_glossary failed", "error", err.Error())
		return mcp.NewToolResultError("Failed to build glossary: " + err.Error()), nil
	}

	resp := map[string]any{
		"from_language": glossary.FromLanguage,
		"to_language":   glossary.ToLanguage,
		"entries":       glossary.Entries,
		"duration_ms":   time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleComplianceCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	terms, err := req.RequireString("terms")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: terms"), nil
	}
	jurisdiction, err := req.RequireString("jurisdiction")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: jurisdiction"), nil
	}

	if ns.complianceEng == nil {
		return mcp.NewToolResultError("Compliance engine is not available"), nil
	}

	result, err := ns.complianceEng.Check(ctx, terms, jurisdiction)
	if err != nil {
		ns.logger.Warn("negotiate_compliance_check failed", "error", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("Compliance check failed: %s", err.Error())), nil
	}

	resp := map[string]any{
		"terms":          result.Terms,
		"jurisdiction":   result.Jurisdiction,
		"overall_status": result.OverallStatus,
		"flags":          result.Flags,
		"pass_count":     result.PassCount,
		"flag_count":     result.FlagCount,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P92: Contract Clause Handlers ───

func (ns *NegotiationServer) handleListClauses(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category := req.GetString("category", "")

	ns.logger.Debug("negotiate_list_clauses called", "category", category)

	clauses, err := ns.clausesStore.ListClauses(ctx, category)
	if err != nil {
		ns.logger.Warn("negotiate_list_clauses failed", "error", err.Error())
		return mcp.NewToolResultError("List clauses failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"clauses":     clauses,
		"total":       len(clauses),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleGetClause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	clauseID, err := req.RequireInt("clause_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: clause_id"), nil
	}

	ns.logger.Debug("negotiate_get_clause called", "clause_id", clauseID)

	clause, err := ns.clausesStore.GetClause(ctx, clauseID)
	if err != nil {
		ns.logger.Warn("negotiate_get_clause failed", "error", err.Error())
		return mcp.NewToolResultError("Get clause failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          clause.ID,
		"category":    clause.Category,
		"title":       clause.Title,
		"content":     clause.Content,
		"description": clause.Description,
		"created_at":  clause.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSearchClauses(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: query"), nil
	}

	ns.logger.Debug("negotiate_search_clauses called", "query", query)

	clauses, err := ns.clausesStore.SearchClauses(ctx, query)
	if err != nil {
		ns.logger.Warn("negotiate_search_clauses failed", "error", err.Error())
		return mcp.NewToolResultError("Search clauses failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"clauses":     clauses,
		"total":       len(clauses),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleAddClause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category, err := req.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: category"), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: title"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: content"), nil
	}
	description := req.GetString("description", "")

	ns.logger.Debug("negotiate_add_clause called", "category", category, "title", title)

	clause, err := ns.clausesStore.AddClause(ctx, category, title, content, description)
	if err != nil {
		ns.logger.Warn("negotiate_add_clause failed", "error", err.Error())
		return mcp.NewToolResultError("Add clause failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          clause.ID,
		"category":    clause.Category,
		"title":       clause.Title,
		"content":     clause.Content,
		"description": clause.Description,
		"created_at":  clause.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P93: E-Signature Handlers ───

func (ns *NegotiationServer) handleSendForSignature(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	contractID, err := req.RequireString("contract_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: contract_id"), nil
	}
	signerEmail, err := req.RequireString("signer_email")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: signer_email"), nil
	}

	ns.logger.Debug("negotiate_send_for_signature called", "contract_id", contractID, "signer_email", signerEmail)

	env, err := ns.esigStore.CreateEnvelope(ctx, contractID, signerEmail)
	if err != nil {
		ns.logger.Warn("negotiate_send_for_signature failed", "error", err.Error())
		return mcp.NewToolResultError("Send for signature failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          env.ID,
		"contract_id": env.ContractID,
		"signer_email": env.SignerEmail,
		"status":      env.Status,
		"envelope_id": env.EnvelopeID,
		"created_at":  env.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSignatureStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	envelopeID, err := req.RequireString("envelope_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: envelope_id"), nil
	}

	ns.logger.Debug("negotiate_signature_status called", "envelope_id", envelopeID)

	env, err := ns.esigStore.GetEnvelope(ctx, envelopeID)
	if err != nil {
		ns.logger.Warn("negotiate_signature_status failed", "error", err.Error())
		return mcp.NewToolResultError("Get signature status failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"envelope_id": env.EnvelopeID,
		"contract_id": env.ContractID,
		"status":     env.Status,
		"signer_email": env.SignerEmail,
		"created_at": env.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleSignedDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	envelopeID, err := req.RequireString("envelope_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: envelope_id"), nil
	}

	ns.logger.Debug("negotiate_signed_document called", "envelope_id", envelopeID)

	env, err := ns.esigStore.GetSignedDocument(ctx, envelopeID)
	if err != nil {
		ns.logger.Warn("negotiate_signed_document failed", "error", err.Error())
		return mcp.NewToolResultError("Get signed document failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"envelope_id": env.EnvelopeID,
		"contract_id": env.ContractID,
		"status":     env.Status,
		"signer_email": env.SignerEmail,
		"created_at": env.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// --- P94: Data Residency Handlers ---

func (ns *NegotiationServer) handleSetDataResidency(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()

        region, err := req.RequireString("region")
        if err != nil {
                return mcp.NewToolResultError("Missing required parameter: region"), nil
        }
        allowed, err := req.RequireBool("allowed")
        if err != nil {
                return mcp.NewToolResultError("Missing required parameter: allowed"), nil
        }

        ns.logger.Debug("negotiate_set_data_residency called", "region", region, "allowed", allowed)

        if ns.residencyStore == nil {
                return mcp.NewToolResultError("Data residency store is not available"), nil
        }

        rule, err := ns.residencyStore.SetRule(ctx, region, allowed)
        if err != nil {
                ns.logger.Warn("negotiate_set_data_residency failed", "error", err.Error())
                return mcp.NewToolResultError("Set data residency failed: " + err.Error()), nil
        }

        resp := map[string]any{
                "id":          rule.ID,
                "region":      rule.Region,
                "allowed":     rule.Allowed,
                "updated_at":  rule.UpdatedAt,
                "duration_ms": time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleCheckResidency(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()

        vendor, err := req.RequireString("vendor")
        if err != nil {
                return mcp.NewToolResultError("Missing required parameter: vendor"), nil
        }

        ns.logger.Debug("negotiate_check_residency called", "vendor", vendor)

        if ns.residencyStore == nil {
                return mcp.NewToolResultError("Data residency store is not available"), nil
        }

        // Get all rules to check each region
        rules, err := ns.residencyStore.ListRules(ctx)
        if err != nil {
                ns.logger.Warn("negotiate_check_residency list failed", "error", err.Error())
                return mcp.NewToolResultError("Check residency failed: " + err.Error()), nil
        }

        var checks []map[string]any
        for _, rule := range rules {
                check, err := ns.residencyStore.CheckVendor(ctx, vendor, rule.Region)
                if err != nil {
                        ns.logger.Warn("negotiate_check_residency check failed", "vendor", vendor, "region", rule.Region, "error", err.Error())
                        continue
                }
                checks = append(checks, map[string]any{
                        "region":     check.Region,
                        "compliant":  check.Compliant,
                        "rule_found": check.RuleFound,
                })
        }

        resp := map[string]any{
                "vendor":     vendor,
                "checks":     checks,
                "total":      len(checks),
                "duration_ms": time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleResidencyReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        start := time.Now()

        ns.logger.Debug("negotiate_residency_report called")

        if ns.residencyStore == nil {
                return mcp.NewToolResultError("Data residency store is not available"), nil
        }

        rules, err := ns.residencyStore.ListRules(ctx)
        if err != nil {
                ns.logger.Warn("negotiate_residency_report failed", "error", err.Error())
                return mcp.NewToolResultError("Residency report failed: " + err.Error()), nil
        }

        var allowedRegions []string
        var blockedRegions []string
        for _, rule := range rules {
                if rule.Allowed {
                        allowedRegions = append(allowedRegions, rule.Region)
                } else {
                        blockedRegions = append(blockedRegions, rule.Region)
                }
        }

        resp := map[string]any{
                "rules":            rules,
                "allowed_regions":  allowedRegions,
                "blocked_regions":  blockedRegions,
                "total_rules":      len(rules),
                "duration_ms":      time.Since(start).Milliseconds(),
        }
        return ns.jsonResult(resp)
}

// ─── P95: Dashboard Widget Handlers ───

func (ns *NegotiationServer) handleCreateWidget(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	widgetType, err := req.RequireString("widget_type")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: widget_type"), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: title"), nil
	}
	config := req.GetString("config", "{}")

	ns.logger.Debug("negotiate_create_widget called", "widget_type", widgetType, "title", title)

	if ns.dashboardStore == nil {
		return mcp.NewToolResultError("Dashboard store is not available"), nil
	}

	widget, err := ns.dashboardStore.CreateWidget(ctx, widgetType, title, config)
	if err != nil {
		ns.logger.Warn("negotiate_create_widget failed", "error", err.Error())
		return mcp.NewToolResultError("Create widget failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"id":          widget.ID,
		"widget_type": widget.WidgetType,
		"title":       widget.Title,
		"config":      widget.Config,
		"created_at":  widget.CreatedAt,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleListWidgets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_list_widgets called")

	if ns.dashboardStore == nil {
		return mcp.NewToolResultError("Dashboard store is not available"), nil
	}

	widgets, err := ns.dashboardStore.ListWidgets(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_list_widgets failed", "error", err.Error())
		return mcp.NewToolResultError("List widgets failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"widgets":     widgets,
		"total":       len(widgets),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleRenderDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	widgetIDsRaw := req.GetString("widget_ids", "")
	ns.logger.Debug("negotiate_render_dashboard called", "widget_ids", widgetIDsRaw)

	if ns.dashboardStore == nil {
		return mcp.NewToolResultError("Dashboard store is not available"), nil
	}

	var widgetIDs []int
	if widgetIDsRaw != "" {
		if err := json.Unmarshal([]byte(widgetIDsRaw), &widgetIDs); err != nil {
			return mcp.NewToolResultError("Invalid widget_ids JSON: " + err.Error()), nil
		}
	}

	dash, err := ns.dashboardStore.RenderDashboard(ctx, widgetIDs)
	if err != nil {
		ns.logger.Warn("negotiate_render_dashboard failed", "error", err.Error())
		return mcp.NewToolResultError("Render dashboard failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"widgets":     dash.Widgets,
		"count":       dash.Count,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleExportDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	format := req.GetString("format", "json")
	ns.logger.Debug("negotiate_export_dashboard called", "format", format)

	if ns.dashboardStore == nil {
		return mcp.NewToolResultError("Dashboard store is not available"), nil
	}

	exported, err := ns.dashboardStore.ExportDashboard(ctx, format)
	if err != nil {
		ns.logger.Warn("negotiate_export_dashboard failed", "error", err.Error())
		return mcp.NewToolResultError("Export dashboard failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"data":        exported,
		"format":      format,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Handler: negotiate_export_chart ───

func (ns *NegotiationServer) handleExportChart(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	dataSource, err := req.RequireString("data_source")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: data_source"), nil
	}
	chartType, err := req.RequireString("chart_type")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: chart_type"), nil
	}
	format, err := req.RequireString("format")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: format"), nil
	}

	ns.logger.Debug("negotiate_export_chart called", "data_source", dataSource, "chart_type", chartType, "format", format)

	if ns.chartExportEng == nil {
		return mcp.NewToolResultError("Chart export engine is not available"), nil
	}

	result, err := ns.chartExportEng.ExportChart(ctx, dataSource, chartType, format)
	if err != nil {
		ns.logger.Warn("negotiate_export_chart failed", "error", err.Error())
		return mcp.NewToolResultError("Chart export failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"format":    result.Format,
		"chart_type": result.ChartType,
		"data":      result.Data,
		"width":     result.Width,
		"height":    result.Height,
		"mime_type": result.MimeType,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── Handler: negotiate_chart_templates ───

func (ns *NegotiationServer) handleChartTemplates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	ns.logger.Debug("negotiate_chart_templates called")

	if ns.chartExportEng == nil {
		return mcp.NewToolResultError("Chart export engine is not available"), nil
	}

	templates, err := ns.chartExportEng.ListTemplates(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_chart_templates failed", "error", err.Error())
		return mcp.NewToolResultError("Failed to list chart templates: " + err.Error()), nil
	}

	resp := map[string]any{
		"templates":   templates,
		"count":       len(templates),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P97: Real-Time Monitoring Dashboard ───

func (ns *NegotiationServer) handleLiveDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_live_dashboard called")

	if ns.monitorDashEng == nil {
		return mcp.NewToolResultError("Monitoring dashboard engine is not available"), nil
	}

	dash, err := ns.monitorDashEng.GetDashboard(ctx)
	if err != nil {
		ns.logger.Warn("negotiate_live_dashboard failed", "error", err.Error())
		return mcp.NewToolResultError("Failed to get live dashboard: " + err.Error()), nil
	}

	resp := map[string]any{
		"active_negotiations": dash.ActiveNegotiations,
		"system_health":       dash.SystemHealth,
		"last_tool_calls":     dash.LastToolCalls,
		"error_rate_5min":     dash.ErrorRate5Min,
		"uptime_seconds":      dash.UptimeSeconds,
		"total_tools":         dash.TotalTools,
		"timestamp":           dash.Timestamp,
		"duration_ms":         time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

// ─── P99: Tool Usage Billing Handlers ───

func (ns *NegotiationServer) handleSetToolPrice(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	toolName, err := req.RequireString("tool_name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: tool_name"), nil
	}
	rawPrice, ok := req.GetArguments()["price_per_call"]
	if !ok {
		return mcp.NewToolResultError("Missing required parameter: price_per_call"), nil
	}
	pricePerCall, ok := rawPrice.(float64)
	if !ok {
		return mcp.NewToolResultError("price_per_call must be a number"), nil
	}

	ns.logger.Debug("negotiate_set_tool_price called", "tool_name", toolName, "price_per_call", pricePerCall)

	tp, err := ns.toolBillingStore.SetToolPrice(ctx, toolName, pricePerCall)
	if err != nil {
		ns.logger.Warn("negotiate_set_tool_price failed", "error", err.Error())
		return mcp.NewToolResultError("Set tool price failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"tool_name":      tp.ToolName,
		"price_per_call": tp.PricePerCall,
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleBillingReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	keyID, err := req.RequireString("key_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: key_id"), nil
	}
	from, err := req.RequireString("from")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: from"), nil
	}
	to, err := req.RequireString("to")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: to"), nil
	}

	ns.logger.Debug("negotiate_billing_report called", "key_id", keyID, "from", from, "to", to)

	report, err := ns.toolBillingStore.GetBillingReport(ctx, keyID, from, to)
	if err != nil {
		ns.logger.Warn("negotiate_billing_report failed", "error", err.Error())
		return mcp.NewToolResultError("Get billing report failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"key_id":       report.KeyID,
		"total_calls":  report.TotalCalls,
		"total_cost":   report.TotalCost,
		"period_from":  report.PeriodFrom,
		"period_to":    report.PeriodTo,
		"per_tool":     report.PerTool,
		"duration_ms":  time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleUsageTier(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	keyID, err := req.RequireString("key_id")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: key_id"), nil
	}

	ns.logger.Debug("negotiate_usage_tier called", "key_id", keyID)

	tier, err := ns.toolBillingStore.GetUsageTier(ctx, keyID)
	if err != nil {
		ns.logger.Warn("negotiate_usage_tier failed", "error", err.Error())
		return mcp.NewToolResultError("Get usage tier failed: " + err.Error()), nil
	}

	resp := map[string]any{
		"key_id":           tier.KeyID,
		"current_tier":     tier.CurrentTier,
		"calls_this_month": tier.CallsThisMonth,
		"tier_limit":       tier.TierLimit,
		"duration_ms":      time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}

func (ns *NegotiationServer) handleOverageAlerts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	ns.logger.Debug("negotiate_overage_alerts called")

	// Return current tier configuration
	resp := map[string]any{
		"tiers": []map[string]any{
			{"name": "tier1", "range": "0-100 calls", "cost_per_call": "free"},
			{"name": "tier2", "range": "101-1000 calls", "cost_per_call": "$0.01"},
			{"name": "tier3", "range": "1001+ calls", "cost_per_call": "$0.005"},
		},
		"duration_ms": time.Since(start).Milliseconds(),
	}
	return ns.jsonResult(resp)
}
