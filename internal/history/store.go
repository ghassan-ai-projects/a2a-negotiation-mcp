package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
	_ "modernc.org/sqlite"
)

// Store manages negotiation session and deal history in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a HistoryStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate history: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS negotiation_sessions (
		id TEXT PRIMARY KEY,
		vendor TEXT NOT NULL,
		sku TEXT NOT NULL DEFAULT '',
		strategy TEXT NOT NULL,
		budget REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		current_offer REAL NOT NULL DEFAULT 0,
		list_price REAL NOT NULL DEFAULT 0,
		rounds_complete INTEGER NOT NULL DEFAULT 0,
		outcome TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS negotiation_rounds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL REFERENCES negotiation_sessions(id),
		round_number INTEGER NOT NULL,
		offer REAL NOT NULL,
		discount_pct REAL NOT NULL,
		counterparty TEXT NOT NULL,
		note TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS deal_outcomes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		sku TEXT NOT NULL DEFAULT '',
		list_price REAL NOT NULL DEFAULT 0,
		final_price REAL NOT NULL DEFAULT 0,
		discount_pct REAL NOT NULL DEFAULT 0,
		seats INTEGER NOT NULL DEFAULT 0,
		term_months INTEGER NOT NULL DEFAULT 12,
		strategy TEXT NOT NULL DEFAULT '',
		session_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_vendor ON negotiation_sessions(vendor);
	CREATE INDEX IF NOT EXISTS idx_sessions_status ON negotiation_sessions(status);
	CREATE INDEX IF NOT EXISTS idx_rounds_session ON negotiation_rounds(session_id);
	CREATE INDEX IF NOT EXISTS idx_deals_vendor ON deal_outcomes(vendor);
	CREATE INDEX IF NOT EXISTS idx_deals_created ON deal_outcomes(created_at);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SaveSession saves a new negotiation session.
func (s *Store) SaveSession(ctx context.Context, sess *SessionRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO negotiation_sessions (id, vendor, sku, strategy, budget, status, current_offer, list_price, rounds_complete, outcome, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sess.ID, sess.Vendor, sess.SKU, sess.Strategy, sess.Budget, sess.Status,
		sess.CurrentOffer, sess.ListPrice, sess.RoundsComplete, sess.Outcome,
		sess.CreatedAt.Format(time.RFC3339), sess.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// UpdateSession updates an existing session's mutable fields.
func (s *Store) UpdateSession(ctx context.Context, sess *SessionRecord) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE negotiation_sessions SET status=?, current_offer=?, rounds_complete=?, outcome=?, updated_at=?
		WHERE id=?
	`, sess.Status, sess.CurrentOffer, sess.RoundsComplete, sess.Outcome,
		sess.UpdatedAt.Format(time.RFC3339), sess.ID)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ierrors.New(ierrors.ErrSessionNotFound, "session not found for update",
			map[string]any{"session_id": sess.ID})
	}
	return nil
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(ctx context.Context, id string) (*SessionRecord, error) {
	var sess SessionRecord
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, sku, strategy, budget, status, current_offer, list_price,
			   rounds_complete, outcome, created_at, updated_at
		FROM negotiation_sessions WHERE id = ?
	`, id).Scan(&sess.ID, &sess.Vendor, &sess.SKU, &sess.Strategy, &sess.Budget,
		&sess.Status, &sess.CurrentOffer, &sess.ListPrice, &sess.RoundsComplete,
		&sess.Outcome, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ierrors.New(ierrors.ErrSessionNotFound, "session not found",
			map[string]any{"session_id": id})
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sess.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &sess, nil
}

// SaveRounds saves negotiation round records.
func (s *Store) SaveRounds(ctx context.Context, rounds []RoundRecord) error {
	for _, r := range rounds {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO negotiation_rounds (session_id, round_number, offer, discount_pct, counterparty, note, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, r.SessionID, r.RoundNumber, r.Offer, r.DiscountPct, r.Counterparty, r.Note,
			r.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("save round: %w", err)
		}
	}
	return nil
}

// GetRounds returns all rounds for a session.
func (s *Store) GetRounds(ctx context.Context, sessionID string) ([]RoundRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, round_number, offer, discount_pct, counterparty, note, created_at
		FROM negotiation_rounds WHERE session_id = ? ORDER BY round_number
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query rounds: %w", err)
	}
	defer rows.Close()

	var rounds []RoundRecord
	for rows.Next() {
		var r RoundRecord
		var createdAt string
		if err := rows.Scan(&r.ID, &r.SessionID, &r.RoundNumber, &r.Offer, &r.DiscountPct,
			&r.Counterparty, &r.Note, &createdAt); err != nil {
			return nil, fmt.Errorf("scan round: %w", err)
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rounds = append(rounds, r)
	}
	return rounds, rows.Err()
}

// SaveDealOutcome saves a completed deal outcome.
func (s *Store) SaveDealOutcome(ctx context.Context, deal *DealOutcome) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, deal.Vendor, deal.SKU, deal.ListPrice, deal.FinalPrice, deal.DiscountPct,
		deal.Seats, deal.TermMonths, deal.Strategy, deal.SessionID,
		deal.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save deal: %w", err)
	}
	return nil
}

