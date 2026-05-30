package contract

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

// ─── Setup ───

func setupTestEngine(t *testing.T) (*Engine, *calendar.Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:contract_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	cstore, err := calendar.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("calendar NewStore: %v", err)
	}

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history NewStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	negEng := negotiation.NewEngine(pstore)
	calEng := calendar.NewEngine(cstore, negEng, hstore, pstore, logger)

	eng := NewEngine(calEng, logger)
	return eng, cstore
}

// ─── ParseContract Tests ───

func TestParseContract_AutoRenewWithDates(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	rawText := "12-month contract starting Jan 1, 2026 ending Dec 31, 2026. Auto-renews unless cancelled 30 days prior."

	result, err := eng.ParseContract(ctx, rawText, "TestVendor", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.EndDate != "2026-12-31" {
		t.Errorf("expected end_date 2026-12-31, got %q", result.Terms.EndDate)
	}
	if result.Terms.StartDate != "2026-01-01" {
		t.Errorf("expected start_date 2026-01-01, got %q", result.Terms.StartDate)
	}
	if !result.Terms.AutoRenew {
		t.Error("expected auto_renew=true")
	}
	if result.Terms.TerminationNotice != 30 {
		t.Errorf("expected termination_notice 30 days, got %d", result.Terms.TerminationNotice)
	}
	if !result.Terms.AnnualContract {
		t.Error("expected annual_contract=true from 12-month term")
	}
	if result.Terms.Vendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", result.Terms.Vendor)
	}
	if result.FieldConf.AutoRenew != "high" {
		t.Errorf("expected auto_renew confidence high, got %s", result.FieldConf.AutoRenew)
	}
	if result.FieldConf.TerminationNotice != "high" {
		t.Errorf("expected termination_notice confidence high, got %s", result.FieldConf.TerminationNotice)
	}
}

func TestParseContract_PriceLock(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	rawText := "Enterprise license agreement — 3 year term, price locked for first 12 months"

	result, err := eng.ParseContract(ctx, rawText, "Vendor", "Enterprise")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.PriceLockPeriod != "12 months" {
		t.Errorf("expected price_lock_period '12 months', got %q", result.Terms.PriceLockPeriod)
	}
	if result.Terms.AnnualContract != true {
		t.Error("expected annual_contract=true for 3 year term")
	}
	if result.Terms.SKU != "Enterprise" {
		t.Errorf("expected SKU Enterprise, got %s", result.Terms.SKU)
	}
	if result.FieldConf.PriceLockPeriod != "high" {
		t.Errorf("expected price_lock confidence high, got %s", result.FieldConf.PriceLockPeriod)
	}
}

func TestParseContract_NoDates(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	rawText := "This is just some ordinary text with no dates or terms."

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.EndDate != "" {
		t.Errorf("expected empty end_date, got %q", result.Terms.EndDate)
	}
	if result.Terms.Confidence != "low" {
		t.Errorf("expected overall confidence low, got %s", result.Terms.Confidence)
	}
	if result.Terms.AutoRenew {
		t.Error("expected auto_renew=false for text with no terms")
	}
	if result.Terms.TerminationNotice != 0 {
		t.Errorf("expected termination_notice 0, got %d", result.Terms.TerminationNotice)
	}
}

func TestParseContract_EmptyText(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	result, err := eng.ParseContract(ctx, "", "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.Confidence != "low" {
		t.Errorf("expected confidence low for empty text, got %s", result.Terms.Confidence)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for empty text")
	}
}

func TestParseContract_WithPricing(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	rawText := "Slack Pro - $8.75 per user per month, 100 users, annual billing, auto-renewal"

	result, err := eng.ParseContract(ctx, rawText, "Slack", "Pro")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.FieldConf.Pricing != "high" {
		t.Errorf("expected pricing confidence high, got %s", result.FieldConf.Pricing)
	}
	if !result.Terms.AutoRenew {
		t.Error("expected auto_renew=true")
	}
	if result.Terms.AnnualContract != true {
		t.Error("expected annual_contract=true for annual billing")
	}
}

