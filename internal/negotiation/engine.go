package negotiation

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// Session represents a negotiation session.
type Session struct {
	ID             string         `json:"session_id"`
	Vendor         string         `json:"vendor"`
	SKU            string         `json:"sku,omitempty"`
	Strategy       string         `json:"strategy"`
	Budget         float64        `json:"budget,omitempty"`
	Constraints    map[string]any `json:"constraints,omitempty"`
	Status         string         `json:"status"`
	CurrentOffer   float64        `json:"current_offer"`
	ListPrice      float64        `json:"list_price"`
	RoundsComplete int            `json:"rounds_completed"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Outcome        string         `json:"outcome,omitempty"`
}

// Round represents a single round of negotiation.
type Round struct {
	ID           int       `json:"id"`
	SessionID    string    `json:"session_id"`
	RoundNumber  int       `json:"round_number"`
	Offer        float64   `json:"offer"`
	DiscountPct  float64   `json:"discount_percentage"`
	Counterparty string    `json:"counterparty"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// NegotiateResult is the outcome of a negotiation run.
type NegotiateResult struct {
	Status         string  `json:"status"`
	CurrentOffer   float64 `json:"current_offer"`
	RoundsComplete int     `json:"rounds_completed"`
	Outcome        string  `json:"outcome,omitempty"`
	History        []Round `json:"history,omitempty"`
	ListPrice      float64 `json:"list_price"`
	TotalDiscount  float64 `json:"total_discount_pct"`
}

// Engine runs the negotiation strategy.
type Engine struct {
	pricingStore *pricing.Store
}

// NewEngine creates a new negotiation engine.
func NewEngine(pricingStore *pricing.Store) *Engine {
	return &Engine{pricingStore: pricingStore}
}

// CreateSession initializes a new negotiation session.
func (e *Engine) CreateSession(ctx context.Context, vendor, sku, strategyName string, budget float64, constraints map[string]any) (*Session, error) {
	strategy := GetStrategy(strategyName)
	if strategy == nil {
		return nil, ierrors.New(ierrors.ErrInvalidStrategy, "unknown strategy",
			map[string]any{"strategy": strategyName})
	}

	var listPrice float64
	priceResult, err := e.pricingStore.GetPricingByVendorSKU(ctx, vendor, sku)
	if err != nil {
		if de, ok := err.(*ierrors.DomainError); ok && de.Code == ierrors.ErrPricingNotFound {
			listPrice = budget
		} else if de, ok := err.(*ierrors.DomainError); ok && de.Code == ierrors.ErrVendorNotFound {
			return nil, err
		} else {
			return nil, err
		}
	} else {
		listPrice = priceResult.ListPrice
	}

	initialDiscount := listPrice * strategy.InitialDiscountPct
	initialOffer := listPrice - initialDiscount

	now := time.Now().UTC()
	session := &Session{
		Vendor:         vendor,
		SKU:            sku,
		Strategy:       strategyName,
		Budget:         budget,
		Constraints:    constraints,
		Status:         "active",
		CurrentOffer:   initialOffer,
		ListPrice:      listPrice,
		RoundsComplete: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return session, nil
}

// RunNegotiation executes the multi-round negotiation loop for a session.
func (e *Engine) RunNegotiation(ctx context.Context, session *Session, maxRounds int, autoApproveThreshold float64) (*NegotiateResult, []Round, error) {
	if session.Status != "active" {
		return nil, nil, ierrors.New(ierrors.ErrNegotiationLimit, "session is not active",
			map[string]any{"session_id": session.ID, "status": session.Status})
	}

	strategy := GetStrategy(session.Strategy)
	if strategy == nil {
		return nil, nil, ierrors.New(ierrors.ErrInvalidStrategy, "strategy not found",
			map[string]any{"strategy": session.Strategy})
	}

	if maxRounds <= 0 || maxRounds > strategy.MaxRounds {
		maxRounds = strategy.MaxRounds
	}

	var rounds []Round
	currentOffer := session.CurrentOffer
	currentDiscount := strategy.InitialDiscountPct

	for round := 1; round <= maxRounds; round++ {
		discount := strategy.InitialDiscountPct + float64(round-1)*strategy.ConcessionPerRound
		if discount > strategy.WalkAwayThresholdPct {
			rounds = append(rounds, Round{
				SessionID: session.ID, RoundNumber: round,
				Offer: currentOffer, DiscountPct: currentDiscount,
				Counterparty: "seller",
				Note:         "Seller rejected — beyond walk-away threshold",
				CreatedAt:    time.Now().UTC(),
			})
			session.Status = "completed"
			session.Outcome = "walked_away"
			session.CurrentOffer = currentOffer
			session.RoundsComplete = round
			session.UpdatedAt = time.Now().UTC()
			break
		}

		offerAmount := session.ListPrice * (1 - discount)
		offerAmount = math.Round(offerAmount*100) / 100

		if autoApproveThreshold > 0 && offerAmount <= autoApproveThreshold {
			currentOffer = offerAmount
			currentDiscount = discount
			rounds = append(rounds, Round{
				SessionID: session.ID, RoundNumber: round,
				Offer: offerAmount, DiscountPct: discount,
				Counterparty: "seller",
				Note:         "Auto-accepted at or below threshold",
				CreatedAt:    time.Now().UTC(),
			})
			session.Status = "completed"
			session.Outcome = "accepted"
			session.CurrentOffer = offerAmount
			session.RoundsComplete = round
			session.UpdatedAt = time.Now().UTC()
			break
		}

		note := fmt.Sprintf("Buyer counter-offer round %d: %.0f%% discount", round, discount*100)
		rounds = append(rounds, Round{
			SessionID: session.ID, RoundNumber: round,
			Offer: offerAmount, DiscountPct: discount,
			Counterparty: "buyer", Note: note,
			CreatedAt: time.Now().UTC(),
		})
		currentOffer = offerAmount
		currentDiscount = discount

		if round == maxRounds {
			acceptNote := "Seller accepted final offer"
			if session.Budget > 0 && offerAmount <= session.Budget {
				acceptNote = "Seller accepted — within budget"
			}
			rounds = append(rounds, Round{
				SessionID: session.ID, RoundNumber: round + 1,
				Offer: offerAmount, DiscountPct: discount,
				Counterparty: "seller", Note: acceptNote,
				CreatedAt: time.Now().UTC(),
			})
			session.Status = "completed"
			session.Outcome = "accepted"
			session.CurrentOffer = offerAmount
			session.RoundsComplete = round + 1
			session.UpdatedAt = time.Now().UTC()
			break
		}

		if session.Budget > 0 && offerAmount <= session.Budget {
			rounds = append(rounds, Round{
				SessionID: session.ID, RoundNumber: round + 1,
				Offer: offerAmount, DiscountPct: discount,
				Counterparty: "seller",
				Note:         "Seller accepted — within budget",
				CreatedAt:    time.Now().UTC(),
			})
			session.Status = "completed"
			session.Outcome = "accepted"
			session.CurrentOffer = offerAmount
			session.RoundsComplete = round + 1
			session.UpdatedAt = time.Now().UTC()
			break
		}
	}

	if session.Status == "active" {
		session.Status = "completed"
		session.Outcome = "rejected"
		session.UpdatedAt = time.Now().UTC()
	}

	totalDiscount := 0.0
	if session.ListPrice > 0 {
		totalDiscount = 1 - (session.CurrentOffer / session.ListPrice)
	}

	result := &NegotiateResult{
		Status:         session.Status,
		CurrentOffer:   session.CurrentOffer,
		RoundsComplete: session.RoundsComplete,
		Outcome:        session.Outcome,
		History:        rounds,
		ListPrice:      session.ListPrice,
		TotalDiscount:  math.Round(totalDiscount*10000) / 10000,
	}
	return result, rounds, nil
}

// ComputeSavings calculates potential savings for a vendor spend.
func (e *Engine) ComputeSavings(ctx context.Context, vendor string, currentSpend float64) (*pricing.SavingsEstimate, error) {
	marketRange, err := e.pricingStore.GetMarketRange(ctx, vendor)
	if err != nil {
		return nil, err
	}

	marketAvg, err := e.pricingStore.GetMarketAverageForVendor(ctx, vendor)
	if err != nil {
		return nil, err
	}

	var savings float64
	if currentSpend > marketAvg && marketAvg > 0 {
		savings = currentSpend - marketAvg
	} else {
		savings = currentSpend * 0.15
	}

	savingsPct := 0.0
	if currentSpend > 0 {
		savingsPct = (savings / currentSpend) * 100
		savingsPct = math.Round(savingsPct*100) / 100
	}

	confidence := "low"
	if marketRange.Count >= 20 {
		confidence = "high"
	} else if marketRange.Count >= 5 {
		confidence = "medium"
	}

	return &pricing.SavingsEstimate{
		Vendor:             vendor,
		CurrentSpend:       currentSpend,
		EstimatedSavings:   math.Round(savings*100) / 100,
		SavingsPercentage:  savingsPct,
		Confidence:         confidence,
		SimilarDeals:       []pricing.SimilarDeal{},
		MarketAveragePrice: math.Round(marketAvg*100) / 100,
	}, nil
}
