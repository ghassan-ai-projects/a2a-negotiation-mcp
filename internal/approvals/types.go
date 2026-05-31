package approvals

// ApprovalStatus represents the current state of an approval request.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// Approval represents a request for approval on a negotiation action.
type Approval struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	Reason     string         `json:"reason"`
	Threshold  float64        `json:"threshold"`
	Status     ApprovalStatus `json:"status"`
	CreatedAt  string         `json:"created_at"`
	ResolvedAt *string        `json:"resolved_at,omitempty"`
}
