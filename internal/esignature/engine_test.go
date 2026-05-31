package esignature

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupEngineTest(t *testing.T) (*Store, *Engine) {
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
	eng := NewEngine()
	return store, eng
}

func TestSimulateSigning(t *testing.T) {
	store, eng := setupEngineTest(t)
	ctx := context.Background()

	// Create an envelope first
	env, err := store.CreateEnvelope(ctx, "contract-010", "signer@example.com")
	if err != nil {
		t.Fatalf("CreateEnvelope: %v", err)
	}

	// Simulate signing
	result, err := eng.SimulateSigning(ctx, store, env.EnvelopeID)
	if err != nil {
		t.Fatalf("SimulateSigning: %v", err)
	}

	if result.EnvelopeID != env.EnvelopeID {
		t.Errorf("expected envelope_id %s, got %s", env.EnvelopeID, result.EnvelopeID)
	}
	if result.ContractID != "contract-010" {
		t.Errorf("expected contract_id 'contract-010', got %s", result.ContractID)
	}
	if result.Status != "signed" {
		t.Errorf("expected status 'signed', got %s", result.Status)
	}
	if result.SignerEmail != "signer@example.com" {
		t.Errorf("expected signer_email 'signer@example.com', got %s", result.SignerEmail)
	}
	if result.SignedAt == "" {
		t.Errorf("expected non-empty signed_at")
	}

	// Verify the envelope was updated in the store
	updated, err := store.GetEnvelope(ctx, env.EnvelopeID)
	if err != nil {
		t.Fatalf("GetEnvelope after signing: %v", err)
	}
	if updated.Status != "signed" {
		t.Errorf("expected stored status 'signed', got %s", updated.Status)
	}
}

func TestSimulateSigning_AlreadySigned(t *testing.T) {
	store, eng := setupEngineTest(t)
	ctx := context.Background()

	env, err := store.CreateEnvelope(ctx, "contract-011", "signer2@example.com")
	if err != nil {
		t.Fatalf("CreateEnvelope: %v", err)
	}

	// Sign once
	_, err = eng.SimulateSigning(ctx, store, env.EnvelopeID)
	if err != nil {
		t.Fatalf("first SimulateSigning: %v", err)
	}

	// Try signing again — should fail
	_, err = eng.SimulateSigning(ctx, store, env.EnvelopeID)
	if err == nil {
		t.Fatal("expected error when signing already-signed envelope")
	}
}

func TestSimulateSigning_NotFound(t *testing.T) {
	store, eng := setupEngineTest(t)
	ctx := context.Background()

	_, err := eng.SimulateSigning(ctx, store, "env_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent envelope")
	}
}
