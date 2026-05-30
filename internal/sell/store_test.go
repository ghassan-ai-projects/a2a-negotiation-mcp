package sell

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestStore creates an in-memory SQLite store.
func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", "file:sell_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	return store
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func createTestListing(t *testing.T, store *Store, ctx context.Context) *Listing {
	t.Helper()
	l := &Listing{
		UserID:       "seller1",
		Title:        "Test Item",
		Description:  "A test item for sale",
		DesiredPrice: 100.00,
		MinPrice:     80.00,
		Strategy:     "fixed",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(72 * time.Hour),
	}
	if err := store.CreateListing(ctx, l); err != nil {
		t.Fatalf("CreateListing: %v", err)
	}
	return l
}

// ─── Store Tests ───

func TestCreateListing(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	got, err := store.GetListing(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if got.Title != "Test Item" {
		t.Errorf("expected title 'Test Item', got %s", got.Title)
	}
	if got.UserID != "seller1" {
		t.Errorf("expected user_id seller1, got %s", got.UserID)
	}
	if got.DesiredPrice != 100.00 {
		t.Errorf("expected desired_price 100.00, got %f", got.DesiredPrice)
	}
	if got.MinPrice != 80.00 {
		t.Errorf("expected min_price 80.00, got %f", got.MinPrice)
	}
	if got.Strategy != "fixed" {
		t.Errorf("expected strategy fixed, got %s", got.Strategy)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if got.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}
}

func TestGetListing_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetListing(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent listing")
	}
}

func TestAddBid(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	b := &Bid{
		ListingID: l.ID,
		BidderID:  "buyer1",
		Amount:    85.00,
		Message:   "Interested in this item",
	}
	if err := store.AddBid(ctx, b); err != nil {
		t.Fatalf("AddBid: %v", err)
	}

	bids, err := store.GetBids(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetBids: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("expected 1 bid, got %d", len(bids))
	}
	if bids[0].BidderID != "buyer1" {
		t.Errorf("expected bidder_id buyer1, got %s", bids[0].BidderID)
	}
	if bids[0].Amount != 85.00 {
		t.Errorf("expected amount 85.00, got %f", bids[0].Amount)
	}
	if bids[0].Message != "Interested in this item" {
		t.Errorf("expected message, got %s", bids[0].Message)
	}
}

func TestAddBid_MultipleBids(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	b1 := &Bid{ListingID: l.ID, BidderID: "buyer1", Amount: 80.00}
	b2 := &Bid{ListingID: l.ID, BidderID: "buyer2", Amount: 90.00}
	b3 := &Bid{ListingID: l.ID, BidderID: "buyer3", Amount: 85.00}

	for _, b := range []*Bid{b1, b2, b3} {
		if err := store.AddBid(ctx, b); err != nil {
			t.Fatalf("AddBid: %v", err)
		}
	}

	bids, err := store.GetBids(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetBids: %v", err)
	}
	if len(bids) != 3 {
		t.Fatalf("expected 3 bids, got %d", len(bids))
	}
	// Should be ordered by amount DESC
	if bids[0].Amount != 90.00 {
		t.Errorf("expected first bid 90.00 (highest), got %f", bids[0].Amount)
	}
	if bids[1].Amount != 85.00 {
		t.Errorf("expected second bid 85.00, got %f", bids[1].Amount)
	}
	if bids[2].Amount != 80.00 {
		t.Errorf("expected third bid 80.00, got %f", bids[2].Amount)
	}
}

func TestUpdateListingStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	if err := store.UpdateListingStatus(ctx, l.ID, "sold"); err != nil {
		t.Fatalf("UpdateListingStatus: %v", err)
	}

	got, err := store.GetListing(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetListing after update: %v", err)
	}
	if got.Status != "sold" {
		t.Errorf("expected status sold, got %s", got.Status)
	}
}

func TestUpdateListingStatus_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.UpdateListingStatus(ctx, "nonexistent", "sold")
	if err == nil {
		t.Fatal("expected error for nonexistent listing")
	}
}

func TestListListings_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	listings, err := store.ListListings(ctx, "")
	if err != nil {
		t.Fatalf("ListListings: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("expected 0 listings, got %d", len(listings))
	}
}

