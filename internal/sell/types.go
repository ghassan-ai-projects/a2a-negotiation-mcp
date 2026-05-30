package sell

import "time"

// Listing represents a sell-side listing posted by a user.
type Listing struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	DesiredPrice float64   `json:"desired_price"`
	MinPrice     float64   `json:"min_price,omitempty"`
	Strategy     string    `json:"strategy"` // "auction", "haggling", "fixed"
	Status       string    `json:"status"`   // "active", "negotiating", "sold", "withdrawn"
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// Bid represents a buyer's bid on a listing.
type Bid struct {
	ID        string    `json:"id"`
	ListingID string    `json:"listing_id"`
	BidderID  string    `json:"bidder_id"`
	Amount    float64   `json:"amount"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SellResult is returned when a sale is completed.
type SellResult struct {
	Listing    Listing `json:"listing"`
	WinningBid Bid     `json:"winning_bid,omitempty"`
	Status     string  `json:"status"`
	FinalPrice float64 `json:"final_price,omitempty"`
}

// ListingStatus is returned by listing status queries.
type ListingStatus struct {
	Listing Listing `json:"listing"`
	Bids    []Bid   `json:"bids,omitempty"`
}
