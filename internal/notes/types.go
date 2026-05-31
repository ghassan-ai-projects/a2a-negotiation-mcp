package notes

// NegotiationNote represents a note attached to a negotiation session.
type NegotiationNote struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}