func TestListListings_FilterByStrategy(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create two active listings with different strategies
	l1 := &Listing{
		UserID: "seller1", Title: "Auction Item", DesiredPrice: 200, Strategy: "auction",
		Status: "active", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(48 * time.Hour),
	}
	l2 := &Listing{
		UserID: "seller2", Title: "Fixed Item", DesiredPrice: 50, Strategy: "fixed",
		Status: "active", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}
	// Sold listing should not appear
	l3 := &Listing{
		UserID: "seller3", Title: "Sold Item", DesiredPrice: 30, Strategy: "fixed",
		Status: "sold", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}

	for _, l := range []*Listing{l1, l2, l3} {
		if err := store.CreateListing(ctx, l); err != nil {
			t.Fatalf("CreateListing: %v", err)
		}
	}

	// All active
	all, err := store.ListListings(ctx, "")
	if err != nil {
		t.Fatalf("ListListings: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 active listings, got %d", len(all))
	}

	// Filtered by auction
	auctionListings, err := store.ListListings(ctx, "auction")
	if err != nil {
		t.Fatalf("ListListings(auction): %v", err)
	}
	if len(auctionListings) != 1 {
		t.Errorf("expected 1 auction listing, got %d", len(auctionListings))
	}
	if auctionListings[0].Title != "Auction Item" {
		t.Errorf("expected 'Auction Item', got %s", auctionListings[0].Title)
	}
}

func TestGetBestBid(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	// No bids yet
	_, err := store.GetBestBid(ctx, l.ID)
	if err == nil {
		t.Fatal("expected error for no bids")
	}

	// Add bids
	b1 := &Bid{ListingID: l.ID, BidderID: "buyer1", Amount: 80.00}
	b2 := &Bid{ListingID: l.ID, BidderID: "buyer2", Amount: 95.00}
	b3 := &Bid{ListingID: l.ID, BidderID: "buyer3", Amount: 90.00}

	for _, b := range []*Bid{b1, b2, b3} {
		if err := store.AddBid(ctx, b); err != nil {
			t.Fatalf("AddBid: %v", err)
		}
	}

	best, err := store.GetBestBid(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetBestBid: %v", err)
	}
	if best.Amount != 95.00 {
		t.Errorf("expected best bid 95.00, got %f", best.Amount)
	}
	if best.BidderID != "buyer2" {
		t.Errorf("expected best bidder buyer2, got %s", best.BidderID)
	}
}

// ─── Engine Tests ───

func setupTestEngine(t *testing.T) (*Engine, *Store) {
	t.Helper()
	store := setupTestStore(t)
	logger := testLogger(t)
	eng := NewEngine(store, logger)
	return eng, store
}

func TestEngine_ListItem(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "My Widget", "A great widget", 100.00, 80.00, "fixed", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	if listing.Title != "My Widget" {
		t.Errorf("expected title 'My Widget', got %s", listing.Title)
	}
	if listing.Status != "active" {
		t.Errorf("expected status active, got %s", listing.Status)
	}

	// Verify it persisted
	got, err := store.GetListing(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if got.Title != "My Widget" {
		t.Errorf("expected persisted title 'My Widget', got %s", got.Title)
	}
}

func TestEngine_ListItem_DefaultExpiry(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// No expiry = 48h for auction
	auction, err := eng.ListItem(ctx, "seller1", "Auction", "", 200, 150, "auction", 0)
	if err != nil {
		t.Fatalf("ListItem(auction): %v", err)
	}
	expiry := auction.ExpiresAt.Sub(auction.CreatedAt)
	if expiry.Hours() < 47 || expiry.Hours() > 49 {
		t.Errorf("expected auction expiry ~48h, got %v", expiry)
	}

	// No expiry = 168h for fixed
	fixed, err := eng.ListItem(ctx, "seller1", "Fixed", "", 50, 40, "fixed", 0)
	if err != nil {
		t.Fatalf("ListItem(fixed): %v", err)
	}
	expiry = fixed.ExpiresAt.Sub(fixed.CreatedAt)
	if expiry.Hours() < 167 || expiry.Hours() > 169 {
		t.Errorf("expected fixed expiry ~168h, got %v", expiry)
	}
}

func TestEngine_PlaceBid_FixedAutoAccept(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Fixed Item", "", 100.00, 80.00, "fixed", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	// Bid below desired price — should NOT auto-accept
	lowBid, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 85.00, "Can you do 85?")
	if err != nil {
		t.Fatalf("PlaceBid (low): %v", err)
	}
	if lowBid.Amount != 85.00 {
		t.Errorf("expected bid amount 85.00, got %f", lowBid.Amount)
	}

	// Listing should still be negotiating (not sold)
	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "negotiating" {
		t.Errorf("expected status negotiating, got %s", got.Status)
	}

	// Bid at or above desired price — should auto-accept
	highBid, err := eng.PlaceBid(ctx, listing.ID, "buyer2", 100.00, "I'll pay full price")
	if err != nil {
		t.Fatalf("PlaceBid (high): %v", err)
	}
	if highBid.Amount != 100.00 {
		t.Errorf("expected bid amount 100.00, got %f", highBid.Amount)
	}

	// Listing should be sold now
	got, _ = store.GetListing(ctx, listing.ID)
	if got.Status != "sold" {
		t.Errorf("expected status sold, got %s", got.Status)
	}
}

