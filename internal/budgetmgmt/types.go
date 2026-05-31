package budgetmgmt

// MonthlyBudget represents per-month budget allocation for a vendor.
type MonthlyBudget struct {
	Vendor       string  `json:"vendor"`
	Month        string  `json:"month"`
	BudgetAmount float64 `json:"budget_amount"`
	Spent        float64 `json:"spent"`
	RolledOver   float64 `json:"rolled_over"`
	CreatedAt    string  `json:"created_at"`
}

// BudgetForecast shows YTD budget vs actual and projects annual spend.
type BudgetForecast struct {
	Vendor          string  `json:"vendor"`
	YTDBudget       float64 `json:"ytd_budget"`
	YTDSpent        float64 `json:"ytd_spent"`
	ProjectedAnnual float64 `json:"projected_annual"`
	RemainingMonths int     `json:"remaining_months"`
	Status          string  `json:"status"`
}

// BudgetDashboard is the per-month dashboard response.
type BudgetDashboard struct {
	Month       string          `json:"month"`
	TotalBudget float64         `json:"total_budget"`
	TotalSpent  float64         `json:"total_spent"`
	Items       []MonthlyBudget `json:"items"`
}

// SetBudgetResult is the response for setting a monthly budget.
type SetBudgetResult struct {
	Vendor string  `json:"vendor"`
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// RolloverResult is the response for rolling over unused budget.
type RolloverResult struct {
	Vendor    string  `json:"vendor"`
	FromMonth string  `json:"from_month"`
	ToMonth   string  `json:"to_month"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
}

// monthlyBudgetRow is the internal DB row.
type monthlyBudgetRow struct {
	ID           int64
	Vendor       string
	Month        string
	BudgetAmount float64
	Spent        float64
	RolledOver   float64
	CreatedAt    string
}
