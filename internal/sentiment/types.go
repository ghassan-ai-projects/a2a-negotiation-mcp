package sentiment

// SentimentResult holds the outcome of a sentiment analysis on vendor communication text.
type SentimentResult struct {
	Score      float64  `json:"score"`
	Confidence float64  `json:"confidence"`
	Label      string   `json:"label"`
	KeyPhrases []string `json:"key_phrases"`
}
