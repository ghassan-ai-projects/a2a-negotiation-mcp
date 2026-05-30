package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Engine manages notification preferences and sending.
type Engine struct {
	store  *Store
	logger *slog.Logger
	client *http.Client
}

// NewEngine creates a notification engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SetPreferences saves notification preferences for a user.
func (e *Engine) SetPreferences(ctx context.Context, channel string, enabledTypes []string, digestFreq, webhookURL string) (*NotificationPreferences, error) {
	e.logger.Debug("set_preferences", "channel", channel, "enabled_types", enabledTypes)

	if digestFreq == "" {
		digestFreq = "never"
	}
	if digestFreq != "daily" && digestFreq != "weekly" && digestFreq != "never" {
		return nil, fmt.Errorf("invalid digest_frequency %q: use daily, weekly, or never", digestFreq)
	}
	if channel != "slack" && channel != "webhook" {
		return nil, fmt.Errorf("invalid channel %q: use slack or webhook", channel)
	}
	if channel == "webhook" && webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required for webhook channel")
	}

	prefs := &NotificationPreferences{
		UserID:       "default",
		Channel:      channel,
		EnabledTypes: enabledTypes,
		DigestFreq:   digestFreq,
		WebhookURL:   webhookURL,
	}

	if err := e.store.SetPreferences(ctx, prefs); err != nil {
		return nil, fmt.Errorf("save preferences: %w", err)
	}

	return prefs, nil
}

// GetPreferences retrieves the current notification preferences for the default user.
func (e *Engine) GetPreferences(ctx context.Context) (*NotificationPreferences, error) {
	e.logger.Debug("get_preferences")

	// Try webhook first, fall back to slack
	prefs, err := e.store.GetPreferences(ctx, "default", "webhook")
	if err != nil {
		return nil, fmt.Errorf("get webhook prefs: %w", err)
	}
	if prefs != nil {
		return prefs, nil
	}

	prefs, err = e.store.GetPreferences(ctx, "default", "slack")
	if err != nil {
		return nil, fmt.Errorf("get slack prefs: %w", err)
	}
	if prefs == nil {
		// Return defaults
		return &NotificationPreferences{
			UserID:       "default",
			Channel:      "webhook",
			EnabledTypes: []string{},
			DigestFreq:   "never",
		}, nil
	}
	return prefs, nil
}

// SendNotification sends a notification via the user's configured channel.
func (e *Engine) SendNotification(ctx context.Context, notifType, message, priority string) (*Notification, error) {
	e.logger.Debug("send_notification", "type", notifType, "priority", priority)

	if priority == "" {
		priority = "normal"
	}

	// Get preferences to know which channel to use
	prefs, err := e.GetPreferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	n := &Notification{
		UserID:    "default",
		Type:      notifType,
		Channel:   prefs.Channel,
		Message:   message,
		Priority:  priority,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	// Send via configured channel
	switch prefs.Channel {
	case "webhook":
		if prefs.WebhookURL == "" {
			n.Status = "failed"
			e.logger.Warn("no webhook URL configured, notification not sent")
		} else {
			if err := e.sendWebhook(ctx, prefs.WebhookURL, n); err != nil {
				n.Status = "failed"
				e.logger.Warn("webhook send failed", "error", err.Error())
			} else {
				n.Status = "sent"
			}
		}
	case "slack":
		// Slack sending would go through the Slack client
		// For now, log and mark as sent
		n.Status = "sent"
		e.logger.Debug("slack notification", "message", message)
	default:
		n.Status = "sent"
	}

	// Log the notification
	id, err := e.store.LogNotification(ctx, n)
	if err != nil {
		e.logger.Warn("failed to log notification", "error", err.Error())
	}
	n.ID = id

	return n, nil
}

// sendWebhook sends a notification payload to a webhook URL.
func (e *Engine) sendWebhook(ctx context.Context, url string, n *Notification) error {
	payload := sendRequest{
		Type:     n.Type,
		Message:  n.Message,
		Priority: n.Priority,
		Status:   n.Status,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
