package negotiation

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/health"
)

// Session represents a negotiation session.
type Session struct {
	ID             string         `json:"session_id"`
	Vendor         string         `json:"vendor"`
	SKU            string         `json:"sku,omitempty"`
	Strategy       string         `json:"strategy"`
	Budget         float64        `json:"budget,omitempty"`
	Constraints    map[string]any `json:"constraints,omitempty"`
	Culture        string         `json:"culture,omitempty"`
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

// Engine runs the negotiation strategy and optionally records learning outcomes.
type Engine struct {
	pricingStore *pricing.Store
	learningEng  *learning.Engine
	healthEng    *health.Engine
}

// NewEngine creates a new negotiation engine.
func NewEngine(pricingStore *pricing.Store) *Engine {
	return &Engine{pricingStore: pricingStore}
}

// SetLearningEngine attaches a learning engine to automatically record outcomes.
func (e *Engine) SetLearningEngine(le *learning.Engine) {
	e.learningEng = le
}

// SetHealthEngine attaches a health engine to leverage vendor health data in negotiations.
func (e *Engine) SetHealthEngine(he *health.Engine) {
	e.healthEng = he
}

// CreateSession initializes a new negotiation session.
func (e *Engine) CreateSession(ctx context.Context, vendor, sku, strategyName string, budget float64, constraints map[string]any, culture string) (*Session, error) {
	strategy := GetStrategy(strategyName)
	if strategy == nil {
		return nil, ierrors.New(ierrors.ErrInvalidStrategy, "unknown strategy",
			map[string]any{"strategy": strategyName})
	}

	// Apply cultural adjustments
	if culture != "" {
		ApplyCulturalAdjustment(strategy, culture)
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
		Culture:        culture,
		Status:         "active",
		CurrentOffer:   initialOffer,
		ListPrice:      listPrice,
		RoundsComplete: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// Check vendor health and adjust strategy if health engine is available
	if e.healthEng != nil {
		leverage, err := e.healthEng.GetLeverage(ctx, vendor)
		if err == nil && leverage != nil {
			if session.Constraints == nil {
				session.Constraints = make(map[string]any)
			}
			session.Constraints["vendor_health_score"] = leverage.Health.Score
			session.Constraints["vendor_health_category"] = leverage.Health.Category
			session.Constraints["leverage"] = leverage.Leverage
			session.Constraints["leverage_suggestion"] = leverage.Suggestion
		}
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

	// Record learning outcome for accepted/walked_away negotiations
	if e.learningEng != nil && (session.Outcome == "accepted" || session.Outcome == "walked_away") {
		outcome := learning.StrategyOutcome{
			Vendor:         session.Vendor,
			SKU:            session.SKU,
			Strategy:       session.Strategy,
			DiscountPct:    result.TotalDiscount,
			RoundsComplete: session.RoundsComplete,
			Outcome:        session.Outcome,
			BudgetUsed:     session.Budget,
			TotalBefore:    session.ListPrice,
			TotalAfter:     session.CurrentOffer,
			Timestamp:      time.Now().UTC(),
		}
		// Non-fatal — log but don't fail the negotiation
		_ = e.learningEng.RecordOutcome(ctx, outcome)
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
