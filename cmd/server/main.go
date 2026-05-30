package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	var (
		dbPath    = flag.String("db", filepath.Join(dataDir(), "negotiations.db"), "Path to SQLite database")
		seedPath  = flag.String("seed", "", "Path to seed CSV file (optional)")
		logFormat = flag.String("log", "json", "Log format: json or text")
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

	// Create MCP negotiation server
	negServer := server.NewNegotiationServer(pricingStore, historyStore, logger)

	logger.Info("server ready, listening on stdio")

	// Handle graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutting down...")
		cancel()
		negServer.MCPServer().SendNotificationToAllClients("notification", map[string]any{"type": "shutdown"})
	}()

	// Start stdio server
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