func TestEngine_PlaceBid_RejectedStatus(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Test", "", 100, 80, "fixed", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	// Mark as sold
	if err := eng.Store().UpdateListingStatus(ctx, listing.ID, "sold"); err != nil {
		t.Fatalf("UpdateListingStatus: %v", err)
	}

	// Try to place a bid on a sold listing
	_, err = eng.PlaceBid(ctx, listing.ID, "buyer1", 110, "Too late")
	if err == nil {
		t.Fatal("expected error for sold listing")
	}
}

func TestEngine_AcceptBid(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Acceptable Item", "", 100, 80, "haggling", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	bid, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 85, "How about 85?")
	if err != nil {
		t.Fatalf("PlaceBid: %v", err)
	}

	result, err := eng.AcceptBid(ctx, listing.ID, bid.ID)
	if err != nil {
		t.Fatalf("AcceptBid: %v", err)
	}

	if result.Status != "sold" {
		t.Errorf("expected status sold, got %s", result.Status)
	}
	if result.FinalPrice != 85.00 {
		t.Errorf("expected final price 85.00, got %f", result.FinalPrice)
	}
	if result.WinningBid.ID != bid.ID {
		t.Errorf("expected winning bid ID %s, got %s", bid.ID, result.WinningBid.ID)
	}

	// Listing should be sold
	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "sold" {
		t.Errorf("expected status sold, got %s", got.Status)
	}
}

func TestEngine_AcceptBid_AlreadySold(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Item", "", 100, 80, "fixed", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	bid, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 100, "Full price")
	if err != nil {
		t.Fatalf("PlaceBid: %v", err)
	}

	// Already auto-sold, try to accept again
	_, err = eng.AcceptBid(ctx, listing.ID, bid.ID)
	if err == nil {
		t.Fatal("expected error for already sold listing")
	}
}

func TestEngine_RejectBid(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Item", "", 100, 80, "haggling", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	bid, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 70, "Too low?")
	if err != nil {
		t.Fatalf("PlaceBid: %v", err)
	}

	err = eng.RejectBid(ctx, listing.ID, bid.ID, "Sorry, too low. How about 90?")
	if err != nil {
		t.Fatalf("RejectBid: %v", err)
	}
}

func TestEngine_RejectBid_NotFound(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Item", "", 100, 80, "haggling", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	err = eng.RejectBid(ctx, listing.ID, "nonexistent-bid", "Nope")
	if err == nil {
		t.Fatal("expected error for nonexistent bid")
	}
}

func TestEngine_CheckExpired_AuctionNoBids(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	// Create an auction that expired 1 hour ago
	past := time.Now().UTC().Add(-1 * time.Hour)
	listing := &Listing{
		UserID: "seller1", Title: "Expired Auction", DesiredPrice: 200, Strategy: "auction",
		Status: "active", CreatedAt: past.Add(-48 * time.Hour), ExpiresAt: past,
	}
	if err := store.CreateListing(ctx, listing); err != nil {
		t.Fatalf("CreateListing: %v", err)
	}

	if err := eng.CheckExpired(ctx, listing.ID); err != nil {
		t.Fatalf("CheckExpired: %v", err)
	}

	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "withdrawn" {
		t.Errorf("expected status withdrawn for expired auction with no bids, got %s", got.Status)
	}
}

