package autotrigger

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestSetTrigger(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	tr, err := store.SetTrigger(ctx, "price_drop > 10%", "start_negotiation", "Slack")
	if err != nil {
		t.Fatalf("SetTrigger: %v", err)
	}

	if tr.Condition != "price_drop > 10%" {
		t.Errorf("expected condition 'price_drop > 10%%', got %q", tr.Condition)
	}
	if tr.Action != "start_negotiation" {
		t.Errorf("expected action 'start_negotiation', got %q", tr.Action)
	}
	if tr.Vendor != "Slack" {
		t.Errorf("expected vendor 'Slack', got %q", tr.Vendor)
	}
	if !tr.Enabled {
		t.Error("expected trigger to be enabled")
	}
	if tr.ID == 0 {
		t.Error("expected non-zero trigger ID")
	}
	if tr.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestListTriggers(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// No triggers yet
	triggers, err := store.ListTriggers(ctx)
	if err != nil {
		t.Fatalf("ListTriggers (empty): %v", err)
	}
	if len(triggers) != 0 {
		t.Errorf("expected 0 triggers, got %d", len(triggers))
	}

	// Add two triggers
	_, err = store.SetTrigger(ctx, "price_drop > 10%", "start_negotiation", "Slack")
	if err != nil {
		t.Fatalf("SetTrigger 1: %v", err)
	}
	_, err = store.SetTrigger(ctx, "usage > 1000", "escalate_discount", "GitHub")
	if err != nil {
		t.Fatalf("SetTrigger 2: %v", err)
	}

	triggers, err = store.ListTriggers(ctx)
	if err != nil {
		t.Fatalf("ListTriggers: %v", err)
	}
	if len(triggers) != 2 {
		t.Errorf("expected 2 triggers, got %d", len(triggers))
	}
	// Both triggers should be present (order is created_at DESC, unstable with same-second timestamps)
	vendors := map[string]bool{}
	for _, t := range triggers {
		vendors[t.Vendor] = true
	}
	if !vendors["Slack"] {
		t.Error("expected Slack trigger in results")
	}
	if !vendors["GitHub"] {
		t.Error("expected GitHub trigger in results")
	}
}

func TestGetTriggerLog(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// No entries yet
	entries, err := store.GetTriggerLog(ctx)
	if err != nil {
		t.Fatalf("GetTriggerLog (empty): %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 log entries, got %d", len(entries))
	}
}
