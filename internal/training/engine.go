package training

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

var validStrategies = map[string]bool{
	"competitive":   true,
	"collaborative": true,
	"aggressive":    true,
	"concessionary": true,
	"principled":    true,
}

// Engine runs negotiation training simulations.
type Engine struct{}

// NewEngine creates a new training engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Simulate runs a multi-round negotiation simulation and returns the result.
func (e *Engine) Simulate(vendor string, strategy string, budget float64, rounds int) (*SimulationResult, error) {
	if vendor == "" {
		return nil, fmt.Errorf("vendor must not be empty")
	}
	if !validStrategies[strategy] {
		return nil, fmt.Errorf("invalid strategy %q: must be one of competitive, collaborative, aggressive, concessionary, principled", strategy)
	}
	if budget <= 0 {
		return nil, fmt.Errorf("budget must be greater than 0")
	}
	if rounds < 1 || rounds > 10 {
		return nil, fmt.Errorf("rounds must be between 1 and 10")
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	simRounds := make([]SimulationRound, rounds)

	// Vendor starts at 95% of budget; buyer starts at 50% of budget.
	vendorPrice := budget * 0.95
	buyerOffer := budget * 0.50

	vendorReductionRange := reductionRange(strategy)
	buyerIncreaseRange := increaseRange()

	for i := 0; i < rounds; i++ {
		roundNum := i + 1

		if roundNum%2 == 1 {
			// Vendor's turn (odd rounds: 1, 3, 5, ...)
			reduction := vendorReductionRange[0] + rng.Float64()*(vendorReductionRange[1]-vendorReductionRange[0])
			vendorPrice = vendorPrice * (1 - reduction)
			offer := math.Round(vendorPrice*100) / 100
			discountPct := (budget - offer) / budget * 100

			simRounds[i] = SimulationRound{
				RoundNumber:  roundNum,
				Offer:        offer,
				DiscountPct:  math.Round(discountPct*100) / 100,
				Counterparty: "vendor",
				Note:         fmt.Sprintf("Vendor lowered price by %.1f%% (strategy: %s)", reduction*100, strategy),
			}
		} else {
			// Buyer's turn (even rounds: 2, 4, 6, ...)
			increase := buyerIncreaseRange[0] + rng.Float64()*(buyerIncreaseRange[1]-buyerIncreaseRange[0])
			buyerOffer = buyerOffer * (1 + increase)
			if buyerOffer > vendorPrice {
				buyerOffer = vendorPrice
			}
			offer := math.Round(buyerOffer*100) / 100
			discountPct := (budget - offer) / budget * 100

			simRounds[i] = SimulationRound{
				RoundNumber:  roundNum,
				Offer:        offer,
				DiscountPct:  math.Round(discountPct*100) / 100,
				Counterparty: "buyer",
				Note:         fmt.Sprintf("Buyer increased offer by %.1f%% (strategy: %s)", increase*100, strategy),
			}
		}
	}

	finalOffer := simRounds[rounds-1].Offer
	finalDiscount := (budget - finalOffer) / budget * 100
	finalDiscount = math.Round(finalDiscount*100) / 100

	finalOutcome, lessons := buildOutcome(strategy, finalDiscount, rounds)

	// Add a round-count lesson.
	if rounds >= 7 {
		lessons = append(lessons, "Longer negotiations provide more opportunities to find common ground")
	} else if rounds <= 3 {
		lessons = append(lessons, "Shorter negotiations are efficient but may leave potential savings unexplored")
	} else {
		lessons = append(lessons, "The negotiation duration allowed for balanced exploration of options")
	}

	id := fmt.Sprintf("sim-%d", time.Now().UnixMilli())

	return &SimulationResult{
		ID:            id,
		Vendor:        vendor,
		Strategy:      strategy,
		Budget:        budget,
		TotalRounds:   rounds,
		Rounds:        simRounds,
		FinalOutcome:  finalOutcome,
		TotalDiscount: finalDiscount,
		Lessons:       lessons,
	}, nil
}

// reductionRange returns the [min, max] vendor price reduction per round for a strategy.
func reductionRange(strategy string) [2]float64 {
	switch strategy {
	case "competitive":
		return [2]float64{0.09, 0.13}
	case "collaborative":
		return [2]float64{0.07, 0.11}
	case "aggressive":
		return [2]float64{0.10, 0.14}
	case "concessionary":
		return [2]float64{0.05, 0.09}
	case "principled":
		return [2]float64{0.08, 0.12}
	default:
		return [2]float64{0.08, 0.12}
	}
}

// increaseRange returns the [min, max] buyer offer increase per round (standard).
func increaseRange() [2]float64 {
	return [2]float64{0.03, 0.07}
}

// buildOutcome generates a final_outcome string and lessons based on simulation params.
func buildOutcome(strategy string, discount float64, _ int) (string, []string) {
	var outcome string
	lessons := make([]string, 0, 4)

	switch strategy {
	case "competitive":
		if discount >= 15 {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — competitive stance paid off", discount)
			lessons = append(lessons, "A competitive strategy works well when the vendor has room to negotiate")
			lessons = append(lessons, "Starting with firm offers signaled seriousness to the vendor")
		} else {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — modest gains from competitive approach", discount)
			lessons = append(lessons, "Competitive negotiation may yield limited results with price-sensitive vendors")
			lessons = append(lessons, "Consider blending competitive tactics with collaborative elements")
		}
	case "collaborative":
		if discount >= 12 {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — collaboration built mutual value", discount)
			lessons = append(lessons, "Collaborative negotiations benefit from multiple rounds of dialogue")
		} else {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — quick collaborative agreement", discount)
			lessons = append(lessons, "Even a short collaborative engagement can produce fair outcomes")
		}
		lessons = append(lessons, "Mutual concessions created a balanced outcome for both parties")
	case "aggressive":
		if discount >= 20 {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — aggressive tactics maximized savings", discount)
			lessons = append(lessons, "Aggressive negotiation can yield maximum discounts when used correctly")
		} else {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — aggressive approach had limited effect", discount)
			lessons = append(lessons, "Aggressive tactics may backfire with vendors who have firm pricing")
		}
		lessons = append(lessons, "Pushing hard early established a low anchor price")
	case "concessionary":
		if discount >= 10 {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — concessions built goodwill", discount)
			lessons = append(lessons, "Making concessions early can build rapport and speed up agreements")
		} else {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — further concessions may have been possible", discount)
			lessons = append(lessons, "Overly concessionary approaches may leave savings on the table")
		}
		lessons = append(lessons, "Flexibility in negotiation helps maintain positive vendor relationships")
	case "principled":
		if discount >= 12 {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — principled negotiation found fair value", discount)
			lessons = append(lessons, "Focusing on objective criteria leads to fair, justifiable outcomes")
		} else {
			outcome = fmt.Sprintf("Deal reached at %.1f%% discount — principled approach prioritized fairness over discounts", discount)
			lessons = append(lessons, "Principled negotiation builds long-term trust even with modest savings")
		}
		lessons = append(lessons, "Separating people from the problem enabled constructive dialogue")
	}

	return outcome, lessons
}
