package summarizer

type SummaryInput struct {
	SessionID string `json:"session_id"`
	Style     string `json:"style"`
}

type SummaryResult struct {
	SessionID string   `json:"session_id"`
	Summary   string   `json:"summary"`
	WordCount int      `json:"word_count"`
	Style     string   `json:"style"`
	KeyPoints []string `json:"key_points"`
}
