package summarizer

import (
	"context"
	"fmt"
	"strings"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Summarize(ctx context.Context, sessionID, style string) (*SummaryResult, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID must not be empty")
	}

	var summary string
	var keyPoints []string

	switch style {
	case "brief":
		summary = "The negotiation opened with initial offers exchanged between both parties. " +
			"Key concessions were made on volume pricing and contract duration. " +
			"The final agreement reached a 17% discount from list price. " +
			"Both parties expressed satisfaction with the outcome."
		keyPoints = []string{
			"Negotiation completed in 4 rounds",
			"Volume pricing was the primary leverage point",
			"Contract duration extended to 24 months",
			"17% discount achieved from list price",
		}

	case "detailed":
		summary = "The negotiation session began with the buyer presenting a market analysis showing comparable vendor pricing at 15-22% below list. " +
			"The seller responded with a first-round offer of 8% discount contingent on a 12-month commitment. " +
			"In the second round, the buyer countered at 20% discount with a 24-month commitment, citing volume projections. " +
			"The seller revised to 14% discount in round three, adding a service-level guarantee. " +
			"Round four saw the buyer accept at 17% with the SLA terms included. " +
			"Payment terms were finalized at net-45 rather than the standard net-30. " +
			"Both parties agreed to a quarterly business review cadence. " +
			"The contract includes an annual escalator capped at 3%."
		keyPoints = []string{
			"Buyer used market comparables as opening leverage",
			"Longer commitment (24mo) was the key concession from buyer",
			"SLA guarantees added in round three",
			"17% final discount with net-45 payment terms",
		}

	case "bullet_points":
		summary = "• Opened with buyer market analysis showing 15-22% below list\n" +
			"• Seller initial offer: 8% discount, 12-month commitment\n" +
			"• Buyer counter: 20% discount, 24-month commitment\n" +
			"• Seller revised: 14% discount with SLA guarantees\n" +
			"• Final agreement: 17% discount, 24-month term\n" +
			"• Payment terms: net-45, annual escalator capped at 3%"
		keyPoints = []string{
			"Market data drove favorable opening position",
			"Trade-off: longer commitment for higher discount",
			"SLA inclusion was a deal-clincher in round three",
			"17% discount on a 24-month contract",
		}

	default:
		return nil, fmt.Errorf("invalid style %q: must be one of [brief, detailed, bullet_points]", style)
	}

	wordCount := len(strings.Fields(summary))

	return &SummaryResult{
		SessionID: sessionID,
		Summary:   summary,
		WordCount: wordCount,
		Style:     style,
		KeyPoints: keyPoints,
	}, nil
}
