package notify

import "time"

// NotificationPreferences represents a user's notification settings.
type NotificationPreferences struct {
	UserID       string   `json:"user_id"`
	Channel      string   `json:"channel"`       // "slack" or "webhook"
	EnabledTypes []string `json:"enabled_types"`  // e.g. ["deal_closed", "renewal", "alert"]
	DigestFreq   string   `json:"digest_frequency"` // "daily", "weekly", "never"
	WebhookURL   string   `json:"webhook_url,omitempty"`
}

// Notification represents a single sent notification.
type Notification struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	Message   string    `json:"message"`
	Priority  string    `json:"priority"`  // "low", "normal", "high", "urgent"
	Status    string    `json:"status"`    // "pending", "sent", "failed"
	CreatedAt time.Time `json:"created_at"`
}

// sendRequest is the payload sent to a webhook URL.
type sendRequest struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
}
