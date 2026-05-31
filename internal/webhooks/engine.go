package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Engine manages webhook subscriptions and dispatches events to subscribers.
type Engine struct {
	store      *Store
	httpClient *http.Client
	logger     *slog.Logger
	backoffs   []time.Duration
}

// defaultBackoffs are the standard retry delays for webhook delivery.
var defaultBackoffs = []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second}

// NewEngine creates a new webhook engine with default backoff durations.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:      store,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		backoffs:   defaultBackoffs,
	}
}

// NewEngineWithBackoff creates a webhook engine with custom backoff durations.
func NewEngineWithBackoff(store *Store, logger *slog.Logger, backoffs []time.Duration) *Engine {
	return &Engine{
		store:      store,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		backoffs:   backoffs,
	}
}

// Register adds a new webhook subscription. Returns the created subscription.
func (e *Engine) Register(ctx context.Context, url string, events []string, secret string) (*Subscription, error) {
	if url == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("at least one event type is required")
	}

	sub := &Subscription{
		URL:    url,
		Events: events,
		Secret: secret,
	}

	if err := e.store.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("register webhook: %w", err)
	}

	e.logger.Info("webhook registered",
		"id", sub.ID,
		"url", url,
		"events", events,
	)
	return sub, nil
}

// Unregister disables a webhook subscription (soft delete).
func (e *Engine) Unregister(ctx context.Context, id string) error {
	if err := e.store.Disable(ctx, id); err != nil {
		return fmt.Errorf("unregister webhook: %w", err)
	}

	e.logger.Info("webhook unregistered", "id", id)
	return nil
}

// List returns all active subscriptions.
func (e *Engine) List(ctx context.Context) ([]Subscription, error) {
	return e.store.List(ctx, "active")
}

// Dispatch fires an event to all subscribed webhooks.
// POSTs JSON payload with HMAC-SHA256 signature in X-Webhook-Signature header.
// Retries: 3 attempts with backoff (1s, 5s, 30s).
func (e *Engine) Dispatch(ctx context.Context, eventType string, data any) error {
	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	subs, err := e.store.GetByEvent(ctx, eventType)
	if err != nil {
		return fmt.Errorf("get subscribers: %w", err)
	}

	if len(subs) == 0 {
		e.logger.Debug("no subscribers for event", "event_type", eventType)
		return nil
	}

	e.logger.Info("dispatching event",
		"event_type", eventType,
		"event_id", event.ID,
		"subscribers", len(subs),
	)

	var lastErr error
	for _, sub := range subs {
		if err := e.deliverWithRetry(ctx, sub, body); err != nil {
			e.logger.Error("webhook delivery failed",
				"webhook_id", sub.ID,
				"url", sub.URL,
				"error", err.Error(),
			)
			lastErr = err
		}
	}

	return lastErr
}

// deliverWithRetry sends the webhook payload with retry and backoff.
func (e *Engine) deliverWithRetry(ctx context.Context, sub Subscription, body []byte) error {
	var lastErr error
	for attempt := 0; attempt <= len(e.backoffs); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(e.backoffs[attempt-1]):
			}
		}

		if err := e.deliver(ctx, sub, body); err != nil {
			lastErr = err
			e.logger.Warn("webhook delivery attempt failed",
				"url", sub.URL,
				"attempt", attempt+1,
				"error", err.Error(),
			)
			continue
		}

		return nil
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", len(e.backoffs)+1, lastErr)
}

// deliver sends a single webhook POST request with HMAC signature.
func (e *Engine) deliver(ctx context.Context, sub Subscription, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	e.logger.Debug("webhook delivered",
		"url", sub.URL,
		"status", resp.StatusCode,
	)
	return nil
}
