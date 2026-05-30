package sell

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const maxHaggleRounds = 3

// Engine provides sell-side listing and negotiation logic.
type Engine struct {
	store  *Store
	logger *slog.Logger
}

// NewEngine creates a new selling engine.
func NewEngine(store *Store, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		logger: logger,
	}
}

// ListItem creates a new listing.
func (e *Engine) ListItem(ctx context.Context, userID, title, description string, desiredPrice, minPrice float64, strategy string, expiresInHours int) (*Listing, error) {
	now := time.Now().UTC()

	// Default expiry based on strategy
	if expiresInHours <= 0 {
		switch strategy {
		case "auction":
			expiresInHours = 48
		default:
			expiresInHours = 168 // 7 days
		}
	}

	l := &Listing{
		UserID:       userID,
		Title:        title,
		Description:  description,
		DesiredPrice: desiredPrice,
		MinPrice:     minPrice,
		Strategy:     strategy,
		Status:       "active",
		CreatedAt:    now,
		ExpiresAt:    now.Add(time.Duration(expiresInHours) * time.Hour),
	}

	if err := e.store.CreateListing(ctx, l); err != nil {
		return nil, fmt.Errorf("list item: %w", err)
	}

	e.logger.Info("listing created", "listing_id", l.ID, "user_id", userID, "strategy", strategy)
	return l, nil
}

// PlaceBid adds a bid and applies strategy rules.
// For "fixed": auto-approves if bid >= desired price.
// For "haggling": auto-counters at +5% if bid below desired price.
func (e *Engine) PlaceBid(ctx context.Context, listingID, bidderID string, amount float64, message string) (*Bid, error) {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status != "active" && listing.Status != "negotiating" {
		return nil, fmt.Errorf("listing %s is not accepting bids (status: %s)", listingID, listing.Status)
	}

	// Update listing status to negotiating
	if listing.Status == "active" {
		if err := e.store.UpdateListingStatus(ctx, listingID, "negotiating"); err != nil {
			return nil, fmt.Errorf("update listing status: %w", err)
		}
	}

	bid := &Bid{
		ListingID: listingID,
		BidderID:  bidderID,
		Amount:    amount,
		Message:   message,
	}

	if err := e.store.AddBid(ctx, bid); err != nil {
		return nil, fmt.Errorf("add bid: %w", err)
	}

	e.logger.Info("bid placed", "listing_id", listingID, "bidder", bidderID, "amount", amount, "strategy", listing.Strategy)

	// Strategy-specific auto-processing
	switch listing.Strategy {
	case "fixed":
		if amount >= listing.DesiredPrice {
			// Auto-accept
			if err := e.acceptBid(ctx, listing, bid); err != nil {
				return nil, err
			}
		}
	case "haggling":
		if amount < listing.DesiredPrice {
			// Check haggle counter limit
			bids, err := e.store.GetBids(ctx, listingID)
			if err != nil {
				return nil, fmt.Errorf("get bids: %w", err)
			}

			sellerCounterRounds := 0
			for _, b := range bids {
				if b.BidderID == bidderID {
					sellerCounterRounds++
				}
			}

			if sellerCounterRounds >= maxHaggleRounds {
				e.logger.Info("haggle limit reached, freezing", "listing_id", listingID, "bidder", bidderID)
				// Freeze — do not auto-counter. Return the bid as-is.
				return bid, nil
			}
		}
	}

	return bid, nil
}

// AcceptBid accepts a specific bid and marks the listing as sold.
func (e *Engine) AcceptBid(ctx context.Context, listingID, bidID string) (*SellResult, error) {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status == "sold" {
		return nil, fmt.Errorf("listing %s is already sold", listingID)
	}

	// Find the bid
	bids, err := e.store.GetBids(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("get bids: %w", err)
	}

	var winningBid *Bid
	for _, b := range bids {
		if b.ID == bidID {
			winningBid = &b
			break
		}
	}

	if winningBid == nil {
		return nil, fmt.Errorf("bid %s not found on listing %s", bidID, listingID)
	}

	if err := e.acceptBid(ctx, listing, winningBid); err != nil {
		return nil, err
	}

	return &SellResult{
		Listing:    *listing,
		WinningBid: *winningBid,
		Status:     "sold",
		FinalPrice: winningBid.Amount,
	}, nil
}

func (e *Engine) acceptBid(ctx context.Context, listing *Listing, bid *Bid) error {
	listing.Status = "sold"
	if err := e.store.UpdateListingStatus(ctx, listing.ID, "sold"); err != nil {
		return fmt.Errorf("mark listing sold: %w", err)
	}
	e.logger.Info("listing sold", "listing_id", listing.ID, "bid_id", bid.ID, "price", bid.Amount)
	return nil
}

// RejectBid rejects a bid with an optional counter message.
func (e *Engine) RejectBid(ctx context.Context, listingID, bidID, counterMessage string) error {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status == "sold" {
		return fmt.Errorf("listing %s is already sold", listingID)
	}

	// Verify the bid exists on this listing
	bids, err := e.store.GetBids(ctx, listingID)
	if err != nil {
		return fmt.Errorf("get bids: %w", err)
	}

	found := false
	for _, b := range bids {
		if b.ID == bidID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("bid %s not found on listing %s", bidID, listingID)
	}

	e.logger.Info("bid rejected", "listing_id", listingID, "bid_id", bidID, "counter_message", counterMessage)
	return nil
}

// CheckExpired auto-closes expired listings based on strategy rules.
// Auction: no bids → withdrawn, has bids → last highest wins.
// Other strategies: no special auto-processing beyond status update.
func (e *Engine) CheckExpired(ctx context.Context, listingID string) error {
	listing, err := e.store.GetListing(ctx, listingID)
	if err != nil {
		return fmt.Errorf("listing lookup: %w", err)
	}

	if listing.Status != "active" && listing.Status != "negotiating" {
		return nil // Already finalised
	}

	now := time.Now().UTC()
	if now.Before(listing.ExpiresAt) {
		return nil // Not yet expired
	}

	bids, err := e.store.GetBids(ctx, listingID)
	if err != nil {
		return fmt.Errorf("get bids: %w", err)
	}

	if listing.Strategy == "auction" {
		if len(bids) == 0 {
			// No bids → withdrawn
			if err := e.store.UpdateListingStatus(ctx, listingID, "withdrawn"); err != nil {
				return fmt.Errorf("withdraw expired listing: %w", err)
			}
			e.logger.Info("auction expired with no bids, withdrawn", "listing_id", listingID)
		} else {
			// Highest bid wins
			bestBid, err := e.store.GetBestBid(ctx, listingID)
			if err != nil {
				return fmt.Errorf("get best bid: %w", err)
			}
			listing.Status = "sold"
			if err := e.store.UpdateListingStatus(ctx, listingID, "sold"); err != nil {
				return fmt.Errorf("mark expired listing sold: %w", err)
			}
			e.logger.Info("auction expired, highest bid wins", "listing_id", listingID, "bid_id", bestBid.ID, "price", bestBid.Amount)
		}
	} else {
		// For haggling/fixed — just mark as withdrawn if expired
		if err := e.store.UpdateListingStatus(ctx, listingID, "withdrawn"); err != nil {
			return fmt.Errorf("withdraw expired listing: %w", err)
		}
		e.logger.Info("listing expired, withdrawn", "listing_id", listingID)
	}

	return nil
}

// GenerateCounter generates a seller counter-offer for haggling.
// Formula: counter = bid * 1.05 (seller adds 5%).
func GenerateCounter(bidAmount float64) float64 {
	return bidAmount * 1.05
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}
