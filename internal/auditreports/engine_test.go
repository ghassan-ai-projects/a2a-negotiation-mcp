package auditreports

import (
	"context"
	"strings"
	"testing"
)

func TestGenerateReport_ValidJSON(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	report, err := eng.GenerateReport(ctx, "2026-01-01", "2026-03-31", "json")
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.PeriodFrom != "2026-01-01" {
		t.Errorf("expected PeriodFrom 2026-01-01, got %s", report.PeriodFrom)
	}
	if report.PeriodTo != "2026-03-31" {
		t.Errorf("expected PeriodTo 2026-03-31, got %s", report.PeriodTo)
	}
	if report.Format != "json" {
		t.Errorf("expected format json, got %s", report.Format)
	}
	if report.RowCount <= 0 {
		t.Errorf("expected positive RowCount, got %d", report.RowCount)
	}
	if !strings.HasPrefix(report.Data, "[") {
		t.Errorf("expected JSON array, got %s", report.Data[:1])
	}
	if report.Summary.TotalNegotiations != report.RowCount {
		t.Errorf("expected TotalNegotiations %d == RowCount %d", report.Summary.TotalNegotiations, report.RowCount)
	}
	if report.Summary.TotalSavings <= 0 {
		t.Errorf("expected positive TotalSavings, got %f", report.Summary.TotalSavings)
	}
	if report.Summary.PeriodDescription == "" {
		t.Error("expected non-empty PeriodDescription")
	}
}

func TestGenerateReport_ValidCSV(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	report, err := eng.GenerateReport(ctx, "2026-01-01", "2026-03-31", "csv")
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.Format != "csv" {
		t.Errorf("expected format csv, got %s", report.Format)
	}
	if !strings.HasPrefix(report.Data, "vendor") {
		t.Errorf("expected CSV header, got %s", report.Data[:6])
	}
	lines := strings.Split(strings.TrimSpace(report.Data), "\n")
	if len(lines)-1 != report.RowCount {
		t.Errorf("expected %d data rows, got %d", report.RowCount, len(lines)-1)
	}
}

func TestGenerateReport_InvalidFrom(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GenerateReport(ctx, "01-01-2026", "2026-03-31", "json")
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
	if !strings.Contains(err.Error(), "from") {
		t.Errorf("expected error mentioning 'from', got %s", err.Error())
	}
}

func TestGenerateReport_InvalidTo(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GenerateReport(ctx, "2026-01-01", "not-a-date", "json")
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
	if !strings.Contains(err.Error(), "to") {
		t.Errorf("expected error mentioning 'to', got %s", err.Error())
	}
}

func TestGenerateReport_InvalidFormat(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GenerateReport(ctx, "2026-01-01", "2026-03-31", "xml")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "format") {
		t.Errorf("expected error mentioning 'format', got %s", err.Error())
	}
}

func TestGetAuditSummary_AllPeriods(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	periods := []struct {
		period string
		want   string
	}{
		{"30d", "Last 30 days"},
		{"90d", "Last 90 days"},
		{"1y", "Last 12 months"},
		{"all", "All time"},
	}

	for _, p := range periods {
		summary, err := eng.GetAuditSummary(ctx, p.period)
		if err != nil {
			t.Errorf("GetAuditSummary(%q): %v", p.period, err)
			continue
		}
		if summary.TotalNegotiations <= 0 {
			t.Errorf("GetAuditSummary(%q): expected positive TotalNegotiations, got %d", p.period, summary.TotalNegotiations)
		}
		if summary.TotalSavings <= 0 {
			t.Errorf("GetAuditSummary(%q): expected positive TotalSavings, got %f", p.period, summary.TotalSavings)
		}
		if summary.PeriodDescription != p.want {
			t.Errorf("GetAuditSummary(%q): expected description %q, got %q", p.period, p.want, summary.PeriodDescription)
		}
	}
}

func TestGetAuditSummary_30dGreaterThanZero(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	summary, err := eng.GetAuditSummary(ctx, "30d")
	if err != nil {
		t.Fatalf("GetAuditSummary: %v", err)
	}
	if summary.TotalNegotiations <= 0 {
		t.Errorf("expected positive TotalNegotiations, got %d", summary.TotalNegotiations)
	}
	if summary.TotalSavings <= 0 {
		t.Errorf("expected positive TotalSavings, got %f", summary.TotalSavings)
	}
	if summary.AvgDiscount <= 0 {
		t.Errorf("expected positive AvgDiscount, got %f", summary.AvgDiscount)
	}
}

func TestGetAuditSummary_InvalidPeriod(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GetAuditSummary(ctx, "forever")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestGetAuditTrail_Valid(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	entries, err := eng.GetAuditTrail(ctx, "negotiation", "session-123")
	if err != nil {
		t.Fatalf("GetAuditTrail: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least 1 entry")
	}

	for i, e := range entries {
		if e.EntityType != "negotiation" {
			t.Errorf("entry[%d]: expected EntityType negotiation, got %s", i, e.EntityType)
		}
		if e.EntityID != "session-123" {
			t.Errorf("entry[%d]: expected EntityID session-123, got %s", i, e.EntityID)
		}
		if e.Action == "" {
			t.Errorf("entry[%d]: expected non-empty Action", i)
		}
		if e.Timestamp == "" {
			t.Errorf("entry[%d]: expected non-empty Timestamp", i)
		}
	}
}

func TestGetAuditTrail_EmptyEntityType(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GetAuditTrail(ctx, "", "id-1")
	if err == nil {
		t.Fatal("expected error for empty entity_type")
	}
}

func TestGetAuditTrail_EmptyEntityID(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GetAuditTrail(ctx, "negotiation", "")
	if err == nil {
		t.Fatal("expected error for empty entity_id")
	}
}
