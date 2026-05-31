package auditlog

import "time"

// AuditEntry represents a single auditable action.
type AuditEntry struct {
	ID        int64     `json:"id"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditSummary provides aggregated audit data.
type AuditSummary struct {
	TotalActions int64            `json:"total_actions"`
	ByAction     map[string]int64 `json:"by_action"`
	ByDay        map[string]int64 `json:"by_day"`
}