func TestEngine_CheckExpired_AuctionWithBids(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	// Create an auction that expired 1 hour ago
	past := time.Now().UTC().Add(-1 * time.Hour)
	listing := &Listing{
		UserID: "seller1", Title: "Expired Auction", DesiredPrice: 200, Strategy: "auction",
		Status: "active", CreatedAt: past.Add(-48 * time.Hour), ExpiresAt: past,
	}
	if err := store.CreateListing(ctx, listing); err != nil {
		t.Fatalf("CreateListing: %v", err)
	}

	// Add some bids
	b1 := &Bid{ListingID: listing.ID, BidderID: "buyer1", Amount: 150}
	b2 := &Bid{ListingID: listing.ID, BidderID: "buyer2", Amount: 180}
	for _, b := range []*Bid{b1, b2} {
		if err := store.AddBid(ctx, b); err != nil {
			t.Fatalf("AddBid: %v", err)
		}
	}

	if err := eng.CheckExpired(ctx, listing.ID); err != nil {
		t.Fatalf("CheckExpired: %v", err)
	}

	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "sold" {
		t.Errorf("expected status sold for expired auction with bids, got %s", got.Status)
	}
}

func TestEngine_CheckExpired_NotYetExpired(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Future", "", 100, 80, "fixed", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	if err := eng.CheckExpired(ctx, listing.ID); err != nil {
		t.Fatalf("CheckExpired: %v", err)
	}

	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "active" {
		t.Errorf("expected status still active, got %s", got.Status)
	}
}

func TestGenerateCounter(t *testing.T) {
	counter := GenerateCounter(100.00)
	if counter != 105.00 {
		t.Errorf("expected counter 105.00, got %f", counter)
	}

	counter = GenerateCounter(50.00)
	if counter != 52.50 {
		t.Errorf("expected counter 52.50, got %f", counter)
	}
}

func TestEngine_PlaceBid_HagglingCounter(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	// Haggling listing with desired price of 100
	listing, err := eng.ListItem(ctx, "seller1", "Haggle Item", "", 100.00, 70.00, "haggling", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	// Place first bid below desired — should succeed
	bid1, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 80.00, "Can you do 80?")
	if err != nil {
		t.Fatalf("PlaceBid1: %v", err)
	}
	if bid1.Amount != 80.00 {
		t.Errorf("expected bid amount 80.00, got %f", bid1.Amount)
	}

	// Seller should be able to accept it
	result, err := eng.AcceptBid(ctx, listing.ID, bid1.ID)
	if err != nil {
		t.Fatalf("AcceptBid: %v", err)
	}
	if result.Status != "sold" {
		t.Errorf("expected status sold, got %s", result.Status)
	}

	// Reset: check that fixed strategy still works correctly
	got, _ := store.GetListing(ctx, listing.ID)
	if got.Status != "sold" {
		t.Errorf("expected status sold, got %s", got.Status)
	}
}

func TestEngine_PlaceBid_HaggleLimitReached(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListItem(ctx, "seller1", "Haggle Limit", "", 100.00, 70.00, "haggling", 72)
	if err != nil {
		t.Fatalf("ListItem: %v", err)
	}

	// Place 3 bids from same buyer below desired price
	// Each should succeed (auto-counter is just a mechanism, bids are placed)
	for i := 0; i < 3; i++ {
		_, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 75.00, "Still interested")
		if err != nil {
			t.Fatalf("PlaceBid round %d: %v", i, err)
		}
	}

	// 4th bid — should still be accepted (freeze means no auto-counter, but bid goes through)
	bid4, err := eng.PlaceBid(ctx, listing.ID, "buyer1", 75.00, "Last try")
	if err != nil {
		t.Fatalf("PlaceBid round 4: %v", err)
	}
	if bid4.Amount != 75.00 {
		t.Errorf("expected bid amount 75.00, got %f", bid4.Amount)
	}

	// Verify all 4 bids exist
	bids, err := store.GetBids(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetBids: %v", err)
	}
	if len(bids) != 4 {
		t.Errorf("expected 4 bids, got %d", len(bids))
	}
}