// GetHistory returns aggregated negotiation history.
func (s *Store) GetHistory(ctx context.Context, vendor string, period string) (*HistorySummary, error) {
	where := ""
	args := []any{}

	switch period {
	case "30d":
		where += " AND d.created_at >= datetime('now', '-30 days')"
	case "90d":
		where += " AND d.created_at >= datetime('now', '-90 days')"
	case "1y":
		where += " AND d.created_at >= datetime('now', '-1 year')"
	case "all", "":
	default:
		return nil, ierrors.New(ierrors.ErrInvalidArgument, "invalid period",
			map[string]any{"period": period})
	}

	if vendor != "" {
		where = " AND d.vendor = ?" + where
		args = append([]any{vendor}, args...)
	}


	// Deal stats
	var totalDeals int
	var totalSavings float64
	var totalDiscount float64

	statsQuery := `SELECT COUNT(*), COALESCE(SUM(d.list_price - d.final_price), 0), COALESCE(AVG(d.discount_pct), 0) FROM deal_outcomes d WHERE 1=1` + where
	err := s.db.QueryRowContext(ctx, statsQuery, args...).Scan(&totalDeals, &totalSavings, &totalDiscount)
	if err != nil {
		return nil, fmt.Errorf("query deal stats: %w", err)
	}

	winRate := 100.0
	if totalDeals > 0 {
		winRate = 100.0
	}

	avgDiscount := 0.0
	if totalDeals > 0 {
		avgDiscount = totalDiscount / float64(totalDeals) * 100
	}

	// Deals list
	dealsQuery := `SELECT vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at FROM deal_outcomes d WHERE 1=1` + where + ` ORDER BY d.created_at DESC LIMIT 100`
	rows, err := s.db.QueryContext(ctx, dealsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query deals: %w", err)
	}
	defer rows.Close()

	var deals []DealOutcome
	for rows.Next() {
		var d DealOutcome
		var createdAt string
		if err := rows.Scan(&d.Vendor, &d.SKU, &d.ListPrice, &d.FinalPrice, &d.DiscountPct,
			&d.Seats, &d.TermMonths, &d.Strategy, &d.SessionID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan deal: %w", err)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deals = append(deals, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range deals {
		deals[i].DiscountPct = deals[i].DiscountPct * 100
	}

	return &HistorySummary{
		TotalDeals:     totalDeals,
		WinRate:        winRate,
		AvgDiscountPct: avgDiscount,
		TotalSavings:   totalSavings,
		Deals:          deals,
	}, nil
}

// GetSimilarDeals finds deals similar to a given vendor configuration.
func (s *Store) GetSimilarDeals(ctx context.Context, vendor string, limit int) ([]DealOutcome, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at
		FROM deal_outcomes WHERE vendor = ? ORDER BY created_at DESC LIMIT ?
	`, vendor, limit)
	if err != nil {
		return nil, fmt.Errorf("query similar deals: %w", err)
	}
	defer rows.Close()

	var deals []DealOutcome
	for rows.Next() {
		var d DealOutcome
		var createdAt string
		if err := rows.Scan(&d.Vendor, &d.SKU, &d.ListPrice, &d.FinalPrice, &d.DiscountPct,
			&d.Seats, &d.TermMonths, &d.Strategy, &d.SessionID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan deal: %w", err)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deals = append(deals, d)
	}
	return deals, rows.Err()
}