func TestParseContract_ISOAndShortDates(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		text     string
		wantEnd  string
		wantStrt string
	}{
		{
			name:     "ISO dates",
			text:     "Contract from 2026-01-15 to 2026-12-31, auto-renew",
			wantEnd:  "2026-12-31",
			wantStrt: "2026-01-15",
		},
		{
			name:     "short dates mm/dd/yyyy",
			text:     "Starts 01/15/2026 ends 12/31/2026, 30 day cancellation",
			wantEnd:  "2026-12-31",
			wantStrt: "2026-01-15",
		},
		{
			name:     "single date",
			text:     "Contract expires December 31, 2026",
			wantEnd:  "2026-12-31",
			wantStrt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.ParseContract(ctx, tt.text, "", "")
			if err != nil {
				t.Fatalf("ParseContract failed: %v", err)
			}
			if result.Terms.EndDate != tt.wantEnd {
				t.Errorf("expected end_date %q, got %q", tt.wantEnd, result.Terms.EndDate)
			}
			if result.Terms.StartDate != tt.wantStrt {
				t.Errorf("expected start_date %q, got %q", tt.wantStrt, result.Terms.StartDate)
			}
		})
	}
}

func TestParseContract_TerminationVariants(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "30 days notice",
			text: "Must provide 30 days notice to cancel. Contract ends 2026-12-31.",
			want: 30,
		},
		{
			name: "60-day notice",
			text: "60-day cancellation notice required. Contract ends 2026-12-31.",
			want: 60,
		},
		{
			name: "cancel within 90 days",
			text: "You may cancel within 90 days of renewal. Contract ends 2026-12-31.",
			want: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.ParseContract(ctx, tt.text, "", "")
			if err != nil {
				t.Fatalf("ParseContract failed: %v", err)
			}
			if result.Terms.TerminationNotice != tt.want {
				t.Errorf("expected termination_notice %d, got %d", tt.want, result.Terms.TerminationNotice)
			}
		})
	}
}

func TestParseContract_DataPortability(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "data portability mentioned",
			text: "Company provides data portability and export options. Contract ends Dec 31, 2026.",
			want: true,
		},
		{
			name: "no portability",
			text: "Standard contract terms. Contract ends Dec 31, 2026.",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.ParseContract(ctx, tt.text, "", "")
			if err != nil {
				t.Fatalf("ParseContract failed: %v", err)
			}
			if result.Terms.DataPortability != tt.want {
				t.Errorf("expected data_portability=%v, got %v", tt.want, result.Terms.DataPortability)
			}
		})
	}
}

func TestParseContract_PriceLockYear(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	rawText := "3 year agreement, price guaranteed for first 2 years. Ends December 31, 2028."

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.PriceLockPeriod != "2 years" {
		t.Errorf("expected price_lock_period '2 years', got %q", result.Terms.PriceLockPeriod)
	}
	if result.Terms.EndDate != "2028-12-31" {
		t.Errorf("expected end_date 2028-12-31, got %q", result.Terms.EndDate)
	}
}

// ─── Warnings Tests ───

func TestParseContract_AmbiguousDates(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Text with ambiguous/missing dates should generate warnings
	rawText := "Annual contract with auto-renewal and 30 day cancellation."

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	// Should have a warning about no dates
	hasDateWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no dates") || strings.Contains(w, "end date") {
			hasDateWarning = true
			break
		}
	}
	if !hasDateWarning {
		t.Errorf("expected warning about missing/ambiguous dates, got warnings: %v", result.Warnings)
	}
}

func TestParseContract_NoticeWarning(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Text without termination notice should generate a warning
	rawText := "Annual contract ending Dec 31, 2026. Auto-renews."

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	hasNoticeWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "termination notice") {
			hasNoticeWarning = true
			break
		}
	}
	if !hasNoticeWarning {
		t.Errorf("expected warning about missing termination notice, got warnings: %v", result.Warnings)
	}
}

// ─── PopulateCalendar Tests ───

