package slacredit

import (
	"log/slog"
	"os"
	"testing"
)

func TestCalculate_EligibleCredit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(logger)

	input := &SLACreditInput{
		Vendor:           "Slack",
		Service:          "Pro",
		MonthlySpend:     10000,
		UptimePct:        99.0,
		GuaranteedUptime: 99.9,
		CreditRate:       5,
	}

	output := eng.Calculate(input)

	if !output.Eligible {
		t.Error("expected eligible=true for uptime below guarantee")
	}
	if output.CreditAmount <= 0 {
		t.Errorf("expected positive credit_amount, got %.2f", output.CreditAmount)
	}
}

func TestCalculate_NotEligible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(logger)

	input := &SLACreditInput{
		Vendor:           "Slack",
		Service:          "Pro",
		MonthlySpend:     10000,
		UptimePct:        99.95,
		GuaranteedUptime: 99.9,
		CreditRate:       5,
	}

	output := eng.Calculate(input)

	if output.Eligible {
		t.Error("expected eligible=false for uptime above guarantee")
	}
	if output.CreditAmount != 0 {
		t.Errorf("expected credit_amount=0, got %.2f", output.CreditAmount)
	}
}

func TestCalculate_ExactMath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(logger)

	// monthly_spend=10000, credit_rate=5, uptime=99.0, guaranteed=99.9
	// credit_amount = 10000 * 5/100 * (1 - 99/100) / (1 - 99.9/100)
	//              = 10000 * 0.05 * 0.01 / 0.001
	//              = 10000 * 0.05 * 10
	//              = 5000
	input := &SLACreditInput{
		Vendor:           "Acme",
		Service:          "Premium",
		MonthlySpend:     10000,
		UptimePct:        99.0,
		GuaranteedUptime: 99.9,
		CreditRate:       5,
	}

	output := eng.Calculate(input)

	if !output.Eligible {
		t.Error("expected eligible=true")
	}
	expected := 5000.00
	if output.CreditAmount != expected {
		t.Errorf("expected credit_amount=%.2f, got %.2f", expected, output.CreditAmount)
	}
}
