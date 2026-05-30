package a2a

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupMandateTest(t *testing.T) (*MandateStore, context.Context) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewMandateStore(db)
	if err != nil {
		t.Fatalf("NewMandateStore: %v", err)
	}

	return store, context.Background()
}

func TestCreateMandate(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-1",
		Type:      "intent",
		Principal: "agent-alpha",
		AgentID:   "agent-beta",
		Status:    "pending",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Terms:     map[string]any{"vendor": "Slack", "sku": "Pro"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	got, err := store.GetMandate(ctx, "mandate-1")
	if err != nil {
		t.Fatalf("GetMandate: %v", err)
	}
	if got.ID != "mandate-1" {
		t.Errorf("ID = %q, want %q", got.ID, "mandate-1")
	}
	if got.Type != "intent" {
		t.Errorf("Type = %q, want %q", got.Type, "intent")
	}
	if got.Principal != "agent-alpha" {
		t.Errorf("Principal = %q, want %q", got.Principal, "agent-alpha")
	}
	if got.Status != "pending" {
		t.Errorf("Status = %q, want %q", got.Status, "pending")
	}
	vendor, ok := got.Terms["vendor"].(string)
	if !ok || vendor != "Slack" {
		t.Errorf("Terms.vendor = %v, want %q", got.Terms["vendor"], "Slack")
	}
}

func TestGetMandate_NotFound(t *testing.T) {
	store, ctx := setupMandateTest(t)

	_, err := store.GetMandate(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mandate")
	}
}

func TestSettleMandate(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-2",
		Type:      "cart",
		Principal: "agent-alpha",
		Status:    "active",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Terms:     map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	if err := store.SettleMandate(ctx, "mandate-2"); err != nil {
		t.Fatalf("SettleMandate: %v", err)
	}

	got, err := store.GetMandate(ctx, "mandate-2")
	if err != nil {
		t.Fatalf("GetMandate: %v", err)
	}
	if got.Status != "settled" {
		t.Errorf("Status = %q, want %q", got.Status, "settled")
	}
}

func TestCancelMandate(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-3",
		Type:      "payment",
		Principal: "agent-alpha",
		Status:    "pending",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Terms:     map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	if err := store.CancelMandate(ctx, "mandate-3"); err != nil {
		t.Fatalf("CancelMandate: %v", err)
	}

	got, err := store.GetMandate(ctx, "mandate-3")
	if err != nil {
		t.Fatalf("GetMandate: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("Status = %q, want %q", got.Status, "cancelled")
	}
}

func TestCancelMandate_AlreadySettled(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-4",
		Type:      "intent",
		Principal: "agent-alpha",
		Status:    "active",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Terms:     map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	_ = store.SettleMandate(ctx, "mandate-4")

	err := store.CancelMandate(ctx, "mandate-4")
	if err == nil {
		t.Fatal("expected error when cancelling settled mandate")
	}
}

func TestExpireMandates(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-expired",
		Type:      "intent",
		Principal: "agent-alpha",
		Status:    "pending",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
		Terms:     map[string]any{},
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	n, err := store.ExpireMandates(ctx)
	if err != nil {
		t.Fatalf("ExpireMandates: %v", err)
	}
	if n != 1 {
		t.Errorf("expired count = %d, want 1", n)
	}

	got, err := store.GetMandate(ctx, "mandate-expired")
	if err != nil {
		t.Fatalf("GetMandate: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("Status = %q, want %q", got.Status, "cancelled")
	}
}

func TestCreateMandate_NilTerms(t *testing.T) {
	store, ctx := setupMandateTest(t)

	m := &Mandate{
		ID:        "mandate-nil-terms",
		Type:      "intent",
		Principal: "agent-alpha",
		Status:    "pending",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := store.CreateMandate(ctx, m); err != nil {
		t.Fatalf("CreateMandate: %v", err)
	}

	got, err := store.GetMandate(ctx, "mandate-nil-terms")
	if err != nil {
		t.Fatalf("GetMandate: %v", err)
	}
	if got.Terms != nil {
		t.Errorf("Terms = %v, want nil", got.Terms)
	}
}
