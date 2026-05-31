package main

import (
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/datresidency"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dashboard"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sandbox"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/a2a"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/aiperformance"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/modelabtesting"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/alerthistory"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/apidocs"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/apikeyrotation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/approvals"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/auditlog"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/autocomplete"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/batchnegotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budget"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budgetalerts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/budgetmgmt"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/commlog"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/compliance"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/chartexport"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/monitordash"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contractclauses"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contractrisk"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contracttemplates"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/contribguide"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/costallocation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/coverage"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dataimport"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/esignature"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dataretention"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/dependency"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/effectiveness"
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
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/limitedoffer"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/marketplace"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/metrics"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/notes"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/notify"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/playbook"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricealerts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricechart"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricingindex"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricingrefresh"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/prompts"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ratelimitdashboard"
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
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/workspaces"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	var (
		dbPath       = flag.String("db", filepath.Join(dataDir(), "negotiations.db"), "Path to SQLite database")
		seedPath     = flag.String("seed", "", "Path to seed CSV file (optional)")
		logFormat    = flag.String("log", "json", "Log format: json or text")
		httpAddr     = flag.String("http", "", "HTTP listen address for A2A endpoints (e.g. :8080)")
		slackWebhook = flag.String("slack-webhook", "", "Slack webhook URL for negotiation alerts (optional)")
		apiKeysFile  = flag.String("api-keys", "", "Path to JSON file of API keys (optional, enables auth)")
		rateLimit    = flag.Int("rate-limit", 0, "Max requests per minute per key (0 = unlimited)")
	)
	flag.Parse()

	// Logger setup
	var logger *slog.Logger
	switch *logFormat {
	case "text":
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	slog.SetDefault(logger)

	// Initialize API key store (optional)
	var apiKeyStore *a2a.APIKeyStore
	if *apiKeysFile != "" {
		apiKeyStore = a2a.NewAPIKeyStore()
		if err := apiKeyStore.LoadFromFile(*apiKeysFile); err != nil {
			logger.Error("failed to load API keys", "path", *apiKeysFile, "error", err.Error())
			os.Exit(1)
		}
		logger.Info("API key authentication enabled", "path", *apiKeysFile)
	}

	// Initialize rate limiter (optional)
	var rateLimiter *a2a.RateLimiter
	if *rateLimit > 0 {
		rateLimiter = a2a.NewRateLimiter(*rateLimit, 1*time.Minute)
		logger.Info("rate limiting enabled", "rate_per_minute", *rateLimit)
	}

	logger.Info("starting a2a-negotiation-mcp server",
		"version", "1.0.0",
		"db_path", *dbPath,
	)

	// Initialize pricing store
	pricingStore, err := pricing.NewStore(*dbPath)
	if err != nil {
		logger.Error("failed to initialize pricing store", "error", err.Error())
		os.Exit(1)
	}
	defer pricingStore.Close()

	// Seed data if CSV provided
	if *seedPath != "" {
		logger.Info("seeding pricing data", "path", *seedPath)
		if err := pricingStore.SeedFromCSV(context.Background(), *seedPath); err != nil {
			logger.Error("failed to seed pricing data", "error", err.Error())
			os.Exit(1)
		}
		logger.Info("pricing data seeded successfully")
	}

	// Initialize history store (shares the same DB)
	historyStore, err := history.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize history store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize mandate store (shares the same DB)
	mandateStore, err := a2a.NewMandateStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize mandate store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize group store (shares the same DB) and engine
	groupStore, err := group.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize group store", "error", err.Error())
		os.Exit(1)
	}
	groupEngine := group.NewEngine(groupStore, pricingStore, logger)

	// Initialize sell store and engine
	sellStore, err := sell.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize sell store", "error", err.Error())
		os.Exit(1)
	}
	sellEngine := sell.NewEngine(sellStore, logger)

	// Initialize negotiation engine for cross-package use
	negEng := negotiation.NewEngine(pricingStore)

	// Initialize industry reports store (shares the same DB)
	// Initialize calendar store and engine
	calendarStore, err := calendar.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize calendar store", "error", err.Error())
		os.Exit(1)
	}
	calendarEngine := calendar.NewEngine(calendarStore, negEng, historyStore, pricingStore, logger)

	// Initialize marketplace store and engine
	marketplaceStore, err := marketplace.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize marketplace store", "error", err.Error())
		os.Exit(1)
	}
	marketplaceEngine := marketplace.NewEngine(marketplaceStore, logger)
	// Initialize health store (shares the same DB)
	healthStore, err := health.NewStoreFromDB(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize health store", "error", err.Error())
		os.Exit(1)
	}
	healthEngine := health.NewEngine(healthStore, logger)

	// Initialize SLA store and engine
	slaStore, err := sla.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize sla store", "error", err.Error())
		os.Exit(1)
	}
	slaEngine := sla.NewEngine(slaStore, logger)

	// Initialize ROI store (shares the same DB)
	roiStore, err := roi.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize roi store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize trends store (shares the same DB)
	trendsStore, err := trends.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize trends store", "error", err.Error())
		os.Exit(1)
	}

	// Seed price_snapshots for trend analysis (first-run only)
	// This generates 12 months of mock price data for trend analysis tools
	if *seedPath != "" {
		logger.Info("seeding price snapshot data", "path", *seedPath)
		seedSnapshotsFromCSV(context.Background(), trendsStore, *seedPath, logger)
	}

	// Initialize Slack client if webhook provided
	var slackClient *slack.Client
	if *slackWebhook != "" {
		slackClient = slack.NewClient(*slackWebhook, logger)
		logger.Info("slack integration enabled", "webhook", *slackWebhook)
	}

	// Initialize webhook engine
	webhookStore, err := webhooks.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize webhook store", "error", err.Error())
		os.Exit(1)
	}
	webhookEng := webhooks.NewEngine(webhookStore, logger)

	// Initialize export store (shares the same DB)
	exportStore, err := export.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize export store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize notification store (shares the same DB)
	notifyStore, err := notify.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize notification store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize budget store (shares the same DB)
	budgetStore, err := budget.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize budget store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize vendorspend engine (read-only, uses deal_outcomes)
	vendorspendEng := vendorspend.NewEngine(pricingStore.DB(), logger)

	// Initialize effectiveness engine (read-only, uses deal_outcomes + user_streaks)
	effectivenessEng := effectiveness.NewEngine(pricingStore.DB(), logger)

	// Initialize price alert store and engine
	priceAlertStore, err := pricealerts.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize price alert store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize reminder engine (read-only, uses calendar store)
	// built inline in NewNegotiationServer

	// Initialize budget alert store and engine
	budgetAlertStore, err := budgetalerts.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize budget alert store", "error", err.Error())
		os.Exit(1)
	}

	// Create MCP negotiation server

	// Initialize vendor comparison, batch negotiation, and strategy comparison engines
	vendorComparisonEng := vendorcomparison.NewEngine(pricingStore.DB(), logger)
	batchNegStore, err := batchnegotiation.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize batch negotiation store", "error", err.Error())
		os.Exit(1)
	}
	batchNegotiationEng := batchnegotiation.NewEngine(batchNegStore, historyStore, logger)
	strategyComparisonEng := strategycomparison.NewEngine(pricingStore.DB(), logger)

	// Initialize reports, pricing index, and price chart engines
	reportsEng := reports.NewEngine(historyStore)
	pricingIndexEng := pricingindex.NewEngine(pricingStore)
	priceChartEng := pricechart.NewEngine(trendsStore, historyStore)

	// Initialize workspace store and engine
	workspaceStore, err := workspaces.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize workspace store", "error", err.Error())
		os.Exit(1)
	}
	workspaceEng := workspaces.NewEngine(workspaceStore, pricingStore.DB(), logger)

	// Initialize audit log store and engine
	auditLogStore, err := auditlog.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize audit log store", "error", err.Error())
		os.Exit(1)
	}
	auditLogEng := auditlog.NewEngine(auditLogStore, logger)

	// Initialize user activity engine (read-only)
	userActivityEng := useractivity.NewEngine(pricingStore.DB(), logger)

	// Initialize contract templates store and engine
	contractTemplatesStore, err := contracttemplates.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize contract templates store", "error", err.Error())
		os.Exit(1)
	}
	contractTemplatesEng := contracttemplates.NewEngine(contractTemplatesStore, logger)

	// Initialize contract risk engine (no store needed)
	contractRiskEng := contractrisk.NewEngine()

	// Initialize scorecards engine (read-only, uses deal_outcomes, vendor_health, sla_breaches)
	// Initialize scorecards engine (read-only, uses deal_outcomes, vendor_health, sla_breaches)
	scorecardsEng := scorecards.NewEngine(pricingStore.DB(), logger)

	// Initialize shared strategies store and engine
	sharedStrategiesStore, err := sharedstrategies.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize shared strategies store", "error", err.Error())
		os.Exit(1)
	}
	sharedStrategiesEng := sharedstrategies.NewEngine(sharedStrategiesStore)

	// Initialize notes store and engine
	notesStore, err := notes.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize notes store", "error", err.Error())
		os.Exit(1)
	}
	notesEng := notes.NewEngine(notesStore)

	// Initialize approvals store and engine
	approvalsStore, err := approvals.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize approvals store", "error", err.Error())
		os.Exit(1)
	}
	approvalsEng := approvals.NewEngine(approvalsStore)

	// Initialize budget management store and engine (P57)
	budgetmgmtStore, err := budgetmgmt.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("budgetmgmt store", "error", err)
		os.Exit(1)
	}
	budgetMgmtEng := budgetmgmt.NewEngine(budgetmgmtStore, pricingStore.DB(), logger)
	// Initialize spending caps store and engine (P58)
	spendingcapsStore, err := spendingcaps.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("spendingcaps store", "error", err)
		os.Exit(1)
	}
	spendingCapsEng := spendingcaps.NewEngine(spendingcapsStore, pricingStore.DB(), logger)
	// Initialize savings realization store and engine (P59)
	savingsrealizationStore, err := savingsrealization.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("savingsrealization store", "error", err)
		os.Exit(1)
	}
	savingsRealizationEng := savingsrealization.NewEngine(savingsrealizationStore, logger)

	// Initialize TCO engine (P60)
	tcoEng := tco.NewEngine(pricingStore)

	// Initialize data import engine (P61)
	dataImportEng := dataimport.NewEngine(pricingStore, historyStore)

	// Initialize cost allocation store and engine (P62)
	costAllocationStore, err := costallocation.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize cost allocation store", "error", err.Error())
		os.Exit(1)
	}
	costAllocationEng := costallocation.NewEngine(costAllocationStore, pricingStore.DB())

	// Initialize alert history engine (P63)
	alertHistoryEng := alerthistory.NewEngine(pricingStore.DB(), logger)

	// Initialize SLA credit calculator engine (P64)
	slaCreditEng := slacredit.NewEngine(logger)

	// Initialize vendor communication log store and engine (P65)
	commLogStore, err := commlog.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize commlog store", "error", err.Error())
		os.Exit(1)
	}
	commLogEng := commlog.NewEngine(commLogStore, pricingStore.DB(), logger)

	// Initialize limited offer engine (P66)
	limitedOfferEng := limitedoffer.NewEngine(pricingStore.DB(), logger)

	// Initialize pricing refresh engine (P67)
	pricingRefreshEng := pricingrefresh.NewEngine(pricingStore.DB(), logger)

	// Initialize rate limit dashboard store and engine (P68)
	rateLimitDashStore, err := ratelimitdashboard.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize rate limit dashboard store", "error", err.Error())
		os.Exit(1)
	}
	rateLimitDashEng := ratelimitdashboard.NewEngine(rateLimitDashStore, logger)

	// Initialize tool stats store (P70)
	toolStatsStore, err := toolstats.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize tool stats store", "error", err.Error())
		os.Exit(1)
	}
	toolStatsEng := toolstats.NewEngine(toolStatsStore)

	// Initialize API docs engine (P69)
	apiDocsEng := apidocs.NewEngine(nil) // set after negServer creation since mcpServer isn't available yet

	// Initialize health check engine (P71)
	healthCheckEng := healthcheck.NewEngine(pricingStore.DB(), 0, time.Now(), *dbPath)

	// Initialize autocomplete engine (P72)
	autocompleteEng := autocomplete.NewEngine(nil) // set after negServer creation since mcpServer isn't available yet

	// Initialize metrics engine (P73)
	metricsEng := metrics.NewEngine(historyStore)

	// Initialize shutdown engine (P74)
	shutdownEng := shutdown.NewEngine()

	// Initialize coverage, dependency, and contrib guide engines (P75-P77)
	coverageEng := coverage.NewEngine()
	dependencyEng := dependency.NewEngine()
	contribguideEng := contribguide.NewEngine()
	apiKeyRotateEng := apikeyrotation.NewEngine()
	ipWhitelistEng := ipwhitelist.NewEngine()
	dataRetentionEng := dataretention.NewEngine()
	playbookEng := playbook.NewEngine()

	vendorKnowledgeStore, err := vendorknowledge.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize vendor knowledge store", "error", err.Error())
		os.Exit(1)
	}
	industryReportsStore, err := industryreports.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize industry reports store", "error", err.Error())
		os.Exit(1)
	}
	webhookLogStore, err := webhooklog.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize webhook log store", "error", err.Error())
		os.Exit(1)
	}

	toolBillingStore, err := toolbilling.NewStore(pricingStore.DB())

	sandboxStore, err := sandbox.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize sandbox store", "error", err.Error())
		os.Exit(1)
	}
	sandboxEng := sandbox.NewEngine()
	if err != nil {
		logger.Error("failed to initialize tool billing store", "error", err.Error())
		os.Exit(1)
	}

	clausesStore, err := contractclauses.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize contract clauses store", "error", err.Error())
		os.Exit(1)
	}
	if err != nil {
		logger.Error("failed to initialize industry reports store", "error", err.Error())
		os.Exit(1)
	}

	aiPerfStore, err := aiperformance.NewStore(pricingStore.DB())
	promptsStore, err := prompts.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize prompts store", "error", err.Error())
		os.Exit(1)
	}

	if err != nil {
		logger.Error("failed to initialize AI performance store", "error", err.Error())
		os.Exit(1)
	}

	trainingEng := training.NewEngine()
        summarizerEng := summarizer.NewEngine()
	modelABTestEng := modelabtesting.NewEngine()
	sentimentEng := sentiment.NewEngine()

	translationStore, _ := translation.NewStore(pricingStore.DB())
	translationEng := translation.NewEngine()

	complianceEng := compliance.NewEngine()

	esigStore, err := esignature.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize e-signature store", "error", err.Error())
		os.Exit(1)
	}
	esigEng := esignature.NewEngine()
	// Initialize datresidency store (shares the same DB)
	residencyStore, err := datresidency.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize datresidency store", "error", err.Error())
		os.Exit(1)
	}

	// Initialize dashboard store (shares the same DB)
	dashboardStore, err := dashboard.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize dashboard store", "error", err.Error())
		os.Exit(1)
	}


        // Initialize chart export engine (no store needed, stateless)
        chartExportEng := chartexport.NewEngine()

	// Initialize monitoring dashboard engine (no store needed, stateless)
	monitorDashEng := monitordash.NewEngine()

	negServer := server.NewNegotiationServer(pricingStore, historyStore, groupEngine, sellEngine, calendarEngine, healthEngine, marketplaceEngine, slaEngine, webhookEng, slackClient, apiKeyStore, roiStore, trendsStore, exportStore, notifyStore, budgetStore, vendorspendEng, effectivenessEng, priceAlertStore, budgetAlertStore, reportsEng, pricingIndexEng, priceChartEng, vendorComparisonEng, batchNegotiationEng, strategyComparisonEng, workspaceEng, auditLogEng, userActivityEng, contractTemplatesEng, contractRiskEng, scorecardsEng, sharedStrategiesEng, notesEng, approvalsEng, budgetMgmtEng, spendingCapsEng, savingsRealizationEng, tcoEng, dataImportEng, costAllocationEng, alertHistoryEng, slaCreditEng, commLogEng, limitedOfferEng, pricingRefreshEng, rateLimitDashEng, apiDocsEng, toolStatsEng, healthCheckEng, autocompleteEng, metricsEng, shutdownEng, coverageEng, dependencyEng, contribguideEng, apiKeyRotateEng, ipWhitelistEng, dataRetentionEng, playbookEng, trainingEng, industryReportsStore, aiPerfStore, promptsStore, modelABTestEng, vendorKnowledgeStore, summarizerEng, sentimentEng, translationStore, translationEng, complianceEng, clausesStore, esigStore, esigEng, residencyStore, dashboardStore, chartExportEng, monitorDashEng, webhookLogStore, toolBillingStore, sandboxStore, sandboxEng, logger)

	// Initialize gamification store and engine (for streaks, leaderboard, badges)
	gamifStore, err := gamification.NewStore(pricingStore.DB())
	if err != nil {
		logger.Error("failed to initialize gamification store", "error", err.Error())
		os.Exit(1)
	}
	gamifEng := gamification.New(gamifStore, logger)
	negServer.SetGamificationEngine(gamifEng)

	// Set the MCP server on the API docs engine (needed to enumerate tools)
	apiDocsEng = apidocs.NewEngine(negServer.MCPServer())
	// Set the MCP server on the autocomplete engine (needed to enumerate tools)
	autocompleteEng = autocomplete.NewEngine(negServer.MCPServer())
	// Update healthcheck with actual tool count after all tools are registered
	healthCheckEng.SetToolCount(len(negServer.MCPServer().ListTools()))

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server if -http flag provided
	if *httpAddr != "" {
		baseURL := fmt.Sprintf("http://%s", *httpAddr)
		a2aRouter := a2a.NewRouter(pricingStore, historyStore, mandateStore, logger, baseURL, apiKeyStore, rateLimiter)
		httpServer := &http.Server{
			Addr:    *httpAddr,
			Handler: a2aRouter,
		}

		go func() {
			logger.Info("A2A HTTP server starting", "address", *httpAddr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("http server error", "error", err.Error())
				os.Exit(1)
			}
		}()

		// Shutdown HTTP server on signal
		go func() {
			<-sigCh
			logger.Info("shutting down http server...")
			httpServer.Shutdown(ctx)
			cancel()
			negServer.MCPServer().SendNotificationToAllClients("notification", map[string]any{"type": "shutdown"})
		}()
	} else {
		go func() {
			<-sigCh
			logger.Info("shutting down...")
			cancel()
			negServer.MCPServer().SendNotificationToAllClients("notification", map[string]any{"type": "shutdown"})
		}()
	}

	// Always start stdio MCP server (blocks until stdio closes)
	if *httpAddr != "" {
		logger.Info("server ready, listening on stdio and HTTP", "http_addr", *httpAddr)
	} else {
		logger.Info("server ready, listening on stdio")
	}

	if err := mcpserver.ServeStdio(negServer.MCPServer()); err != nil {
		logger.Error("server error", "error", err.Error())
		os.Exit(1)
	}
}

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".a2a-negotiation")
}

// seedSnapshotsFromCSV reads a CSV of price snapshots and bulk-inserts them
// for trend analysis. This is a first-run operation when -seed is provided.
func seedSnapshotsFromCSV(ctx context.Context, store *trends.Store, path string, logger *slog.Logger) {
	importCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read seed CSV", "error", err.Error())
		return
	}

	lines := strings.Split(string(data), "\n")
	var snapshots []trends.PriceSnapshot
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		price, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		listPrice, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		date, err := time.Parse(time.DateOnly, strings.TrimSpace(parts[4]))
		if err != nil {
			continue
		}

		snapshots = append(snapshots, trends.PriceSnapshot{
			Vendor:    strings.TrimSpace(parts[0]),
			SKU:       strings.TrimSpace(parts[1]),
			Price:     price,
			ListPrice: listPrice,
			Date:      date,
			CreatedAt: time.Now().UTC(),
		})
	}

	if len(snapshots) == 0 {
		logger.Warn("no price snapshots to seed")
		return
	}

	if err := store.BulkInsert(importCtx, snapshots); err != nil {
		logger.Error("failed to bulk insert price snapshots", "error", err.Error())
		return
	}

	logger.Info("price snapshots seeded", "count", len(snapshots))
}
