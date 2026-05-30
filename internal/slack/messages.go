package slack

import (
	"fmt"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
)

// RenewalAlert builds a Slack Block Kit message for an upcoming contract renewal.
//
// Example:
//
//	📋 Renewal Alert
//	Your Salesforce (50 seats) renews in 30 days. Savings: ~$12K
func RenewalAlert(contract calendar.Contract, savings float64, daysUntil int) []Block {
	header := Block{
		Type: "header",
		Text: &Text{Type: "plain_text", Text: "📋 Renewal Alert"},
	}

	detailText := fmt.Sprintf(
		"Your *%s* (%d seats) renews in *%d days*. Estimated savings: ~$%.0f",
		contract.Vendor, contract.Seats, daysUntil, savings,
	)
	detail := Block{
		Type: "section",
		Text: &Text{Type: "mrkdwn", Text: detailText},
	}

	priceText := fmt.Sprintf(
		"• *Current:* $%.2f/seat\n• *Renewal:* %s",
		contract.CurrentPrice, contract.RenewalDate.Format("Jan 2, 2006"),
	)
	priceSection := Block{
		Type:   "section",
		Fields: []*Text{{Type: "mrkdwn", Text: priceText}},
	}

	return []Block{header, detail, priceSection}
}

// NegotiationResult builds a Slack Block Kit message for a completed negotiation.
//
// Example:
//
//	🤝 Negotiation Complete!
//	Vendor: Salesforce | Discount: 22%
//	From: $165/seat → To: $128/seat
//	Savings: $22,200/year | Strategy: Balanced (4 rounds)
func NegotiationResult(outcome learning.StrategyOutcome) []Block {
	header := Block{
		Type: "header",
		Text: &Text{Type: "plain_text", Text: "🤝 Negotiation Complete!"},
	}

	emoji := "✅"
	if outcome.Outcome == "walked_away" {
		emoji = "🚶"
	}

	titleText := fmt.Sprintf("%s *%s* — %s", emoji, outcome.Vendor, outcome.Outcome)
	title := Block{
		Type: "section",
		Text: &Text{Type: "mrkdwn", Text: titleText},
	}

	discountPct := outcome.DiscountPct * 100
	savings := outcome.TotalBefore - outcome.TotalAfter

	fields := []*Text{
		{Type: "mrkdwn", Text: fmt.Sprintf("*Vendor:* %s", outcome.Vendor)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Discount:* %.0f%%", discountPct)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*From:* $%.2f/seat", outcome.TotalBefore)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*To:* $%.2f/seat", outcome.TotalAfter)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Savings:* $%.0f/year", savings)},
		{Type: "mrkdwn", Text: fmt.Sprintf("*Strategy:* %s (%d rounds)", outcome.Strategy, outcome.RoundsComplete)},
	}

	details := Block{
		Type:   "section",
		Fields: fields,
	}

	return []Block{header, title, details}
}

// SavingsReport builds a Slack Block Kit monthly savings summary.
//
// Example:
//
//	📊 Monthly Savings Report
//	March: 3 deals, $24K saved. Best: Salesforce 22% off.
func SavingsReport(totalSavings float64, deals int, bestDeal string, bestDiscount float64) []Block {
	header := Block{
		Type: "header",
		Text: &Text{Type: "plain_text", Text: "📊 Monthly Savings Report"},
	}

	summaryText := fmt.Sprintf(
		"This month: *%d deals* closed, *$%.0f* saved total. Best deal: *%s* at *%.0f%%* off.",
		deals, totalSavings, bestDeal, bestDiscount,
	)
	summary := Block{
		Type: "section",
		Text: &Text{Type: "mrkdwn", Text: summaryText},
	}

	return []Block{header, summary}
}
