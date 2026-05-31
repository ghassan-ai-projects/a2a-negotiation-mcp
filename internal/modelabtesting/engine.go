package modelabtesting

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
)

// Engine performs stateless model A/B test simulations.
type Engine struct{}

// NewEngine creates a new A/B test engine.
func NewEngine() *Engine {
	return &Engine{}
}

// RunABTest simulates a negotiation for each model and returns a comparison.
func (e *Engine) RunABTest(ctx context.Context, input ABTestInput) (*ABTestResult, error) {
	if input.ModelA == "" {
		return nil, fmt.Errorf("model_a is required")
	}
	if input.ModelB == "" {
		return nil, fmt.Errorf("model_b is required")
	}
	if input.ModelA == input.ModelB {
		return nil, fmt.Errorf("model_a and model_b must be different")
	}
	if input.ScenarioID == "" {
		return nil, fmt.Errorf("scenario_id is required")
	}

	// Seed a deterministic RNG from the scenario ID hash.
	h := fnv.New64a()
	_, _ = h.Write([]byte(input.ScenarioID))
	seed := int64(h.Sum64())
	rng := rand.New(rand.NewSource(seed))

	// Model A: savings 10-25%, Model B: savings 8-22%
	modelAResult := simulateModel(rng, input.ModelA, 10.0, 25.0)
	modelBResult := simulateModel(rng, input.ModelB, 8.0, 22.0)

	savingsDiff := math.Abs(modelAResult.SavingsPct - modelBResult.SavingsPct)

	var winner string
	switch {
	case modelAResult.SavingsPct > modelBResult.SavingsPct && savingsDiff >= 2.0:
		winner = input.ModelA
	case modelBResult.SavingsPct > modelAResult.SavingsPct && savingsDiff >= 2.0:
		winner = input.ModelB
	default:
		winner = "draw"
	}

	rec := buildRecommendation(input.ModelA, input.ModelB, modelAResult.SavingsPct, modelBResult.SavingsPct, savingsDiff)

	return &ABTestResult{
		ModelAResult:   modelAResult,
		ModelBResult:   modelBResult,
		Winner:         winner,
		SavingsDiff:    savingsDiff,
		Recommendation: rec,
	}, nil
}

func simulateModel(rng *rand.Rand, model string, minSavings, maxSavings float64) ModelResult {
	savingsPct := minSavings + rng.Float64()*(maxSavings-minSavings)
	durationRounds := 3 + rng.Intn(8) // 3-10 rounds
	finalOffer := 100.0 - savingsPct

	// Pick aggressiveness based on savings bucket.
	agg := "moderate"
	if savingsPct > 20 {
		agg = "aggressive"
	} else if savingsPct < 13 {
		agg = "conservative"
	}

	return ModelResult{
		Model:          model,
		SavingsPct:     math.Round(savingsPct*100) / 100,
		DurationRounds: durationRounds,
		Aggressiveness: agg,
		FinalOffer:     math.Round(finalOffer*100) / 100,
	}
}

func buildRecommendation(modelA, modelB string, savingsA, savingsB, diff float64) string {
	switch {
	case diff < 2.0:
		return fmt.Sprintf(
			"Both %s (%.2f%%) and %s (%.2f%%) performed similarly (Δ %.2f%%). Consider cost or other qualitative factors.",
			modelA, savingsA, modelB, savingsB, diff,
		)
	case savingsA > savingsB:
		return fmt.Sprintf(
			"%s outperformed %s with %.2f%% savings vs %.2f%% (Δ %.2f%%). Recommend %s for scenario.",
			modelA, modelB, savingsA, savingsB, diff, modelA,
		)
	default:
		return fmt.Sprintf(
			"%s outperformed %s with %.2f%% savings vs %.2f%% (Δ %.2f%%). Recommend %s for scenario.",
			modelB, modelA, savingsB, savingsA, diff, modelB,
		)
	}
}
