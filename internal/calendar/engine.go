package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/google/uuid"
)

// Engine orchestrates renewal calendar checks and auto-negotiation.
type Engine struct {
	store        *Store
	negEng       *negotiation.Engine
	histStore    *history.Store
	pricingStore *pricing.Store
	logger       *slog.Logger
}

// NewEngine creates a new calendar engine.
// Store returns the underlying calendar store.
func (e *Engine) Store() *Store {
	return e.store
}

func NewEngine(store *Store, negEng *negotiation.Engine, histStore *history.Store, pricingStore *pricing.Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:        store,
		negEng:       negEng,
		histStore:    histStore,
		pricingStore: pricingStore,
		logger:       logger,
	}
}

// CheckRenewals returns contracts renewing within N days with urgency classification.
func (e *Engine) CheckRenewals(ctx context.Context, daysAhead int) ([]RenewalCheck, error) {
	contracts, err := e.store.GetContractsExpiringSoon(ctx, daysAhead)
	if err != nil {
		return nil, fmt.Errorf("check renewals: %w", err)
	}

	now := time.Now().UTC()
	checks := make([]RenewalCheck, 0, len(contracts))

	for _, c := range contracts {
		daysUntil := int(math.Ceil(c.RenewalDate.Sub(now).Hours() / 24))
		if daysUntil < 0 {
			daysUntil = 0
		}

		check := RenewalCheck{
			Contract:  c,
			DaysUntil: daysUntil,
		}

		// Urgency classification
		switch {
		case daysUntil < 30:
			check.Urgency = "high"
		case daysUntil <= 90:
			check.Urgency = "medium"
		default:
			check.Urgency = "low"
		}

		// Action needed
		switch {
		case daysUntil < 14:
			check.ActionNeeded = "urgent"
		case daysUntil <= 60:
			check.ActionNeeded = "soon"
		case daysUntil <= 90:
			check.ActionNeeded = "monitor"
		default:
			check.ActionNeeded = "none"
		}

		// Compute suggested savings via market comparison
		savings, err := e.computeSuggestedSavings(ctx, c)
		if err != nil {
			e.logger.Debug("could not compute savings for contract",
				"contract_id", c.ID, "vendor", c.Vendor, "error", err.Error())
		} else {
			check.SuggestedSavings = savings
		}

		checks = append(checks, check)
	}

	return checks, nil
}

// TriggerNegotiation auto-starts a negotiation for a contract.
// Creates a session via negotiation engine, runs it, and updates contract status.
func (e *Engine) TriggerNegotiation(ctx context.Context, contractID string) (*negotiation.Session, error) {
	contract, err := e.store.GetContract(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("trigger negotiation: %w", err)
	}

	if contract.Status != "active" {
		return nil, fmt.Errorf("contract %s is not active (status: %s)", contractID, contract.Status)
	}

	// Mark as negotiating
	contract.Status = "negotiating"
	if err := e.store.UpdateContract(ctx, contract); err != nil {
		return nil, fmt.Errorf("update contract status to negotiating: %w", err)
	}

	// Create a negotiation session
	session, err := e.negEng.CreateSession(ctx, contract.Vendor, contract.SKU, "balanced", contract.CurrentPrice, nil, "")
	if err != nil {
		// Revert status on failure
		contract.Status = "active"
		_ = e.store.UpdateContract(ctx, contract)
		return nil, fmt.Errorf("create negotiation session: %w", err)
	}

	session.ID = uuid.New().String()

	// Run negotiation with balanced strategy (4 rounds max)
	result, rounds, err := e.negEng.RunNegotiation(ctx, session, 4, 0)
	if err != nil {
		contract.Status = "active"
		_ = e.store.UpdateContract(ctx, contract)
		return nil, fmt.Errorf("run negotiation: %w", err)
	}

	e.logger.Info("negotiation completed for contract",
		"contract_id", contractID,
		"outcome", result.Outcome,
		"final_offer", result.CurrentOffer,
		"rounds", result.RoundsComplete)

	// Save session to history store
	histSess := &history.SessionRecord{
		ID:             session.ID,
		Vendor:         session.Vendor,
		SKU:            session.SKU,
		Strategy:       session.Strategy,
		Budget:         session.Budget,
		Status:         session.Status,
		CurrentOffer:   session.CurrentOffer,
		ListPrice:      session.ListPrice,
		RoundsComplete: session.RoundsComplete,
		Outcome:        session.Outcome,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}
	if err := e.histStore.SaveSession(ctx, histSess); err != nil {
		e.logger.Error("failed to save negotiation session", "error", err.Error())
	}

	// Save rounds
	var roundRecords []history.RoundRecord
	for _, r := range rounds {
		if r.SessionID == "" {
			r.SessionID = session.ID
		}
		roundRecords = append(roundRecords, history.RoundRecord{
			SessionID:    session.ID,
			RoundNumber:  r.RoundNumber,
			Offer:        r.Offer,
			DiscountPct:  r.DiscountPct,
			Counterparty: r.Counterparty,
			Note:         r.Note,
			CreatedAt:    r.CreatedAt,
		})
	}
	if len(roundRecords) > 0 {
		if err := e.histStore.SaveRounds(ctx, roundRecords); err != nil {
			e.logger.Error("failed to save rounds", "error", err.Error())
		}
	}

	// Save deal outcome if accepted
	if result.Outcome == "accepted" {
		dealOutcome := &history.DealOutcome{
			Vendor:      contract.Vendor,
			SKU:         contract.SKU,
			ListPrice:   contract.CurrentPrice,
			FinalPrice:  result.CurrentOffer,
			DiscountPct: result.TotalDiscount,
			Seats:       contract.Seats,
			TermMonths:  12,
			Strategy:    "balanced",
			SessionID:   session.ID,
			CreatedAt:   time.Now().UTC(),
		}
		if err := e.histStore.SaveDealOutcome(ctx, dealOutcome); err != nil {
			e.logger.Error("failed to save deal outcome", "error", err.Error())
		}
	}

	// Update contract with final state
	contract.Status = "renewed"
	contract.LastNegotiatedPrice = result.CurrentOffer
	contract.CurrentPrice = result.CurrentOffer
	if err := e.store.UpdateContract(ctx, contract); err != nil {
		e.logger.Error("failed to update contract after negotiation", "error", err.Error())
	}

	return session, nil
}

// computeSuggestedSavings computes the potential savings by comparing current price to market data.
func (e *Engine) computeSuggestedSavings(ctx context.Context, c Contract) (float64, error) {
	priceResult, err := e.pricingStore.GetPricingByVendorSKU(ctx, c.Vendor, c.SKU)
	if err != nil {
		return 0, err
	}

	if priceResult.SuggestedMin > 0 && c.CurrentPrice > priceResult.SuggestedMin {
		perUnit := c.CurrentPrice - priceResult.SuggestedMin
		total := perUnit * float64(c.Seats) * 12 // annualized
		if total < 0 {
			total = 0
		}
		return math.Round(total*100) / 100, nil
	}
	return 0, nil
}
