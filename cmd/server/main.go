package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/a2a"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/group"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sell"
        "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/marketplace"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	var (
		dbPath    = flag.String("db", filepath.Join(dataDir(), "negotiations.db"), "Path to SQLite database")
		seedPath  = flag.String("seed", "", "Path to seed CSV file (optional)")
		logFormat = flag.String("log", "json", "Log format: json or text")
		httpAddr  = flag.String("http", "", "HTTP listen address for A2A endpoints (e.g. :8080)")
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
	// Create MCP negotiation server
	negServer := server.NewNegotiationServer(pricingStore, historyStore, groupEngine, sellEngine, calendarEngine, marketplaceEngine, logger)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server if -http flag provided
	if *httpAddr != "" {
		baseURL := fmt.Sprintf("http://%s", *httpAddr)
		a2aRouter := a2a.NewRouter(pricingStore, historyStore, mandateStore, logger, baseURL)
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
