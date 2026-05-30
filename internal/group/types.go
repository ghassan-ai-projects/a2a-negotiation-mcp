package group

import "time"

// BuyingGroup represents a collective buying group targeting a specific vendor SKU.
type BuyingGroup struct {
	ID           string    `json:"id"`
	TargetVendor string    `json:"target_vendor"`
	TargetSKU    string    `json:"target_sku"`
	TargetPrice  float64   `json:"target_price,omitempty"`
	MinMembers   int       `json:"min_members"`
	Status       string    `json:"status"` // "forming", "negotiating", "completed", "expired"
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// GroupMember represents a single user's commitment within a buying group.
type GroupMember struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	UserID      string    `json:"user_id"`
	Quantity    int       `json:"quantity"`
	MaxPrice    float64   `json:"max_price,omitempty"`
	CommittedAt time.Time `json:"committed_at"`
}

// ConsolidatedOffer is the computed collective pricing offer for a group.
type ConsolidatedOffer struct {
	GroupID        string  `json:"group_id"`
	Vendor         string  `json:"vendor"`
	SKU            string  `json:"sku"`
	TotalDemand    int     `json:"total_demand"`
	MemberCount    int     `json:"member_count"`
	ListPrice      float64 `json:"list_price"`
	OfferPrice     float64 `json:"offer_price"`
	SavingsPerUnit float64 `json:"savings_per_unit"`
	DiscountPct    float64 `json:"discount_pct"`
}
