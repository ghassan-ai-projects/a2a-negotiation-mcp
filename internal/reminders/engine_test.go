package reminders

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEngine_CheckRenewals_Mixed(t *testing.T) {
	now := time.Now()
	contractsFn := func(ctx context.Context, daysAhead int) ([]ContractRow, error) {
		return []ContractRow{
			{ID: "c1", Vendor: "Slack", SKU: "Pro", RenewalDate: now.AddDate(0, 0, 3).Format(time.DateOnly)},    // 3 days → critical
			{ID: "c2", Vendor: "GitHub", SKU: "Team", RenewalDate: now.AddDate(0, 0, 15).Format(time.DateOnly)}, // 15 days → soon
			{ID: "c3", Vendor: "AWS", SKU: "Basic", RenewalDate: now.AddDate(0, 0, 45).Format(time.DateOnly)},   // 45 days → upcoming
		}, nil
	}

	eng := NewEngine(contractsFn, testLogger())
	result, err := eng.CheckRenewals(context.Background())
	if err != nil {
		t.Fatalf("CheckRenewals: %v", err)
	}

	if len(result.Critical) != 1 {
		t.Errorf("critical = %d, want 1", len(result.Critical))
	}
	if len(result.Soon) != 1 {
		t.Errorf("soon = %d, want 1", len(result.Soon))
	}
	if len(result.Upcoming) != 1 {
		t.Errorf("upcoming = %d, want 1", len(result.Upcoming))
	}

	if len(result.Critical) > 0 && result.Critical[0].Vendor != "Slack" {
		t.Errorf("critical vendor = %s, want Slack", result.Critical[0].Vendor)
	}
	if len(result.Soon) > 0 && result.Soon[0].Vendor != "GitHub" {
		t.Errorf("soon vendor = %s, want GitHub", result.Soon[0].Vendor)
	}
	if len(result.Upcoming) > 0 && result.Upcoming[0].Vendor != "AWS" {
		t.Errorf("upcoming vendor = %s, want AWS", result.Upcoming[0].Vendor)
	}
}

func TestEngine_CheckRenewals_NoContracts(t *testing.T) {
	contractsFn := func(ctx context.Context, daysAhead int) ([]ContractRow, error) {
		return nil, nil
	}

	eng := NewEngine(contractsFn, testLogger())
	result, err := eng.CheckRenewals(context.Background())
	if err != nil {
		t.Fatalf("CheckRenewals: %v", err)
	}

	if len(result.Critical) != 0 {
		t.Errorf("critical = %d, want 0", len(result.Critical))
	}
	if len(result.Soon) != 0 {
		t.Errorf("soon = %d, want 0", len(result.Soon))
	}
	if len(result.Upcoming) != 0 {
		t.Errorf("upcoming = %d, want 0", len(result.Upcoming))
	}
}

func TestEngine_CheckRenewals_PastDue(t *testing.T) {
	now := time.Now()
	contractsFn := func(ctx context.Context, daysAhead int) ([]ContractRow, error) {
		return []ContractRow{
			{ID: "c1", Vendor: "Slack", SKU: "Pro", RenewalDate: now.AddDate(0, 0, -5).Format(time.DateOnly)}, // past due = 0 days
		}, nil
	}

	eng := NewEngine(contractsFn, testLogger())
	result, err := eng.CheckRenewals(context.Background())
	if err != nil {
		t.Fatalf("CheckRenewals: %v", err)
	}

	if len(result.Critical) != 1 {
		t.Fatalf("critical = %d, want 1", len(result.Critical))
	}
	if result.Critical[0].DaysUntil != 0 {
		t.Errorf("days_until = %d, want 0", result.Critical[0].DaysUntil)
	}
}
