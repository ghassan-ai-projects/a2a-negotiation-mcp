package shutdown_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/shutdown"
	_ "modernc.org/sqlite"
)

type mockClosable struct {
	closed    bool
	closeErr  error
}

func (m *mockClosable) Close() error {
	m.closed = true
	return m.closeErr
}

func TestShutdownWithDB(t *testing.T) {
	eng := shutdown.NewEngine()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// db should be open
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping before close: %v", err)
	}

	mock1 := &mockClosable{}
	mock2 := &mockClosable{}

	result := eng.Shutdown(db, []shutdown.Closable{mock1, mock2})

	if result.Status != "shutdown_complete" {
		t.Errorf("expected status 'shutdown_complete', got %q", result.Status)
	}
	if result.DurationMs < 0 {
		t.Errorf("expected positive duration_ms, got %d", result.DurationMs)
	}
	if len(result.ResourcesCleaned) < 3 {
		t.Errorf("expected at least 3 cleaned resources (2 stores + db), got %d: %v",
			len(result.ResourcesCleaned), result.ResourcesCleaned)
	}

	if !mock1.closed {
		t.Error("expected mock1 to be closed")
	}
	if !mock2.closed {
		t.Error("expected mock2 to be closed")
	}

	// DB should be closed
	if err := db.Ping(); err == nil {
		t.Error("expected db ping to fail after shutdown")
	}
}

func TestShutdownWithoutDB(t *testing.T) {
	eng := shutdown.NewEngine()

	mock1 := &mockClosable{}

	result := eng.Shutdown(nil, []shutdown.Closable{mock1})

	if result.Status != "shutdown_complete" {
		t.Errorf("expected status 'shutdown_complete', got %q", result.Status)
	}
	if !mock1.closed {
		t.Error("expected mock1 to be closed")
	}
}

func TestShutdownWithCloseError(t *testing.T) {
	eng := shutdown.NewEngine()

	mock1 := &mockClosable{closeErr: errors.New("close failed")}
	mock2 := &mockClosable{}

	result := eng.Shutdown(nil, []shutdown.Closable{mock1, mock2})

	if result.Status != "shutdown_complete" {
		t.Errorf("expected status 'shutdown_complete', got %q", result.Status)
	}
	if !mock1.closed {
		t.Error("expected mock1 to be closed")
	}
	if !mock2.closed {
		t.Error("expected mock2 to be closed")
	}

	// Check that the error is reported in resources_cleaned
	hasError := false
	for _, r := range result.ResourcesCleaned {
		if contains(r, "error: close failed") {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected resources_cleaned to report close error, got", result.ResourcesCleaned)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
