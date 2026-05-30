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
