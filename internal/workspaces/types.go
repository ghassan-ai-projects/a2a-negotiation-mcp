package workspaces

import "time"

// Workspace represents a team workspace for collaborative negotiation.
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceSummary provides aggregated data for a workspace.
type WorkspaceSummary struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	VendorCount  int     `json:"vendor_count"`
	DealCount    int     `json:"deal_count"`
	TotalSavings float64 `json:"total_savings"`
	MemberCount  int     `json:"member_count"`
}
