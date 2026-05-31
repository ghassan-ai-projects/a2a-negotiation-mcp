package costallocation

import (
	"context"
	"database/sql"
	"fmt"
	"math"
)

// Engine manages cost allocation calculations and reporting.
type Engine struct {
	store *Store
	db    *sql.DB
}

// NewEngine creates a cost allocation engine.
func NewEngine(store *Store, db *sql.DB) *Engine {
	return &Engine{
		store: store,
		db:    db,
	}
}

// SetAllocation saves a cost allocation for a vendor/department.
func (e *Engine) SetAllocation(ctx context.Context, vendor, department string, pct float64) (*CostAllocation, error) {
	return e.store.Set(ctx, vendor, department, pct)
}

// Report generates a cost allocation report for a given period.
func (e *Engine) Report(ctx context.Context, period string) (*AllocationReport, error) {
	if period == "" {
		period = "90d"
	}

	allocations, err := e.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list allocations: %w", err)
	}

	if len(allocations) == 0 {
		return &AllocationReport{
			Period:       period,
			TotalSpend:   0,
			ByDepartment: []DeptAllocation{},
			ByVendorDept: []VendorDeptAllocation{},
		}, nil
	}

	// Build a set of unique vendors
	vendorSet := make(map[string]bool)
	for _, a := range allocations {
		vendorSet[a.Vendor] = true
	}

	// For each vendor, get total spend from deal_outcomes
	vendorSpend := make(map[string]float64)
	for vendor := range vendorSet {
		var total sql.NullFloat64
		var periodClause string
		switch period {
		case "30d":
			periodClause = "AND created_at >= datetime('now', '-30 days')"
		case "90d":
			periodClause = "AND created_at >= datetime('now', '-90 days')"
		case "1y":
			periodClause = "AND created_at >= datetime('now', '-1 year')"
		default:
			periodClause = "AND created_at >= datetime('now', '-90 days')"
		}

		err := e.db.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(SUM(final_price * seats), 0)
			FROM deal_outcomes
			WHERE vendor = ? %s
		`, periodClause), vendor).Scan(&total)
		if err != nil {
			vendorSpend[vendor] = 0
		} else {
			vendorSpend[vendor] = total.Float64
		}
	}

	// Compute total spend across all vendors
	var totalSpend float64
	for _, spend := range vendorSpend {
		totalSpend += spend
	}

	// Build department allocations
	deptMap := make(map[string]float64)
	for _, a := range allocations {
		allocated := vendorSpend[a.Vendor] * a.AllocationPct / 100.0
		deptMap[a.Department] += allocated
	}

	var byDept []DeptAllocation
	for dept, spend := range deptMap {
		spend = math.Round(spend*100) / 100
		var pctOfTotal float64
		if totalSpend > 0 {
			pctOfTotal = math.Round(spend/totalSpend*10000) / 100
		}
		byDept = append(byDept, DeptAllocation{
			Department: dept,
			TotalSpend: spend,
			PctOfTotal: pctOfTotal,
		})
	}
	if byDept == nil {
		byDept = []DeptAllocation{}
	}

	// Build vendor-per-department breakdown
	var byVendorDept []VendorDeptAllocation
	for _, a := range allocations {
		amount := math.Round(vendorSpend[a.Vendor]*a.AllocationPct/100.0*100) / 100
		byVendorDept = append(byVendorDept, VendorDeptAllocation{
			Vendor:     a.Vendor,
			Department: a.Department,
			Amount:     amount,
			Pct:        a.AllocationPct,
		})
	}
	if byVendorDept == nil {
		byVendorDept = []VendorDeptAllocation{}
	}

	return &AllocationReport{
		Period:       period,
		TotalSpend:   math.Round(totalSpend*100) / 100,
		ByDepartment: byDept,
		ByVendorDept: byVendorDept,
	}, nil
}

// DepartmentReport generates a report for a single department.
func (e *Engine) DepartmentReport(ctx context.Context, department, period string) (*AllocationReport, error) {
	if period == "" {
		period = "90d"
	}

	allocations, err := e.store.ListByDepartment(ctx, department)
	if err != nil {
		return nil, fmt.Errorf("list by department: %w", err)
	}

	if len(allocations) == 0 {
		return &AllocationReport{
			Period:       period,
			TotalSpend:   0,
			ByDepartment: []DeptAllocation{},
			ByVendorDept: []VendorDeptAllocation{},
		}, nil
	}

	vendorSet := make(map[string]bool)
	for _, a := range allocations {
		vendorSet[a.Vendor] = true
	}

	vendorSpend := make(map[string]float64)
	for vendor := range vendorSet {
		var total sql.NullFloat64
		var periodClause string
		switch period {
		case "30d":
			periodClause = "AND created_at >= datetime('now', '-30 days')"
		case "90d":
			periodClause = "AND created_at >= datetime('now', '-90 days')"
		case "1y":
			periodClause = "AND created_at >= datetime('now', '-1 year')"
		default:
			periodClause = "AND created_at >= datetime('now', '-90 days')"
		}

		err := e.db.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(SUM(final_price * seats), 0)
			FROM deal_outcomes
			WHERE vendor = ? %s
		`, periodClause), vendor).Scan(&total)
		if err != nil {
			vendorSpend[vendor] = 0
		} else {
			vendorSpend[vendor] = total.Float64
		}
	}

	var totalSpend float64
	for _, spend := range vendorSpend {
		totalSpend += spend
	}

	deptTotal := 0.0
	var byVendorDept []VendorDeptAllocation
	for _, a := range allocations {
		amount := math.Round(vendorSpend[a.Vendor]*a.AllocationPct/100.0*100) / 100
		deptTotal += amount
		byVendorDept = append(byVendorDept, VendorDeptAllocation{
			Vendor:     a.Vendor,
			Department: a.Department,
			Amount:     amount,
			Pct:        a.AllocationPct,
		})
	}
	if byVendorDept == nil {
		byVendorDept = []VendorDeptAllocation{}
	}

	deptTotal = math.Round(deptTotal*100) / 100
	var pctOfTotal float64
	if totalSpend > 0 {
		pctOfTotal = math.Round(deptTotal/totalSpend*10000) / 100
	}

	return &AllocationReport{
		Period:     period,
		TotalSpend: math.Round(totalSpend*100) / 100,
		ByDepartment: []DeptAllocation{{
			Department: department,
			TotalSpend: deptTotal,
			PctOfTotal: pctOfTotal,
		}},
		ByVendorDept: byVendorDept,
	}, nil
}
