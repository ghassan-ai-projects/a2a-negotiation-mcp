package training

// TrainingSession represents a saved training configuration.
type TrainingSession struct {
	ID        string  `json:"id"`
	Vendor    string  `json:"vendor"`
	Strategy  string  `json:"strategy"`
	Budget    float64 `json:"budget"`
	Rounds    int     `json:"rounds"`
	CreatedAt string  `json:"created_at"`
}

// SimulationRound represents a single round of the simulation.
type SimulationRound struct {
	RoundNumber  int     `json:"round_number"`
	Offer        float64 `json:"offer"`
	DiscountPct  float64 `json:"discount_percentage"`
	Counterparty string  `json:"counterparty"`
	Note         string  `json:"note"`
}

// SimulationResult holds the complete outcome of a simulation run.
type SimulationResult struct {
	ID            string            `json:"id"`
	Vendor        string            `json:"vendor"`
	Strategy      string            `json:"strategy"`
	Budget        float64           `json:"budget"`
	TotalRounds   int               `json:"total_rounds"`
	Rounds        []SimulationRound `json:"rounds"`
	FinalOutcome  string            `json:"final_outcome"`
	TotalDiscount float64           `json:"total_discount_pct"`
	Lessons       []string          `json:"lessons"`
}
