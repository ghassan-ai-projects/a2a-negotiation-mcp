package marketplace

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Engine provides marketplace business logic.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a new marketplace engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		logger: logger,
	}
}

// ListSeats creates a new listing for unused SaaS seats.
// AskPrice must be less than OrigPrice (can't sell above list price).
func (e *Engine) ListSeats(ctx context.Context, vendor, sku string, seats int, origPrice, askPrice, minPrice float64, expiresInHours int) (*Listing, error) {
	if askPrice >= origPrice {
		return nil, fmt.Errorf("ask price (%.2f) must be less than original price (%.2f)", askPrice, origPrice)
	}
	if seats <= 0 {
		return nil, fmt.Errorf("seats must be positive")
	}
	if expiresInHours <= 0 {
		expiresInHours = 168 // 7 days default
	}

	now := time.Now().UTC()
	l := &Listing{
		Vendor:    vendor,
		SKU:       sku,
		Seats:     seats,
		OrigPrice: origPrice,
		AskPrice:  askPrice,
		MinPrice:  minPrice,
		Status:    "active",
		SellerID:  "default",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(expiresInHours) * time.Hour),
	}

	if err := e.store.CreateListing(ctx, l); err != nil {
		return nil, fmt.Errorf("list seats: %w", err)
	}

	e.logger.Info("marketplace listing created",
		"listing_id", l.ID, "vendor", vendor, "sku", sku,
		"seats", seats, "ask_price", askPrice)
	return l, nil
}

// Search finds active listings matching vendor and/or sku, sorted by ask_price ASC.
func (e *Engine) Search(ctx context.Context, vendor, sku string, maxSeats int) ([]Listing, error) {
	listings, err := e.store.SearchListings(ctx, vendor, sku, maxSeats)
	if err != nil {
		return nil, fmt.Errorf("search marketplace: %w", err)
	}
	return listings, nil
}

// MakeOffer places a buy offer on a listing.
// If the listing's ask_price <= buyer's max_price, the offer is auto-accepted
// and a transaction is created immediately.
func (e *Engine) MakeOffer(ctx context.Context, listingID, buyerID string, seats int, maxPrice float64) (*Offer, error) {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status != "active" {
		return nil, fmt.Errorf("listing %s is not active (status: %s)", listingID, listing.Status)
	}

	if seats > listing.Seats {
		return nil, fmt.Errorf("requested seats (%d) exceed available (%d)", seats, listing.Seats)
	}

	offer := &Offer{
		ListingID: listingID,
		BuyerID:   buyerID,
		Seats:     seats,
		MaxPrice:  maxPrice,
		Status:    "pending",
	}

	if err := e.store.AddOffer(ctx, offer); err != nil {
		return nil, fmt.Errorf("add offer: %w", err)
	}

	// Auto-accept if ask price <= buyer's max price
	if listing.AskPrice <= maxPrice {
		e.logger.Info("offer auto-accepted",
			"listing_id", listingID, "offer_id", offer.ID,
			"ask_price", listing.AskPrice, "max_price", maxPrice)

		// Use the listing's ask price as the final price per seat
		total := listing.AskPrice * float64(seats)
		platformFee := total * 0.05

		txn := &Transaction{
			ListingID:    listingID,
			Vendor:       listing.Vendor,
			SKU:          listing.SKU,
			Seats:        seats,
			PricePerSeat: listing.AskPrice,
			Total:        total,
			PlatformFee:  platformFee,
			SellerID:     listing.SellerID,
			BuyerID:      buyerID,
			Status:       "completed",
		}

		if err := e.store.CreateTransaction(ctx, txn); err != nil {
			return nil, fmt.Errorf("create transaction: %w", err)
		}

		// Update offer and listing status
		if err := e.store.UpdateOfferStatus(ctx, offer.ID, "accepted"); err != nil {
			return nil, fmt.Errorf("update offer status: %w", err)
		}

		if err := e.store.UpdateListingStatus(ctx, listingID, "completed"); err != nil {
			return nil, fmt.Errorf("update listing status: %w", err)
		}

		offer.Status = "accepted"

		e.logger.Info("marketplace transaction completed",
			"transaction_id", txn.ID, "listing_id", listingID,
			"total", total, "platform_fee", platformFee)
	}

	return offer, nil
}

// AcceptOffer finalizes an offer: creates a Transaction (with 5% platform fee)
// and updates the listing status to "completed".
func (e *Engine) AcceptOffer(ctx context.Context, listingID, offerID string) (*Transaction, error) {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status == "completed" {
		return nil, fmt.Errorf("listing %s is already completed", listingID)
	}

	offers, err := e.store.GetOffers(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("get offers: %w", err)
	}

	var targetOffer *Offer
	for _, o := range offers {
		if o.ID == offerID {
			targetOffer = &o
			break
		}
	}
	if targetOffer == nil {
		return nil, fmt.Errorf("offer %s not found on listing %s", offerID, listingID)
	}

	if targetOffer.Status != "pending" {
		return nil, fmt.Errorf("offer %s is not pending (status: %s)", offerID, targetOffer.Status)
	}

	// Use the offer's max price as the price per seat (capped at ask price)
	pricePerSeat := targetOffer.MaxPrice
	if pricePerSeat > listing.AskPrice {
		pricePerSeat = listing.AskPrice
	}

	total := pricePerSeat * float64(targetOffer.Seats)
	platformFee := total * 0.05

	txn := &Transaction{
		ListingID:    listingID,
		Vendor:       listing.Vendor,
		SKU:          listing.SKU,
		Seats:        targetOffer.Seats,
		PricePerSeat: pricePerSeat,
		Total:        total,
		PlatformFee:  platformFee,
		SellerID:     listing.SellerID,
		BuyerID:      targetOffer.BuyerID,
		Status:       "completed",
	}

	if err := e.store.CreateTransaction(ctx, txn); err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	if err := e.store.UpdateOfferStatus(ctx, offerID, "accepted"); err != nil {
		return nil, fmt.Errorf("update offer status: %w", err)
	}

	if err := e.store.UpdateListingStatus(ctx, listingID, "completed"); err != nil {
		return nil, fmt.Errorf("update listing status: %w", err)
	}

	e.logger.Info("marketplace offer accepted",
		"transaction_id", txn.ID, "listing_id", listingID,
		"offer_id", offerID, "total", total, "platform_fee", platformFee)

	return txn, nil
}

// Overview returns a summary of active listings count and recent transactions.
func (e *Engine) Overview(ctx context.Context) (map[string]any, error) {
	activeListings, err := e.store.GetActiveListings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active listings: %w", err)
	}

	recentTxns, err := e.store.GetRecentTransactions(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("get recent transactions: %w", err)
	}

	return map[string]any{
		"active_listings_count": len(activeListings),
		"active_listings":       activeListings,
		"recent_transactions":   recentTxns,
		"transaction_count":     len(recentTxns),
	}, nil
}
