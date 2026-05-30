package sla

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides SLA contract and breach data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate sla: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sla_contracts (
		id TEXT PRIMARY KEY,
		vendor TEXT,
		service TEXT,
		uptime_pct REAL,
		credit_pct REAL,
		max_credit_pct REAL,
		monthly_spend REAL,
		status TEXT DEFAULT 'active'
	);

	CREATE TABLE IF NOT EXISTS sla_breaches (
		id TEXT PRIMARY KEY,
		vendor TEXT,
		service TEXT,
		date TEXT,
		duration_mins INTEGER,
		credit_due REAL,
		filed INTEGER DEFAULT 0,
		filed_at TEXT,
		payout REAL,
		notes TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_sla_breaches_vendor ON sla_breaches(vendor);
	CREATE INDEX IF NOT EXISTS idx_sla_breaches_date ON sla_breaches(date);
	CREATE INDEX IF NOT EXISTS idx_sla_contracts_status ON sla_contracts(status);
	`
	_, err := s.db.Exec(schema)
	return err
}

// AddContract inserts a new SLA contract.
func (s *Store) AddContract(ctx context.Context, c *SLAContract) error {
	c.ID = uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sla_contracts (id, vendor, service, uptime_pct, credit_pct, max_credit_pct, monthly_spend, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Vendor, c.Service, c.UptimePct, c.CreditPct, c.MaxCreditPct, c.MonthlySpend, c.Status)
	if err != nil {
		return fmt.Errorf("add contract: %w", err)
	}
	return nil
}

// GetContract retrieves an SLA contract by ID.
func (s *Store) GetContract(ctx context.Context, id string) (*SLAContract, error) {
	var c SLAContract
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, service, uptime_pct, credit_pct, max_credit_pct, monthly_spend, status
		FROM sla_contracts WHERE id = ?
	`, id).Scan(&c.ID, &c.Vendor, &c.Service, &c.UptimePct, &c.CreditPct, &c.MaxCreditPct, &c.MonthlySpend, &c.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contract not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get contract: %w", err)
	}
	return &c, nil
}

// ListContracts returns all SLA contracts, optionally filtered by status.
func (s *Store) ListContracts(ctx context.Context, status string) ([]SLAContract, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, uptime_pct, credit_pct, max_credit_pct, monthly_spend, status
			FROM sla_contracts WHERE status = ? ORDER BY vendor, service
		`, status)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, uptime_pct, credit_pct, max_credit_pct, monthly_spend, status
			FROM sla_contracts ORDER BY vendor, service
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []SLAContract
	for rows.Next() {
		var c SLAContract
		if err := rows.Scan(&c.ID, &c.Vendor, &c.Service, &c.UptimePct, &c.CreditPct, &c.MaxCreditPct, &c.MonthlySpend, &c.Status); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		contracts = append(contracts, c)
	}
	return contracts, rows.Err()
}

// ListBreaches returns breaches, optionally filtered by vendor and/or date range.
func (s *Store) ListBreaches(ctx context.Context, vendor string, startDate, endDate time.Time) ([]SLABreach, error) {
	var rows *sql.Rows
	var err error

	switch {
	case vendor != "" && !startDate.IsZero() && !endDate.IsZero():
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes
			FROM sla_breaches WHERE vendor = ? AND date >= ? AND date <= ? ORDER BY date DESC
		`, vendor, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))
	case vendor != "":
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes
			FROM sla_breaches WHERE vendor = ? ORDER BY date DESC
		`, vendor)
	case !startDate.IsZero() && !endDate.IsZero():
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes
			FROM sla_breaches WHERE date >= ? AND date <= ? ORDER BY date DESC
		`, startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))
	default:
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes
			FROM sla_breaches ORDER BY date DESC
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("list breaches: %w", err)
	}
	defer rows.Close()

	var breaches []SLABreach
	for rows.Next() {
		var b SLABreach
		var dateStr, filedAtStr sql.NullString
		if err := rows.Scan(&b.ID, &b.Vendor, &b.Service, &dateStr, &b.DurationMins, &b.CreditDue, &b.Filed, &filedAtStr, &b.Payout, &b.Notes); err != nil {
			return nil, fmt.Errorf("scan breach: %w", err)
		}
		if dateStr.Valid {
			b.Date, _ = time.Parse(time.RFC3339, dateStr.String)
		}
		if filedAtStr.Valid {
			b.FiledAt, _ = time.Parse(time.RFC3339, filedAtStr.String)
		}
		breaches = append(breaches, b)
	}
	return breaches, rows.Err()
}

// AddBreach inserts a new SLA breach.
func (s *Store) AddBreach(ctx context.Context, b *SLABreach) error {
	b.ID = uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sla_breaches (id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.Vendor, b.Service, b.Date.Format(time.RFC3339), b.DurationMins, b.CreditDue, boolToInt(b.Filed), nullTime(b.FiledAt), b.Payout, b.Notes)
	if err != nil {
		return fmt.Errorf("add breach: %w", err)
	}
	return nil
}

// FileBreach marks a breach as filed, sets the filed timestamp and payout.
func (s *Store) FileBreach(ctx context.Context, breachID string, payout float64) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE sla_breaches SET filed = 1, filed_at = ?, payout = ? WHERE id = ?
	`, now.Format(time.RFC3339), payout, breachID)
	if err != nil {
		return fmt.Errorf("file breach: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("breach not found: %s", breachID)
	}
	return nil
}

// GetBreach retrieves a single breach by ID.
func (s *Store) GetBreach(ctx context.Context, id string) (*SLABreach, error) {
	var b SLABreach
	var dateStr, filedAtStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, service, date, duration_mins, credit_due, filed, filed_at, payout, notes
		FROM sla_breaches WHERE id = ?
	`, id).Scan(&b.ID, &b.Vendor, &b.Service, &dateStr, &b.DurationMins, &b.CreditDue, &b.Filed, &filedAtStr, &b.Payout, &b.Notes)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("breach not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get breach: %w", err)
	}
	if dateStr.Valid {
		b.Date, _ = time.Parse(time.RFC3339, dateStr.String)
	}
	if filedAtStr.Valid {
		b.FiledAt, _ = time.Parse(time.RFC3339, filedAtStr.String)
	}
	return &b, nil
}

// UpdateContractStatus updates the status of an SLA contract.
func (s *Store) UpdateContractStatus(ctx context.Context, contractID, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE sla_contracts SET status = ? WHERE id = ?", status, contractID)
	if err != nil {
		return fmt.Errorf("update contract status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("contract not found: %s", contractID)
	}
	return nil
}

// --- helpers ---

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullTime(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
