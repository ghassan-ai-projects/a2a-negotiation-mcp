package slack

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
)

// validateBlocksJSON marshals the blocks to JSON and unmarshals back to verify
// it produces valid Slack Block Kit JSON.
func validateBlocksJSON(t *testing.T, blocks []Block) {
	t.Helper()

	data, err := json.Marshal(blocks)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded []Block
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v\nraw: %s", err, string(data))
	}

	if len(decoded) != len(blocks) {
		t.Fatalf("expected %d blocks, got %d", len(blocks), len(decoded))
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRenewalAlert_ValidJSON(t *testing.T) {
	contract := calendar.Contract{
		Vendor:       "Salesforce",
		SKU:          "Enterprise",
		Seats:        50,
		CurrentPrice: 165.00,
		RenewalDate:  time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Status:       "active",
	}

	blocks := RenewalAlert(contract, 12000, 30)
	validateBlocksJSON(t, blocks)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "header" {
		t.Errorf("block[0].type = %q, want header", blocks[0].Type)
	}
	if blocks[0].Text.Type != "plain_text" {
		t.Errorf("block[0].text.type = %q, want plain_text", blocks[0].Text.Type)
	}
	if blocks[0].Text.Text != "📋 Renewal Alert" {
		t.Errorf("block[0].text.text = %q, want 📋 Renewal Alert", blocks[0].Text.Text)
	}

	if blocks[1].Type != "section" {
		t.Errorf("block[1].type = %q, want section", blocks[1].Type)
	}
	if blocks[1].Text.Type != "mrkdwn" {
		t.Errorf("block[1].text.type = %q, want mrkdwn", blocks[1].Text.Type)
	}
}

func TestRenewalAlert_ZeroSeats(t *testing.T) {
	contract := calendar.Contract{
		Vendor:       "GitHub",
		Seats:        0,
		CurrentPrice: 4.00,
		RenewalDate:  time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	}
	blocks := RenewalAlert(contract, 5000, 45)
	validateBlocksJSON(t, blocks)
}

func TestNegotiationResult_Accepted_ValidJSON(t *testing.T) {
	outcome := learning.StrategyOutcome{
		Vendor:         "Slack",
		Strategy:       "balanced",
		DiscountPct:    0.22,
		RoundsComplete: 4,
		Outcome:        "accepted",
		TotalBefore:    8.75,
		TotalAfter:     6.50,
		Timestamp:      time.Now().UTC(),
	}

	blocks := NegotiationResult(outcome)
	validateBlocksJSON(t, blocks)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "header" {
		t.Errorf("block[0].type = %q, want header", blocks[0].Type)
	}
	if blocks[0].Text.Text != "🤝 Negotiation Complete!" {
		t.Errorf("block[0].text.text = %q, want 🤝 Negotiation Complete!", blocks[0].Text.Text)
	}

	details := blocks[2]
	if details.Type != "section" {
		t.Errorf("block[2].type = %q, want section", details.Type)
	}
	if len(details.Fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(details.Fields))
	}
}

func TestNegotiationResult_WalkedAway_ValidJSON(t *testing.T) {
	outcome := learning.StrategyOutcome{
		Vendor:         "Salesforce",
		Strategy:       "aggressive",
		DiscountPct:    0.30,
		RoundsComplete: 5,
		Outcome:        "walked_away",
		TotalBefore:    165.00,
		TotalAfter:     115.00,
		Timestamp:      time.Now().UTC(),
	}

	blocks := NegotiationResult(outcome)
	validateBlocksJSON(t, blocks)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
}

func TestNegotiationResult_ZeroValues(t *testing.T) {
	outcome := learning.StrategyOutcome{
		Vendor:         "TestVendor",
		Strategy:       "conservative",
		DiscountPct:    0,
		RoundsComplete: 0,
		Outcome:        "accepted",
		TotalBefore:    0,
		TotalAfter:     0,
	}

	blocks := NegotiationResult(outcome)
	validateBlocksJSON(t, blocks)
}

func TestSavingsReport_ValidJSON(t *testing.T) {
	blocks := SavingsReport(24000, 3, "Salesforce", 22)
	validateBlocksJSON(t, blocks)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "header" {
		t.Errorf("block[0].type = %q, want header", blocks[0].Type)
	}
	if blocks[0].Text.Text != "📊 Monthly Savings Report" {
		t.Errorf("block[0].text.text = %q, want 📊 Monthly Savings Report", blocks[0].Text.Text)
	}
	if blocks[1].Type != "section" {
		t.Errorf("block[1].type = %q, want section", blocks[1].Type)
	}
}

func TestSavingsReport_ZeroDeals(t *testing.T) {
	blocks := SavingsReport(0, 0, "None", 0)
	validateBlocksJSON(t, blocks)
}

func TestClient_Disabled(t *testing.T) {
	logger := testLogger()
	c := NewClient("", logger)
	if c.Enabled() {
		t.Error("expected disabled client")
	}
	if err := c.Send(nil, nil); err != nil {
		t.Errorf("disabled client should no-op: %v", err)
	}
}

func TestClient_Enabled(t *testing.T) {
	logger := testLogger()
	c := NewClient("https://hooks.slack.com/test", logger)
	if !c.Enabled() {
		t.Error("expected enabled client")
	}
}
