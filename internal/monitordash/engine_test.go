package monitordash

import (
	"context"
	"strings"
	"testing"
)

func TestGetDashboard_ReturnsValidStructure(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	dash, err := eng.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}

	if dash == nil {
		t.Fatal("expected non-nil LiveDashboard")
	}

	if dash.ActiveNegotiations < 3 || dash.ActiveNegotiations > 15 {
		t.Errorf("ActiveNegotiations out of range [3-15]: got %d", dash.ActiveNegotiations)
	}

	if dash.TotalTools != 150 {
		t.Errorf("expected TotalTools=150, got %d", dash.TotalTools)
	}

	if dash.UptimeSeconds <= 0 {
		t.Errorf("expected positive UptimeSeconds, got %d", dash.UptimeSeconds)
	}

	if dash.Timestamp == "" {
		t.Error("expected non-empty Timestamp")
	}

	if dash.ErrorRate5Min < 0 || dash.ErrorRate5Min > 8.0 {
		t.Errorf("ErrorRate5Min out of range [0-8]: got %f", dash.ErrorRate5Min)
	}
}

func TestGetDashboard_SystemHealthIsValid(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	validHealth := map[string]bool{"healthy": true, "degraded": true}

	for i := 0; i < 20; i++ {
		dash, err := eng.GetDashboard(ctx)
		if err != nil {
			t.Fatalf("GetDashboard iteration %d: %v", i, err)
		}
		if !validHealth[dash.SystemHealth] {
			t.Errorf("invalid system_health %q at iteration %d", dash.SystemHealth, i)
		}
	}
}

func TestGetDashboard_ToolCallsCount(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	dash, err := eng.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}

	if len(dash.LastToolCalls) != 10 {
		t.Fatalf("expected 10 tool calls, got %d", len(dash.LastToolCalls))
	}

	for i, call := range dash.LastToolCalls {
		if strings.TrimSpace(call.ToolName) == "" {
			t.Errorf("tool call %d: empty ToolName", i)
		}
		if call.DurationMs <= 0 {
			t.Errorf("tool call %d: non-positive DurationMs (%d)", i, call.DurationMs)
		}
		if call.Timestamp == "" {
			t.Errorf("tool call %d: empty Timestamp", i)
		}
	}
}
