package esignature

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// Store manages e-signature envelopes in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates an EsignatureStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate esignature: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS signature_envelopes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		contract_id TEXT NOT NULL,
		signer_email TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'sent',
		envelope_id TEXT NOT NULL,
		signed_at TEXT,
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func generateEnvelopeID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "env_" + hex.EncodeToString(b)
}

// CreateEnvelope creates a new e-signature envelope and returns it.
func (s *Store) CreateEnvelope(ctx context.Context, contractID, signerEmail string) (*Envelope, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	envelopeID := generateEnvelopeID()

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO signature_envelopes (contract_id, signer_email, status, envelope_id, created_at)
		VALUES (?, ?, 'sent', ?, ?)
	`, contractID, signerEmail, envelopeID, now)
	if err != nil {
		return nil, fmt.Errorf("create envelope: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create envelope last insert id: %w", err)
	}

	return &Envelope{
		ID:          int(id),
		ContractID:  contractID,
		SignerEmail: signerEmail,
		Status:      "sent",
		EnvelopeID:  envelopeID,
		CreatedAt:   now,
	}, nil
}

// GetEnvelope retrieves an envelope by envelope_id.
func (s *Store) GetEnvelope(ctx context.Context, envelopeID string) (*Envelope, error) {
	var e Envelope
	err := s.db.QueryRowContext(ctx, `
		SELECT id, contract_id, signer_email, status, envelope_id, created_at
		FROM signature_envelopes WHERE envelope_id = ?
	`, envelopeID).Scan(&e.ID, &e.ContractID, &e.SignerEmail, &e.Status, &e.EnvelopeID, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("envelope not found: %s", envelopeID)
	}
	if err != nil {
		return nil, fmt.Errorf("get envelope: %w", err)
	}
	return &e, nil
}

// GetSignedDocument retrieves an envelope by envelope_id only if status is 'signed'.
func (s *Store) GetSignedDocument(ctx context.Context, envelopeID string) (*Envelope, error) {
	e, err := s.GetEnvelope(ctx, envelopeID)
	if err != nil {
		return nil, err
	}
	if e.Status != "signed" {
		return nil, fmt.Errorf("envelope %s has not been signed (status: %s)", envelopeID, e.Status)
	}
	return e, nil
}
