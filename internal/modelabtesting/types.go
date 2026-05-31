package modelabtesting

// ABTestInput represents the parameters for a model A/B test.
type ABTestInput struct {
	ModelA     string `json:"model_a"`
	ModelB     string `json:"model_b"`
	ScenarioID string `json:"scenario_id"`
}

// ModelResult represents the outcome of a simulated negotiation for one model.
type ModelResult struct {
	Model          string  `json:"model"`
	SavingsPct     float64 `json:"savings_pct"`
	DurationRounds int     `json:"duration_rounds"`
	Aggressiveness string  `json:"aggressiveness"`
	FinalOffer     float64 `json:"final_offer"`
}

// ABTestResult holds the full A/B test comparison result.
type ABTestResult struct {
	ModelAResult   ModelResult `json:"model_a_result"`
	ModelBResult   ModelResult `json:"model_b_result"`
	Winner         string      `json:"winner"`
	SavingsDiff    float64     `json:"savings_difference_pct"`
	Recommendation string      `json:"recommendation"`
}
