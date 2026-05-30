package health

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store provides vendor health data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store and ensures the schema exists.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// NewInMemoryStore creates a Store using an in-memory SQLite database (for tests).
func NewInMemoryStore() (*Store, error) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open in-memory db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping in-memory db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate in-memory: %w", err)
	}
	return s, nil
}

// NewStoreFromDB creates a Store using an existing *sql.DB (for sharing a DB connection).
func NewStoreFromDB(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate from db: %w", err)
	}
	return s, nil
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vendor_health (
		vendor TEXT PRIMARY KEY,
		score INTEGER NOT NULL DEFAULT 50,
		category TEXT NOT NULL DEFAULT 'stable',
		last_updated TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS health_signals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		type TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT 'manual',
		detail TEXT NOT NULL DEFAULT '',
		weight INTEGER NOT NULL DEFAULT 0,
		date TEXT NOT NULL DEFAULT (date('now')),
		FOREIGN KEY (vendor) REFERENCES vendor_health(vendor)
	);

	CREATE INDEX IF NOT EXISTS idx_health_signals_vendor ON health_signals(vendor);

	CREATE TABLE IF NOT EXISTS vendor_reputation (
		vendor TEXT PRIMARY KEY,
		deal_count INTEGER DEFAULT 0,
		total_discount_pct REAL DEFAULT 0,
		max_discount_pct REAL DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		updated_at TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// UpsertHealth inserts or updates a vendor health record.
func (s *Store) UpsertHealth(ctx context.Context, vh *VendorHealth) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_health (vendor, score, category, last_updated)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(vendor) DO UPDATE SET
			score=excluded.score,
			category=excluded.category,
			last_updated=excluded.last_updated
	`, vh.Vendor, vh.Score, vh.Category, vh.LastUpdated.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert vendor health: %w", err)
	}
	return nil
}

// GetHealth retrieves the health record for a vendor.
func (s *Store) GetHealth(ctx context.Context, vendor string) (*VendorHealth, error) {
	var vh VendorHealth
	var updated string
	err := s.db.QueryRowContext(ctx, `
		SELECT vendor, score, category, last_updated
		FROM vendor_health WHERE vendor = ?
	`, vendor).Scan(&vh.Vendor, &vh.Score, &vh.Category, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get vendor health: %w", err)
	}
	vh.LastUpdated, _ = time.Parse(time.RFC3339, updated)

	signals, err := s.GetSignals(ctx, vendor)
	if err != nil {
		return nil, err
	}
	vh.Signals = signals
	return &vh, nil
}

// AddSignal adds a signal and updates the last_updated timestamp.
func (s *Store) AddSignal(ctx context.Context, vendor, signalType, source, detail string, weight int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO health_signals (vendor, type, source, detail, weight, date)
		VALUES (?, ?, ?, ?, ?, date('now'))
	`, vendor, signalType, source, detail, weight)
	if err != nil {
		return fmt.Errorf("add signal: %w", err)
	}
	return nil
}

// GetSignals retrieves all signals for a vendor.
func (s *Store) GetSignals(ctx context.Context, vendor string) ([]Signal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, type, source, detail, weight, date
		FROM health_signals WHERE vendor = ?
		ORDER BY date DESC, id DESC
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("get signals: %w", err)
	}
	defer rows.Close()

	var signals []Signal
	for rows.Next() {
		var sig Signal
		if err := rows.Scan(&sig.ID, &vendor, &sig.Type, &sig.Source, &sig.Detail, &sig.Weight, &sig.Date); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		signals = append(signals, sig)
	}
	return signals, rows.Err()
}

// ListAll returns all vendor health records sorted by score ascending.
func (s *Store) ListAll(ctx context.Context) ([]VendorHealth, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, score, category, last_updated
		FROM vendor_health ORDER BY score ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list vendor health: %w", err)
	}
	defer rows.Close()

	var result []VendorHealth
	for rows.Next() {
		var vh VendorHealth
		var updated string
		if err := rows.Scan(&vh.Vendor, &vh.Score, &vh.Category, &updated); err != nil {
			return nil, fmt.Errorf("scan vendor health: %w", err)
		}
		vh.LastUpdated, _ = time.Parse(time.RFC3339, updated)
		signals, err := s.GetSignals(ctx, vh.Vendor)
		if err != nil {
			return nil, err
		}
		vh.Signals = signals
		result = append(result, vh)
	}
	return result, rows.Err()
}

// ─── Reputation Store Methods ───

// getReputationRow retrieves the raw reputation row for a vendor.
// Returns nil, nil if the vendor has no reputation record.
func (s *Store) getReputationRow(ctx context.Context, vendor string) (*reputationRow, error) {
	var row reputationRow
	err := s.db.QueryRowContext(ctx, `
		SELECT vendor, deal_count, total_discount_pct, max_discount_pct, success_count, updated_at
		FROM vendor_reputation WHERE vendor = ?
	`, vendor).Scan(&row.Vendor, &row.DealCount, &row.TotalDiscountPct, &row.MaxDiscountPct, &row.SuccessCount, &row.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get reputation row: %w", err)
	}
	return &row, nil
}

// upsertReputation inserts or updates a vendor reputation row.
func (s *Store) upsertReputation(ctx context.Context, row *reputationRow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_reputation (vendor, deal_count, total_discount_pct, max_discount_pct, success_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor) DO UPDATE SET
			deal_count=excluded.deal_count,
			total_discount_pct=excluded.total_discount_pct,
			max_discount_pct=excluded.max_discount_pct,
			success_count=excluded.success_count,
			updated_at=excluded.updated_at
	`, row.Vendor, row.DealCount, row.TotalDiscountPct, row.MaxDiscountPct, row.SuccessCount, row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert reputation: %w", err)
	}
	return nil
}

// listReputations returns all vendor reputation rows, up to limit (0 = no limit).
func (s *Store) listReputations(ctx context.Context, limit int) ([]reputationRow, error) {
	query := `SELECT vendor, deal_count, total_discount_pct, max_discount_pct, success_count, updated_at
		FROM vendor_reputation ORDER BY total_discount_pct DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list reputations: %w", err)
	}
	defer rows.Close()

	var result []reputationRow
	for rows.Next() {
		var row reputationRow
		if err := rows.Scan(&row.Vendor, &row.DealCount, &row.TotalDiscountPct, &row.MaxDiscountPct, &row.SuccessCount, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reputation: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
