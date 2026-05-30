package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Block represents a Slack Block Kit block element.
type Block struct {
	Type   string  `json:"type"`
	Text   *Text   `json:"text,omitempty"`
	Fields []*Text `json:"fields,omitempty"`
}

// Text represents a Slack Block Kit text object.
type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// webhookPayload is the top-level JSON payload sent to a Slack webhook.
type webhookPayload struct {
	Blocks []Block `json:"blocks"`
}

// Client sends messages to Slack via incoming webhooks.
type Client struct {
	webhookURL string
	logger     *slog.Logger
	enabled    bool
	mu         sync.Mutex
	lastSent   time.Time
}

// NewClient creates a new Slack webhook client. When webhookURL is empty,
// the client is disabled and all Send calls are no-ops.
func NewClient(webhookURL string, logger *slog.Logger) *Client {
	return &Client{
		webhookURL: webhookURL,
		logger:     logger,
		enabled:    webhookURL != "",
	}
}

// Enabled returns true if the client is configured to send messages.
func (c *Client) Enabled() bool {
	return c.enabled
}

// LastSent returns the timestamp of the last successful message send.
func (c *Client) LastSent() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSent
}

// Send sends a Slack message via incoming webhook using Block Kit format.
// Returns nil immediately (no-op) when the client is not enabled.
func (c *Client) Send(ctx context.Context, blocks []Block) error {
	if !c.enabled {
		return nil
	}

	payload := webhookPayload{Blocks: blocks}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	c.mu.Lock()
	c.lastSent = time.Now()
	c.mu.Unlock()

	c.logger.Debug("slack message sent",
		"blocks", len(blocks),
		"status", resp.StatusCode,
	)
	return nil
}
