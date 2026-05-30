package parallel

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// Engine orchestrates parallel negotiations across multiple sessions.
type Engine struct {
	negEng    *negotiation.Engine
	histStore *history.Store
	pricing   *pricing.Store
	logger    *slog.Logger
}

// NewEngine creates a new parallel negotiation engine.
func NewEngine(negEng *negotiation.Engine, histStore *history.Store, pricing *pricing.Store, logger *slog.Logger) *Engine {
	return &Engine{
		negEng:    negEng,
		histStore: histStore,
		pricing:   pricing,
		logger:    logger,
	}
}

// RunParallel executes negotiations concurrently using goroutines + errgroup.
// Returns ParallelResult with the best offer selected by strategy.
func (e *Engine) RunParallel(ctx context.Context, cfg ParallelConfig) (*ParallelResult, error) {
	if len(cfg.SessionIDs) == 0 {
		return nil, fmt.Errorf("parallel: session_ids cannot be empty")
	}

	strategy := cfg.Strategy
	if strategy == "" {
		strategy = "best_price"
	}

	timeoutSec := cfg.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	var (
		g       errgroup.Group
		mu      sync.Mutex
		results []SessionSummary
	)

	start := time.Now()

	for _, sid := range cfg.SessionIDs {
		sid := sid // capture range variable
		g.Go(func() error {
			sessCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			sessStart := time.Now()

			// Load session from history store
			sessRec, err := e.histStore.GetSession(sessCtx, sid)
			if err != nil {
				return fmt.Errorf("parallel: session %s: %w", sid, err)
			}

			// Reconstruct negotiation.Session
			session := &negotiation.Session{
				ID:             sessRec.ID,
				Vendor:         sessRec.Vendor,
				SKU:            sessRec.SKU,
				Strategy:       sessRec.Strategy,
				Status:         sessRec.Status,
				Budget:         sessRec.Budget,
				CurrentOffer:   sessRec.CurrentOffer,
				ListPrice:      sessRec.ListPrice,
				RoundsComplete: sessRec.RoundsComplete,
				Outcome:        sessRec.Outcome,
				CreatedAt:      sessRec.CreatedAt,
				UpdatedAt:      sessRec.UpdatedAt,
			}

			// Run negotiation
			result, rounds, err := e.negEng.RunNegotiation(sessCtx, session, 0, 0)
			if err != nil {
				return fmt.Errorf("parallel: session %s negotiation: %w", sid, err)
			}

			// Persist updated session
			sessRec.Status = session.Status
			sessRec.CurrentOffer = session.CurrentOffer
			sessRec.RoundsComplete = session.RoundsComplete
			sessRec.Outcome = session.Outcome
			sessRec.UpdatedAt = session.UpdatedAt
			if err := e.histStore.UpdateSession(sessCtx, sessRec); err != nil {
				e.logger.Warn("parallel: failed to update session", "session_id", sid, "error", err)
			}

			// Persist rounds
			var roundRecords []history.RoundRecord
			for _, r := range rounds {
				roundRecords = append(roundRecords, history.RoundRecord{
					SessionID:    r.SessionID,
					RoundNumber:  r.RoundNumber,
					Offer:        r.Offer,
					DiscountPct:  r.DiscountPct,
					Counterparty: r.Counterparty,
					Note:         r.Note,
					CreatedAt:    r.CreatedAt,
				})
			}
			if len(roundRecords) > 0 {
				if err := e.histStore.SaveRounds(sessCtx, roundRecords); err != nil {
					e.logger.Warn("parallel: failed to save rounds", "session_id", sid, "error", err)
				}
			}

			// Persist deal outcome if accepted
			if session.Outcome == "accepted" {
				deal := &history.DealOutcome{
					Vendor:      session.Vendor,
					SKU:         session.SKU,
					ListPrice:   session.ListPrice,
					FinalPrice:  session.CurrentOffer,
					DiscountPct: result.TotalDiscount,
					Seats:       0,
					TermMonths:  12,
					Strategy:    session.Strategy,
					SessionID:   session.ID,
					CreatedAt:   time.Now().UTC(),
				}
				if err := e.histStore.SaveDealOutcome(sessCtx, deal); err != nil {
					e.logger.Warn("parallel: failed to save deal outcome", "session_id", sid, "error", err)
				}
			}

			durationMs := time.Since(sessStart).Milliseconds()

			discountPct := 0.0
			if session.ListPrice > 0 {
				discountPct = math.Round((1-session.CurrentOffer/session.ListPrice)*10000) / 100
			}

			summary := SessionSummary{
				SessionID:   session.ID,
				Vendor:      session.Vendor,
				SKU:         session.SKU,
				Strategy:    session.Strategy,
				Offer:       session.CurrentOffer,
				DiscountPct: discountPct,
				Outcome:     session.Outcome,
				Rounds:      session.RoundsComplete,
				DurationMs:  durationMs,
			}

			mu.Lock()
			results = append(results, summary)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	totalDuration := time.Since(start).Milliseconds()

	winner := e.selectWinner(results, strategy)

	var totalRounds int
	for _, r := range results {
		totalRounds += r.Rounds
	}

	winnerDiscount := 0.0
	if winner.DiscountPct > 0 {
		winnerDiscount = winner.DiscountPct
	}

	return &ParallelResult{
		WinnerSessionID: winner.SessionID,
		WinnerVendor:    winner.Vendor,
		WinnerOffer:     winner.Offer,
		WinnerDiscount:  winnerDiscount,
		Strategy:        strategy,
		TotalRounds:     totalRounds,
		AllResults:      results,
		DurationMs:      totalDuration,
	}, nil
}

// selectWinner picks the best session by the configured strategy.
func (e *Engine) selectWinner(results []SessionSummary, strategy string) SessionSummary {
	if len(results) == 0 {
		return SessionSummary{}
	}

	winner := results[0]

	for _, r := range results[1:] {
		switch strategy {
		case "best_discount":
			// Higher discount is better
			if r.DiscountPct > winner.DiscountPct {
				winner = r
			}
		case "fastest":
			// Fewest rounds is better; tiebreaker: lower price
			if r.Rounds < winner.Rounds || (r.Rounds == winner.Rounds && r.Offer < winner.Offer) {
				winner = r
			}
		default: // "best_price" — lower offer is better
			if r.Offer < winner.Offer {
				winner = r
			}
		}
	}

	return winner
}
