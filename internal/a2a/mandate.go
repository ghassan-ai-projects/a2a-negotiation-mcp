package a2a

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
)

// MandateStatus represents the lifecycle status of a mandate.
type MandateStatus string

const (
	MandateStatusPending   MandateStatus = "pending"
	MandateStatusActive    MandateStatus = "active"
	MandateStatusSettled   MandateStatus = "settled"
	MandateStatusCancelled MandateStatus = "cancelled"
)

// MandateType represents the type of mandate.
type MandateType string

const (
	MandateTypeIntent  MandateType = "intent"
	MandateTypeCart    MandateType = "cart"
	MandateTypePayment MandateType = "payment"
)

// Mandate is an AP2-style mandate for agent-to-agent authorization.
type Mandate struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`       // "intent" | "cart" | "payment"
	Principal string         `json:"principal"`  // requesting agent identity
	AgentID   string         `json:"agent_id"`   // responding agent identity
	Status    string         `json:"status"`     // "pending" | "active" | "settled" | "cancelled"
	ExpiresAt time.Time      `json:"expires_at"`
	Terms     map[string]any `json:"terms"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// MandateStore provides mandate CRUD operations backed by SQLite.
type MandateStore struct {
	db *sql.DB
}

// NewMandateStore creates a MandateStore using an existing DB connection.
func NewMandateStore(db *sql.DB) (*MandateStore, error) {
	s := &MandateStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate mandates: %w", err)
	}
	return s, nil
}

func (s *MandateStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mandates (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		principal TEXT NOT NULL,
		agent_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		expires_at TEXT NOT NULL,
		terms TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_mandates_status ON mandates(status);
	CREATE INDEX IF NOT EXISTS idx_mandates_principal ON mandates(principal);
	CREATE INDEX IF NOT EXISTS idx_mandates_type ON mandates(type);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateMandate stores a new mandate.
func (s *MandateStore) CreateMandate(ctx context.Context, m *Mandate) error {
	termsJSON, err := marshalTerms(m.Terms)
	if err != nil {
		return fmt.Errorf("marshal terms: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO mandates (id, type, principal, agent_id, status, expires_at, terms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Type, m.Principal, m.AgentID, m.Status,
		m.ExpiresAt.Format(time.RFC3339), termsJSON,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create mandate: %w", err)
	}
	return nil
}

// GetMandate retrieves a mandate by ID.
func (s *MandateStore) GetMandate(ctx context.Context, id string) (*Mandate, error) {
	var m Mandate
	var expiresAt, createdAt, updatedAt string
	var termsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, principal, agent_id, status, expires_at, terms, created_at, updated_at
		FROM mandates WHERE id = ?
	`, id).Scan(&m.ID, &m.Type, &m.Principal, &m.AgentID, &m.Status,
		&expiresAt, &termsJSON, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ierrors.New(ierrors.ErrSessionNotFound, "mandate not found",
			map[string]any{"mandate_id": id})
	}
	if err != nil {
		return nil, fmt.Errorf("get mandate: %w", err)
	}

	m.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	m.Terms = unmarshalTerms(termsJSON)
	return &m, nil
}

// SettleMandate marks a mandate as settled.
func (s *MandateStore) SettleMandate(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE mandates SET status = 'settled', updated_at = ? WHERE id = ? AND status IN ('active', 'pending')
	`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("settle mandate: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ierrors.New(ierrors.ErrSessionNotFound, "mandate not found or not in settleable state",
			map[string]any{"mandate_id": id})
	}
	return nil
}

// CancelMandate marks a mandate as cancelled.
func (s *MandateStore) CancelMandate(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE mandates SET status = 'cancelled', updated_at = ? WHERE id = ? AND status NOT IN ('settled', 'cancelled')
	`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("cancel mandate: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ierrors.New(ierrors.ErrSessionNotFound, "mandate not found or already finalized",
			map[string]any{"mandate_id": id})
	}
	return nil
}

// ExpireMandates marks all expired pending/active mandates as cancelled.
func (s *MandateStore) ExpireMandates(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE mandates SET status = 'cancelled', updated_at = ?
		WHERE status IN ('pending', 'active') AND expires_at <= ?
	`, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("expire mandates: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListMandates returns mandates filtered by optional status and type.
func (s *MandateStore) ListMandates(ctx context.Context, status, mandateType string) ([]*Mandate, error) {
	query := "SELECT id, type, principal, agent_id, status, expires_at, terms, created_at, updated_at FROM mandates WHERE 1=1"
	var args []any

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if mandateType != "" {
		query += " AND type = ?"
		args = append(args, mandateType)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mandates: %w", err)
	}
	defer rows.Close()

	var mandates []*Mandate
	for rows.Next() {
		var m Mandate
		var expiresAt, createdAt, updatedAt, termsJSON string
		if err := rows.Scan(&m.ID, &m.Type, &m.Principal, &m.AgentID, &m.Status,
			&expiresAt, &termsJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan mandate: %w", err)
		}
		m.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		m.Terms = unmarshalTerms(termsJSON)
		mandates = append(mandates, &m)
	}
	return mandates, rows.Err()
}

// marshalTerms serializes terms to a JSON string for SQLite storage.
func marshalTerms(terms map[string]any) (string, error) {
	if terms == nil {
		return "{}", nil
	}
	b, err := json.Marshal(terms)
	if err != nil {
		return "{}", fmt.Errorf("marshal terms: %w", err)
	}
	return string(b), nil
}

// unmarshalTerms deserializes terms from a JSON string.
func unmarshalTerms(s string) map[string]any {
	if s == "" || s == "{}" {
		return nil
	}
	var terms map[string]any
	if err := json.Unmarshal([]byte(s), &terms); err != nil {
		return nil
	}
	return terms
}
