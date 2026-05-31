package toolstats

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	s, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestLogCall(t *testing.T) {
	s := newTestStore(t)
	eng := NewEngine(s)

	ctx := context.Background()
	if err := eng.LogCall(ctx, "test_tool_query_price"); err != nil {
		t.Fatalf("LogCall failed: %v", err)
	}
	if err := eng.LogCall(ctx, "test_tool_query_price"); err != nil {
		t.Fatalf("LogCall failed: %v", err)
	}
	if err := eng.LogCall(ctx, "test_tool_create_session"); err != nil {
		t.Fatalf("LogCall failed: %v", err)
	}

	report, err := eng.GetReport(ctx, "24h")
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}

	if report.TotalCalls != 3 {
		t.Errorf("expected 3 total calls, got %d", report.TotalCalls)
	}
	if report.UniqueTools != 2 {
		t.Errorf("expected 2 unique tools, got %d", report.UniqueTools)
	}
	if len(report.TopTools) != 2 {
		t.Errorf("expected 2 top tools, got %d", len(report.TopTools))
	}
	if report.TopTools[0].ToolName != "test_tool_query_price" {
		t.Errorf("expected top tool to be test_tool_query_price, got %s", report.TopTools[0].ToolName)
	}
	if report.TopTools[0].CallCount != 2 {
		t.Errorf("expected top tool call count 2, got %d", report.TopTools[0].CallCount)
	}
}

func TestGetReport_EmptyData(t *testing.T) {
	s := newTestStore(t)
	eng := NewEngine(s)

	ctx := context.Background()
	report, err := eng.GetReport(ctx, "7d")
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}

	if report.TotalCalls != 0 {
		t.Errorf("expected 0 total calls, got %d", report.TotalCalls)
	}
	if report.UniqueTools != 0 {
		t.Errorf("expected 0 unique tools, got %d", report.UniqueTools)
	}
	if len(report.TopTools) != 0 {
		t.Errorf("expected 0 top tools, got %d", len(report.TopTools))
	}
	if report.Period != "7d" {
		t.Errorf("expected period 7d, got %s", report.Period)
	}
}

func TestParsePeriod_Invalid(t *testing.T) {
	_, err := parsePeriod("invalid")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}
