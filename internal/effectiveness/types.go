package effectiveness

// ScoreComponent is a named sub-score that contributes to the overall score.
type ScoreComponent struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`  // 0-100
	Weight float64 `json:"weight"` // e.g. 0.40 for 40%
}

// ScoreTrendPoint is a single data point in the score trend.
type ScoreTrendPoint struct {
	Date  string  `json:"date"`
	Score float64 `json:"score"`
}

// EffectivenessScore is the complete effectiveness score response.
type EffectivenessScore struct {
	UserID      string             `json:"user_id"`
	Period      string             `json:"period"`
	OverallScore float64           `json:"overall_score"` // 0-100
	Components  []ScoreComponent   `json:"components"`
	Trend       []ScoreTrendPoint  `json:"trend"`
	VsAverage   float64            `json:"vs_average"`
	Tips        []string           `json:"tips"`
}
