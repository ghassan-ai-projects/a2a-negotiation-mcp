package slacredit

import (
	"log/slog"
	"math"
)

// Engine calculates SLA credit entitlements.
type Engine struct {
	logger *slog.Logger
}

// NewEngine creates a new SLA credit calculator engine.
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{logger: logger}
}

// Calculate determines SLA credit eligibility and amount.
func (e *Engine) Calculate(input *SLACreditInput) *SLACreditOutput {
	output := &SLACreditOutput{
		Vendor:           input.Vendor,
		Service:          input.Service,
		MonthlySpend:     input.MonthlySpend,
		ActualUptime:     input.UptimePct,
		GuaranteedUptime: input.GuaranteedUptime,
		CreditRate:       input.CreditRate,
	}

	if input.UptimePct < input.GuaranteedUptime {
		output.Eligible = true

		// credit_amount = monthly_spend * credit_rate / 100 * (1 - uptime_pct/100) / (1 - guaranteed_uptime/100)
		uptimeRatio := 1 - input.UptimePct/100
		guaranteedRatio := 1 - input.GuaranteedUptime/100

		if guaranteedRatio > 0 {
			raw := input.MonthlySpend * input.CreditRate / 100 * uptimeRatio / guaranteedRatio
			output.CreditAmount = math.Round(raw*100) / 100
		}
	}

	return output
}
