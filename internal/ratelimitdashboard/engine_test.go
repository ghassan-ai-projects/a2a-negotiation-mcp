package ratelimitdashboard

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestLogRequestAndCount(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}

	ctx := context.Background()

	entry, err := store.LogRequest(ctx, "key-1", "negotiate_query_price")
	if err != nil {
		t.Fatalf("LogRequest: %v", err)
	}

	if entry.APIKeyID != "key-1" {
		t.Errorf("expected api_key_id=key-1, got %s", entry.APIKeyID)
	}
	if entry.Endpoint != "negotiate_query_price" {
		t.Errorf("expected endpoint=negotiate_query_price, got %s", entry.Endpoint)
	}
	if entry.ID <= 0 {
		t.Errorf("expected positive id, got %d", entry.ID)
	}
	if entry.Timestamp == "" {
		t.Errorf("expected non-empty timestamp")
	}
}

func TestCount(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}

	ctx := context.Background()

	// Log a few requests
	for i := 0; i < 5; i++ {
		_, err := store.LogRequest(ctx, "key-1", "negotiate_query_price")
		if err != nil {
			t.Fatalf("LogRequest %d: %v", i, err)
		}
	}

	today, err := store.CountToday(ctx)
	if err != nil {
		t.Fatalf("CountToday: %v", err)
	}
	if today != 5 {
		t.Errorf("expected count_today=5, got %d", today)
	}
}

func TestGetStatus_Green(t *testing.T) {
	store, err := NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	eng := NewEngine(store, logger)

	ctx := context.Background()

	// Log just a few requests (well under 50)
	for i := 0; i < 10; i++ {
		_, err := store.LogRequest(ctx, "key-1", "negotiate_query_price")
		if err != nil {
			t.Fatalf("LogRequest %d: %v", i, err)
		}
	}

	status, err := eng.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if status.Status != ColorGreen {
		t.Errorf("expected status=green for <50 requests, got %s", status.Status)
	}
	if status.RequestsToday != 10 {
		t.Errorf("expected requests_today=10, got %d", status.RequestsToday)
	}
	if status.RemainingBudget != 90 {
		t.Errorf("expected remaining_budget=90 (100-10), got %d", status.RemainingBudget)
	}
}
