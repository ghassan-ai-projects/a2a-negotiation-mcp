package autotrigger

// Trigger represents a configured auto-negotiation trigger.
type Trigger struct {
	ID        int    `json:"id"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Vendor    string `json:"vendor"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// TriggerLogEntry represents a log entry of a triggered action.
type TriggerLogEntry struct {
	ID        int    `json:"id"`
	TriggerID int    `json:"trigger_id"`
	FiredAt   string `json:"fired_at"`
	Outcome   string `json:"outcome"`
}
