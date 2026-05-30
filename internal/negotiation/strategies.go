package negotiation

// Strategy represents a negotiation strategy profile.
type Strategy struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	Aggressiveness       string  `json:"aggressiveness"`
	IdealFor             string  `json:"ideal_for"`
	InitialDiscountPct   float64 `json:"initial_discount_pct"`
	MaxConcessions       int     `json:"max_concessions"`
	ConcessionPerRound   float64 `json:"concession_per_round"`
	WalkAwayThresholdPct float64 `json:"walk_away_threshold_pct"`
	MaxRounds            int     `json:"max_rounds"`
}

// AvailableStrategies returns the built-in strategy profiles.
func AvailableStrategies() map[string]Strategy {
	return map[string]Strategy{
		"aggressive": {
			Name:                 "aggressive",
			Description:          "Start at 30% below asking, concede slowly over 4-5 rounds. Ideal when you have strong alternatives or leverage.",
			Aggressiveness:       "high",
			IdealFor:             "Competitive markets, multiple vendors, large volume commitments",
			InitialDiscountPct:   0.30,
			MaxConcessions:       4,
			ConcessionPerRound:   0.03,
			WalkAwayThresholdPct: 0.45,
			MaxRounds:            5,
		},
		"balanced": {
			Name:                 "balanced",
			Description:          "Start at 20% below asking, 3-4 rounds with moderate concessions. Default approach for most negotiations.",
			Aggressiveness:       "medium",
			IdealFor:             "Standard enterprise SaaS procurement, renewal negotiations",
			InitialDiscountPct:   0.20,
			MaxConcessions:       3,
			ConcessionPerRound:   0.04,
			WalkAwayThresholdPct: 0.35,
			MaxRounds:            4,
		},
		"conservative": {
			Name:                 "conservative",
			Description:          "Start at 10% below asking, 2-3 rounds, quick to accept. Best for critical vendors with few alternatives.",
			Aggressiveness:       "low",
			IdealFor:             "Mission-critical vendors, sole-source suppliers, renewals with strong relationships",
			InitialDiscountPct:   0.10,
			MaxConcessions:       2,
			ConcessionPerRound:   0.05,
			WalkAwayThresholdPct: 0.20,
			MaxRounds:            3,
		},
	}
}

// GetStrategy returns a strategy by name, or nil if not found.
func GetStrategy(name string) *Strategy {
	s, ok := AvailableStrategies()[name]
	if !ok {
		return nil
	}
	return &s
}

// StrategyInfo is a public-safe representation of a strategy.
type StrategyInfo struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Aggressiveness string `json:"aggressiveness"`
	IdealFor       string `json:"ideal_for"`
}
