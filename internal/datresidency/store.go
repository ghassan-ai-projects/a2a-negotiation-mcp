package datresidency

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages data residency rules in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a DataResidencyStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate datresidency: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS data_residency_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		region TEXT NOT NULL UNIQUE,
		allowed INTEGER NOT NULL DEFAULT 1,
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetRule creates or replaces a data residency rule for a region.
func (s *Store) SetRule(ctx context.Context, region string, allowed bool) (*ResidencyRule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO data_residency_rules (region, allowed, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(region) DO UPDATE SET
			allowed = excluded.allowed,
			updated_at = excluded.updated_at
	`, region, allowed, now)
	if err != nil {
		return nil, fmt.Errorf("set rule: %w", err)
	}
	return s.GetRule(ctx, region)
}

// GetRule returns the residency rule for a given region.
func (s *Store) GetRule(ctx context.Context, region string) (*ResidencyRule, error) {
	var r ResidencyRule
	err := s.db.QueryRowContext(ctx, `
		SELECT id, region, allowed, updated_at
		FROM data_residency_rules WHERE region = ?
	`, region).Scan(&r.ID, &r.Region, &r.Allowed, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: region=%s", region)
	}
	if err != nil {
		return nil, fmt.Errorf("get rule: %w", err)
	}
	return &r, nil
}

// ListRules returns all data residency rules.
func (s *Store) ListRules(ctx context.Context) ([]ResidencyRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, region, allowed, updated_at
		FROM data_residency_rules
		ORDER BY region ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []ResidencyRule
	for rows.Next() {
		var r ResidencyRule
		if err := rows.Scan(&r.ID, &r.Region, &r.Allowed, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []ResidencyRule{}
	}
	return rules, nil
}

// CheckVendor checks if a vendor's region is compliant with data residency rules.
func (s *Store) CheckVendor(ctx context.Context, vendor, region string) (*VendorResidencyCheck, error) {
	rule, err := s.GetRule(ctx, region)
	if err != nil {
		// Rule not found → not compliant
		return &VendorResidencyCheck{
			Vendor:    vendor,
			Region:    region,
			Compliant: false,
			RuleFound: false,
		}, nil
	}
	return &VendorResidencyCheck{
		Vendor:    vendor,
		Region:    region,
		Compliant: rule.Allowed,
		RuleFound: true,
	}, nil
}
