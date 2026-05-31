package costallocation

import (
	"context"
	"database/sql"
	"fmt"
)

// Store manages cost allocation records in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a cost allocation store.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate costallocation: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cost_allocations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		department TEXT NOT NULL,
		allocation_pct REAL NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(vendor, department)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Set inserts or updates a cost allocation for a vendor/department.
func (s *Store) Set(ctx context.Context, vendor, department string, allocationPct float64) (*CostAllocation, error) {
	now := "datetime('now')"
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO cost_allocations (vendor, department, allocation_pct, created_at, updated_at)
		VALUES (?, ?, ?, `+now+`, `+now+`)
		ON CONFLICT(vendor, department) DO UPDATE SET
			allocation_pct=excluded.allocation_pct,
			updated_at=`+now+`
	`, vendor, department, allocationPct)
	if err != nil {
		return nil, fmt.Errorf("set allocation: %w", err)
	}

	_, _ = result.LastInsertId()
	return s.Get(ctx, vendor, department)
}

// Get retrieves a cost allocation by vendor and department.
func (s *Store) Get(ctx context.Context, vendor, department string) (*CostAllocation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, department, allocation_pct, created_at, updated_at
		FROM cost_allocations
		WHERE vendor = ? AND department = ?
	`, vendor, department)

	var ca CostAllocation
	if err := row.Scan(&ca.ID, &ca.Vendor, &ca.Department, &ca.AllocationPct, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get allocation: %w", err)
	}
	return &ca, nil
}

// ListByVendor returns all allocations for a vendor.
func (s *Store) ListByVendor(ctx context.Context, vendor string) ([]CostAllocation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, department, allocation_pct, created_at, updated_at
		FROM cost_allocations
		WHERE vendor = ?
		ORDER BY department
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("list by vendor: %w", err)
	}
	defer rows.Close()

	var results []CostAllocation
	for rows.Next() {
		var ca CostAllocation
		if err := rows.Scan(&ca.ID, &ca.Vendor, &ca.Department, &ca.AllocationPct, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan allocation: %w", err)
		}
		results = append(results, ca)
	}
	if results == nil {
		results = []CostAllocation{}
	}
	return results, rows.Err()
}

// ListByDepartment returns all allocations for a department.
func (s *Store) ListByDepartment(ctx context.Context, department string) ([]CostAllocation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, department, allocation_pct, created_at, updated_at
		FROM cost_allocations
		WHERE department = ?
		ORDER BY vendor
	`, department)
	if err != nil {
		return nil, fmt.Errorf("list by department: %w", err)
	}
	defer rows.Close()

	var results []CostAllocation
	for rows.Next() {
		var ca CostAllocation
		if err := rows.Scan(&ca.ID, &ca.Vendor, &ca.Department, &ca.AllocationPct, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan allocation: %w", err)
		}
		results = append(results, ca)
	}
	if results == nil {
		results = []CostAllocation{}
	}
	return results, rows.Err()
}

// Delete removes a cost allocation.
func (s *Store) Delete(ctx context.Context, vendor, department string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cost_allocations
		WHERE vendor = ? AND department = ?
	`, vendor, department)
	return err
}

// ListAll returns all cost allocations.
func (s *Store) ListAll(ctx context.Context) ([]CostAllocation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, vendor, department, allocation_pct, created_at, updated_at
		FROM cost_allocations
		ORDER BY vendor, department
	`)
	if err != nil {
		return nil, fmt.Errorf("list all: %w", err)
	}
	defer rows.Close()

	var results []CostAllocation
	for rows.Next() {
		var ca CostAllocation
		if err := rows.Scan(&ca.ID, &ca.Vendor, &ca.Department, &ca.AllocationPct, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan allocation: %w", err)
		}
		results = append(results, ca)
	}
	if results == nil {
		results = []CostAllocation{}
	}
	return results, rows.Err()
}
