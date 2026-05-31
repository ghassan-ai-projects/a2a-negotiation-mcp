package budgetalerts

// Level indicates budget consumption severity.
type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelCritical Level = "critical"
)

// BudgetAlert describes a single budget threshold result.
type BudgetAlert struct {
	Vendor      string  `json:"vendor"`
	Budget      float64 `json:"budget"`
	Actual      float64 `json:"actual"`
	ConsumedPct float64 `json:"consumed_pct"`
	Level       Level   `json:"level"`
	Action      string  `json:"action"`
}

// BudgetAlertHistory is a persisted record of a budget alert.
type BudgetAlertHistory struct {
	ID          int64   `json:"id"`
	Vendor      string  `json:"vendor"`
	Budget      float64 `json:"budget"`
	Actual      float64 `json:"actual"`
	ConsumedPct float64 `json:"consumed_pct"`
	Level       Level   `json:"level"`
	CreatedAt   string  `json:"created_at"`
}
