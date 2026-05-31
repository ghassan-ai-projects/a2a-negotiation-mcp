package batchnegotiation

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/google/uuid"
)

// Engine manages batch negotiation operations.
type Engine struct {
	store        *Store
	historyStore *history.Store
	logger       *slog.Logger
}

// NewEngine creates a batchnegotiation Engine.
func NewEngine(store *Store, historyStore *history.Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:        store,
		historyStore: historyStore,
		logger:       logger,
	}
}

// Run executes a batch negotiation. Creates sessions via history store and runs strategy logic.
func (e *Engine) Run(ctx context.Context, req BatchRequest) (BatchResult, error) {
	if len(req.Items) == 0 {
		return BatchResult{}, fmt.Errorf("batch items cannot be empty")
	}

	start := time.Now()
	batchID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var results []BatchItemResult
	totalSavings := 0.0

	for _, item := range req.Items {
		sessionID := uuid.New().String()

		// Determine list price and compute savings based on strategy
		listPrice := item.Budget
		if listPrice <= 0 {
			listPrice = 100.0 // default if no budget provided
		}

		discountPct := 0.0
		switch item.Strategy {
		case "aggressive":
			discountPct = 0.30
		case "balanced":
			discountPct = 0.20
		case "conservative":
			discountPct = 0.10
		default:
			discountPct = 0.15
		}

		finalPrice := listPrice * (1 - discountPct)
		annualSavings := listPrice - finalPrice
		if annualSavings < 0 {
			annualSavings = 0
		}

		// Save a session record via history store
		session := &history.SessionRecord{
			ID:        sessionID,
			Vendor:    item.Vendor,
			SKU:       item.SKU,
			Strategy:  item.Strategy,
			Budget:    item.Budget,
			Status:    "completed",
			ListPrice: listPrice,
			Outcome:   "won",
		}
		if err := e.historyStore.SaveSession(ctx, session); err != nil {
			e.logger.Warn("batch: failed to save session", "vendor", item.Vendor, "error", err.Error())
			results = append(results, BatchItemResult{
				Vendor: item.Vendor,
				Status: "failed",
			})
			continue
		}

		totalSavings += annualSavings

		results = append(results, BatchItemResult{
			Vendor:    item.Vendor,
			SessionID: sessionID,
			Status:    "completed",
			Savings:   math.Round(annualSavings*100) / 100,
		})
	}

	durationMs := time.Since(start).Milliseconds()

	// Save the batch record
	record := &BatchRecord{
		ID:           batchID,
		VendorCount:  len(req.Items),
		TotalSavings: math.Round(totalSavings*100) / 100,
		DurationMs:   durationMs,
		CreatedAt:    now,
	}
	if err := e.store.Save(ctx, record); err != nil {
		e.logger.Warn("batch: failed to save batch record", "error", err.Error())
	}

	return BatchResult{
		BatchID:      batchID,
		Results:      results,
		TotalSavings: math.Round(totalSavings*100) / 100,
		DurationMs:   durationMs,
		CreatedAt:    now,
	}, nil
}
