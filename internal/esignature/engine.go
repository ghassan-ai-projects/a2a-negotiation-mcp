package esignature

import (
	"context"
	"fmt"
	"time"
)

// Engine handles e-signature operations (stateless).
type Engine struct{}

// NewEngine creates a new EsignatureEngine.
func NewEngine() *Engine {
	return &Engine{}
}

// SimulateSigning simulates signing an envelope by updating its status.
// It requires a store reference internally for the DB operation.
func (e *Engine) SimulateSigning(ctx context.Context, store *Store, envelopeID string) (*SignatureResult, error) {
	if store == nil {
		return nil, fmt.Errorf("esignature store is nil")
	}
	env, err := store.GetEnvelope(ctx, envelopeID)
	if err != nil {
		return nil, fmt.Errorf("simulate signing: %w", err)
	}
	if env.Status != "sent" {
		return nil, fmt.Errorf("envelope %s cannot be signed: current status is %s", envelopeID, env.Status)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = store.db.ExecContext(ctx, `
		UPDATE signature_envelopes SET status = 'signed', signed_at = ? WHERE envelope_id = ?
	`, now, envelopeID)
	if err != nil {
		return nil, fmt.Errorf("simulate signing update: %w", err)
	}

	return &SignatureResult{
		EnvelopeID:  envelopeID,
		ContractID:  env.ContractID,
		Status:      "signed",
		SignerEmail: env.SignerEmail,
		SignedAt:    now,
	}, nil
}
