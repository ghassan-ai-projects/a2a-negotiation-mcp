package budget

import "time"

// VendorBudget represents a budget target for a single vendor.
type VendorBudget struct {
	Vendor       string  `json:"vendor"`
	BudgetAmount float64 `json:"budget_amount"`
	PeriodMonth  string  `json:"period_month"` // "2026-05"
}

// BudgetTrend represents a single monthly data point in the dashboard trend.
type BudgetTrend struct {
	Month  string  `json:"month"` // "2026-05"
	Budget float64 `json:"budget"`
	Actual float64 `json:"actual"`
}

// Warning describes an overspend alert.
type Warning struct {
	Vendor      string  `json:"vendor"`
	Budget      float64 `json:"budget"`
	Actual      float64 `json:"actual"`
	VariancePct float64 `json:"variance_pct"`
	Message     string  `json:"message"`
}

// BudgetDashboard is the full dashboard response.
type BudgetDashboard struct {
	Period       string         `json:"period"`
	TotalBudget  float64        `json:"total_budget"`
	TotalActual  float64        `json:"total_actual"`
	Variance     float64        `json:"variance"`
	VariancePct  float64        `json:"variance_pct"`
	ByVendor     []VendorBudget `json:"by_vendor"`
	MonthlyTrend []BudgetTrend  `json:"monthly_trend"`
	Warnings     []Warning      `json:"warnings"`
}

// SetBudgetResult is returned by the SetBudget MCP tool.
type SetBudgetResult struct {
	Vendor string  `json:"vendor"`
	Budget float64 `json:"budget"`
	Status string  `json:"status"`
}

// DeleteBudgetResult is returned by the DeleteBudget MCP tool.
type DeleteBudgetResult struct {
	Vendor string `json:"vendor"`
	Status string `json:"status"`
}

// budgetRow is an internal DB row.
type budgetRow struct {
	ID           int64
	Vendor       string
	BudgetAmount float64
	PeriodMonth  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
