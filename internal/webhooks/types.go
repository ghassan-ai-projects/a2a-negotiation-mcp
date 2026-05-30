package webhooks

import "time"

// Subscription represents a webhook subscription registered by an external system.
type Subscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret"`
	Status    string    `json:"status"` // "active", "disabled"
	CreatedAt time.Time `json:"created_at"`
}

// Event represents a webhook event fired to subscribed endpoints.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}
