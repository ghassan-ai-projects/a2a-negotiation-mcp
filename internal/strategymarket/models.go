package strategymarket

// Strategy represents a community-shared negotiation strategy.
type Strategy struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Config      string  `json:"config"`
	Category    string  `json:"category"`
	Rating      float64 `json:"rating"`
	RatingCount int     `json:"rating_count"`
	CreatedAt   string  `json:"created_at"`
}
