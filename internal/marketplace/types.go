package marketplace

import "time"

// Listing represents a used SaaS seat listing posted by a seller.
type Listing struct {
	ID        string    `json:"id"`
	Vendor    string    `json:"vendor"`
	SKU       string    `json:"sku"`
	Seats     int       `json:"seats"`
	OrigPrice float64   `json:"orig_price"`
	AskPrice  float64   `json:"ask_price"`
	MinPrice  float64   `json:"min_price"`
	Status    string    `json:"status"` // "active", "dealmaking", "completed", "expired"
	SellerID  string    `json:"seller_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Offer represents a buyer's offer on a listing.
type Offer struct {
	ID        string    `json:"id"`
	ListingID string    `json:"listing_id"`
	BuyerID   string    `json:"buyer_id"`
	Seats     int       `json:"seats"`
	MaxPrice  float64   `json:"max_price"`
	Status    string    `json:"status"` // "pending", "accepted", "rejected"
	CreatedAt time.Time `json:"created_at"`
}

// Transaction represents a completed marketplace transaction.
type Transaction struct {
	ID           string    `json:"id"`
	ListingID    string    `json:"listing_id"`
	Vendor       string    `json:"vendor"`
	SKU          string    `json:"sku"`
	Seats        int       `json:"seats"`
	PricePerSeat float64   `json:"price_per_seat"`
	Total        float64   `json:"total"`
	PlatformFee  float64   `json:"platform_fee"`
	SellerID     string    `json:"seller_id"`
	BuyerID      string    `json:"buyer_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
