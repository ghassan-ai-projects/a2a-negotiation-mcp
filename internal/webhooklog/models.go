package webhooklog

// WebhookEvent represents a logged webhook event.
type WebhookEvent struct {
	ID        int    `json:"id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
	CreatedAt string `json:"created_at"`
}

// WebhookStats aggregates webhook event statistics.
type WebhookStats struct {
	TotalEvents     int            `json:"total_events"`
	SuccessRate     float64        `json:"success_rate"`
	AvgAttempts     float64        `json:"avg_attempts"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
}
