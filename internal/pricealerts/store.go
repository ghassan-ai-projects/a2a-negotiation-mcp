package pricealerts

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store provides SQLite persistence for price alert rules and baselines.
type Store struct {
	db *sql.DB
}

// NewStore creates a new pricealert store sharing the given DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("pricealerts migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS price_alerts (
		vendor         TEXT NOT NULL,
		sku            TEXT NOT NULL DEFAULT '',
		threshold_pct  REAL NOT NULL,
		channel        TEXT NOT NULL DEFAULT 'webhook',
		enabled        INTEGER NOT NULL DEFAULT 1,
		created_at     TEXT NOT NULL DEFAULT (datetime('now')),
		last_checked_at TEXT,
		PRIMARY KEY (vendor, sku)
	);
	CREATE TABLE IF NOT EXISTS price_baselines (
		vendor         TEXT NOT NULL,
		sku            TEXT NOT NULL DEFAULT '',
		baseline_price REAL NOT NULL,
		recorded_at    TEXT NOT NULL DEFAULT (datetime('now')),
		PRIMARY KEY (vendor, sku)
	);`
	_, err := s.db.Exec(schema)
	return err
}

// SetRule inserts or replaces a price alert rule.
func (s *Store) SetRule(ctx context.Context, rule *PriceAlertRule) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_alerts (vendor, sku, threshold_pct, channel, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor, sku) DO UPDATE SET
			threshold_pct=excluded.threshold_pct,
			channel=excluded.channel,
			enabled=excluded.enabled
	`, rule.Vendor, rule.SKU, rule.ThresholdPct, rule.Channel, boolToInt(rule.Enabled), rule.CreatedAt)
	return err
}

// ListRules returns all enabled price alert rules.
func (s *Store) ListRules(ctx context.Context) ([]PriceAlertRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT vendor, sku, threshold_pct, channel, enabled, created_at, COALESCE(last_checked_at, '')
		FROM price_alerts WHERE enabled = 1 ORDER BY vendor, sku`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []PriceAlertRule
	for rows.Next() {
		var r PriceAlertRule
		var enabledInt int
		if err := rows.Scan(&r.Vendor, &r.SKU, &r.ThresholdPct, &r.Channel, &enabledInt, &r.CreatedAt, &r.LastChecked); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		r.Enabled = enabledInt == 1
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// GetRule retrieves a single rule by vendor and sku.
func (s *Store) GetRule(ctx context.Context, vendor, sku string) (*PriceAlertRule, error) {
	var r PriceAlertRule
	var enabledInt int
	var lastChecked sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT vendor, sku, threshold_pct, channel, enabled, created_at, last_checked_at
		FROM price_alerts WHERE vendor = ? AND sku = ?`, vendor, sku).
		Scan(&r.Vendor, &r.SKU, &r.ThresholdPct, &r.Channel, &enabledInt, &r.CreatedAt, &lastChecked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	r.Enabled = enabledInt == 1
	if lastChecked.Valid {
		r.LastChecked = lastChecked.String
	}
	return &r, nil
}

// DeleteRule removes a price alert rule.
func (s *Store) DeleteRule(ctx context.Context, vendor, sku string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM price_alerts WHERE vendor = ? AND sku = ?`, vendor, sku)
	return err
}

// SetBaseline records the baseline price for a vendor/SKU.
func (s *Store) SetBaseline(ctx context.Context, vendor, sku string, price float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_baselines (vendor, sku, baseline_price, recorded_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(vendor, sku) DO UPDATE SET
			baseline_price=excluded.baseline_price,
			recorded_at=excluded.recorded_at
	`, vendor, sku, price, now)
	return err
}

// GetBaseline returns the recorded baseline price.
func (s *Store) GetBaseline(ctx context.Context, vendor, sku string) (float64, error) {
	var price float64
	err := s.db.QueryRowContext(ctx, `
		SELECT baseline_price FROM price_baselines WHERE vendor = ? AND sku = ?`, vendor, sku).Scan(&price)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get baseline: %w", err)
	}
	return price, nil
}

// UpdateLastChecked sets the last_checked_at timestamp for a rule.
func (s *Store) UpdateLastChecked(ctx context.Context, vendor, sku string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE price_alerts SET last_checked_at = ? WHERE vendor = ? AND sku = ?`, now, vendor, sku)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
