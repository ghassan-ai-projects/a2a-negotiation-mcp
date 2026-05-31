package esignature

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

func TestCreateEnvelope(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	env, err := s.CreateEnvelope(ctx, "contract-001", "signer@example.com")
	if err != nil {
		t.Fatalf("CreateEnvelope: %v", err)
	}

	if env.ContractID != "contract-001" {
		t.Errorf("expected contract_id 'contract-001', got %s", env.ContractID)
	}
	if env.SignerEmail != "signer@example.com" {
		t.Errorf("expected signer_email 'signer@example.com', got %s", env.SignerEmail)
	}
	if env.Status != "sent" {
		t.Errorf("expected status 'sent', got %s", env.Status)
	}
	if env.EnvelopeID == "" {
		t.Errorf("expected non-empty envelope_id")
	}
	if env.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if env.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}

	// Verify envelope_id starts with "env_"
	if len(env.EnvelopeID) < 4 || env.EnvelopeID[:4] != "env_" {
		t.Errorf("expected envelope_id to start with 'env_', got %s", env.EnvelopeID)
	}
}

func TestGetEnvelope(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	created, err := s.CreateEnvelope(ctx, "contract-002", "user@test.com")
	if err != nil {
		t.Fatalf("CreateEnvelope: %v", err)
	}

	got, err := s.GetEnvelope(ctx, created.EnvelopeID)
	if err != nil {
		t.Fatalf("GetEnvelope: %v", err)
	}

	if got.ContractID != "contract-002" {
		t.Errorf("expected contract_id 'contract-002', got %s", got.ContractID)
	}
	if got.SignerEmail != "user@test.com" {
		t.Errorf("expected signer_email 'user@test.com', got %s", got.SignerEmail)
	}
	if got.Status != "sent" {
		t.Errorf("expected status 'sent', got %s", got.Status)
	}
	if got.EnvelopeID != created.EnvelopeID {
		t.Errorf("expected envelope_id %s, got %s", created.EnvelopeID, got.EnvelopeID)
	}
}

func TestGetEnvelope_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetEnvelope(ctx, "env_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent envelope")
	}
}

func TestGetSignedDocument(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Create an envelope
	created, err := s.CreateEnvelope(ctx, "contract-003", "signer@test.com")
	if err != nil {
		t.Fatalf("CreateEnvelope: %v", err)
	}

	// Should fail — not signed yet
	_, err = s.GetSignedDocument(ctx, created.EnvelopeID)
	if err == nil {
		t.Fatal("expected error for unsigned document")
	}

	// Sign it directly in the DB
	_, err = s.db.ExecContext(ctx,
		`UPDATE signature_envelopes SET status = 'signed', signed_at = datetime('now') WHERE envelope_id = ?`,
		created.EnvelopeID)
	if err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Now should succeed
	doc, err := s.GetSignedDocument(ctx, created.EnvelopeID)
	if err != nil {
		t.Fatalf("GetSignedDocument: %v", err)
	}
	if doc.Status != "signed" {
		t.Errorf("expected status 'signed', got %s", doc.Status)
	}
}
