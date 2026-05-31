package costallocation

// CostAllocation represents a vendor cost allocation to a department.
type CostAllocation struct {
	ID            int64   `json:"id"`
	Vendor        string  `json:"vendor"`
	Department    string  `json:"department"`
	AllocationPct float64 `json:"allocation_pct"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// AllocationReport is the full cost allocation report.
type AllocationReport struct {
	Period       string                    `json:"period"`
	TotalSpend   float64                   `json:"total_spend"`
	ByDepartment []DeptAllocation          `json:"by_department"`
	ByVendorDept []VendorDeptAllocation    `json:"by_vendor_per_dept"`
}

// DeptAllocation is a department-level allocation summary.
type DeptAllocation struct {
	Department string  `json:"department"`
	TotalSpend float64 `json:"total_spend"`
	PctOfTotal float64 `json:"pct_of_total"`
}

// VendorDeptAllocation is a per-vendor, per-department allocation detail.
type VendorDeptAllocation struct {
	Vendor     string  `json:"vendor"`
	Department string  `json:"department"`
	Amount     float64 `json:"amount"`
	Pct        float64 `json:"pct"`
}