func TestPopulateCalendar_CreatesContract(t *testing.T) {
	eng, cstore := setupTestEngine(t)
	ctx := context.Background()

	result := &ContractParseResult{
		RawText: "Test contract ending Dec 31, 2026",
		Terms: ContractTerms{
			Vendor:  "TestVendor",
			SKU:     "TestSKU",
			EndDate: "2026-12-31",
		},
	}

	if err := eng.PopulateCalendar(ctx, result); err != nil {
		t.Fatalf("PopulateCalendar failed: %v", err)
	}

	if !result.AutoPopulated {
		t.Error("expected AutoPopulated=true after calendar population")
	}

	// Verify the contract was created in the calendar store
	contracts, err := cstore.ListContracts(ctx, calendar.ContractFilter{Vendor: "TestVendor"})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].Vendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", contracts[0].Vendor)
	}
	if contracts[0].SKU != "TestSKU" {
		t.Errorf("expected SKU TestSKU, got %s", contracts[0].SKU)
	}
	expectedEnd, _ := time.Parse("2006-01-02", "2026-12-31")
	if !contracts[0].RenewalDate.Equal(expectedEnd) {
		t.Errorf("expected renewal_date %v, got %v", expectedEnd, contracts[0].RenewalDate)
	}
	if contracts[0].Status != "active" {
		t.Errorf("expected status active, got %s", contracts[0].Status)
	}
}

func TestPopulateCalendar_NoEndDate(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	result := &ContractParseResult{
		RawText: "Test contract with no dates",
		Terms: ContractTerms{
			Vendor: "TestVendor",
		},
	}

	err := eng.PopulateCalendar(ctx, result)
	if err == nil {
		t.Fatal("expected error for PopulateCalendar with no end date")
	}
	if result.AutoPopulated {
		t.Error("expected AutoPopulated=false when PopulateCalendar fails")
	}
}

func TestPopulateCalendar_VendorContractCreated(t *testing.T) {
	eng, cstore := setupTestEngine(t)
	ctx := context.Background()

	result := &ContractParseResult{
		RawText: "Slack Enterprise agreement ending 2026-06-30",
		Terms: ContractTerms{
			Vendor:  "Slack",
			SKU:     "Enterprise",
			EndDate: "2026-06-30",
		},
	}

	if err := eng.PopulateCalendar(ctx, result); err != nil {
		t.Fatalf("PopulateCalendar failed: %v", err)
	}

	contracts, err := cstore.ListContracts(ctx, calendar.ContractFilter{Vendor: "Slack"})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 Slack contract, got %d", len(contracts))
	}
	if contracts[0].SKU != "Enterprise" {
		t.Errorf("expected SKU Enterprise, got %s", contracts[0].SKU)
	}
}

// ─── Confidence Edge Cases ───

func TestParseContract_MediumConfidence(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Only has end date (medium), no other high-confidence fields
	rawText := "Contract ends Dec 31, 2026"

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.EndDate != "2026-12-31" {
		t.Errorf("expected end_date 2026-12-31, got %q", result.Terms.EndDate)
	}
}

func TestParseContract_HighConfidence(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Has end date, auto-renew, termination notice, annual term = high confidence
	rawText := "Contract from Jan 1, 2026 to Dec 31, 2026 for $8.75 per seat per month. Auto-renews unless cancelled with 30 days notice."

	result, err := eng.ParseContract(ctx, rawText, "Vendor", "SKU")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.Confidence != "high" {
		t.Errorf("expected overall confidence high, got %s", result.Terms.Confidence)
	}
}

// ─── Helper Tests ───

func TestParseContract_DateOrder(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Dates in reverse order — earliest should be start, latest end
	rawText := "Ends December 31, 2027. Starts January 15, 2026."

	result, err := eng.ParseContract(ctx, rawText, "", "")
	if err != nil {
		t.Fatalf("ParseContract failed: %v", err)
	}

	if result.Terms.StartDate != "2026-01-15" {
		t.Errorf("expected start_date 2026-01-15, got %q", result.Terms.StartDate)
	}
	if result.Terms.EndDate != "2027-12-31" {
		t.Errorf("expected end_date 2027-12-31, got %q", result.Terms.EndDate)
	}
}
