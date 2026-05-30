package marketplace

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", "file:mp_test_"+t.Name()+"?mode=memory&cache=private")
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
		Vendor:    "Slack",
		SKU:       "pro-seat",
		Seats:     50,
		OrigPrice: 15.00,
		AskPrice:  10.00,
		MinPrice:  7.00,
		Status:    "active",
		SellerID:  "seller1",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}
	if err := store.CreateListing(ctx, l); err != nil {
		t.Fatalf("CreateListing: %v", err)
	}
	return l
}

// ─── Store Tests ───

func TestCreateListing_Marketplace(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	got, err := store.GetListing(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if got.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", got.Vendor)
	}
	if got.SKU != "pro-seat" {
		t.Errorf("expected SKU pro-seat, got %s", got.SKU)
	}
	if got.Seats != 50 {
		t.Errorf("expected seats 50, got %d", got.Seats)
	}
	if got.OrigPrice != 15.00 {
		t.Errorf("expected orig_price 15.00, got %f", got.OrigPrice)
	}
	if got.AskPrice != 10.00 {
		t.Errorf("expected ask_price 10.00, got %f", got.AskPrice)
	}
	if got.MinPrice != 7.00 {
		t.Errorf("expected min_price 7.00, got %f", got.MinPrice)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.SellerID != "seller1" {
		t.Errorf("expected seller_id seller1, got %s", got.SellerID)
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

func TestGetListing_NotFound_Marketplace(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetListing(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent listing")
	}
}

func TestSearchListings(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l1 := createTestListing(t, store, ctx)

	// Create a second listing
	l2 := &Listing{
		Vendor:    "GitHub",
		SKU:       "team-seat",
		Seats:     20,
		OrigPrice: 10.00,
		AskPrice:  8.00,
		MinPrice:  5.00,
		Status:    "active",
		SellerID:  "seller2",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}
	if err := store.CreateListing(ctx, l2); err != nil {
		t.Fatalf("CreateListing l2: %v", err)
	}

	// Search by vendor
	results, err := store.SearchListings(ctx, "Slack", "", 0)
	if err != nil {
		t.Fatalf("SearchListings: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 Slack listing, got %d", len(results))
	}
	if results[0].ID != l1.ID {
		t.Errorf("expected listing %s, got %s", l1.ID, results[0].ID)
	}

	// Search by vendor and SKU
	results, err = store.SearchListings(ctx, "GitHub", "team-seat", 0)
	if err != nil {
		t.Fatalf("SearchListings: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 GitHub listing, got %d", len(results))
	}

	// Search all
	results, err = store.SearchListings(ctx, "", "", 0)
	if err != nil {
		t.Fatalf("SearchListings all: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 total listings, got %d", len(results))
	}

	// Verifies ordering by ask_price ASC
	if results[0].AskPrice > results[1].AskPrice {
		t.Errorf("expected listings sorted by ask_price ASC, got %f > %f", results[0].AskPrice, results[1].AskPrice)
	}
}

func TestSearchListings_MaxSeats(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	createTestListing(t, store, ctx) // 50 seats

	l2 := &Listing{
		Vendor:    "Slack",
		SKU:       "basic-seat",
		Seats:     5,
		OrigPrice: 15.00,
		AskPrice:  12.00,
		MinPrice:  8.00,
		Status:    "active",
		SellerID:  "seller1",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}
	if err := store.CreateListing(ctx, l2); err != nil {
		t.Fatalf("CreateListing l2: %v", err)
	}

	results, err := store.SearchListings(ctx, "Slack", "", 10)
	if err != nil {
		t.Fatalf("SearchListings: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 listing with <= 10 seats, got %d", len(results))
	}
	if results[0].Seats != 5 {
		t.Errorf("expected 5 seats, got %d", results[0].Seats)
	}
}

func TestAddOffer(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	o := &Offer{
		ListingID: l.ID,
		BuyerID:   "buyer1",
		Seats:     10,
		MaxPrice:  9.00,
		Status:    "pending",
	}
	if err := store.AddOffer(ctx, o); err != nil {
		t.Fatalf("AddOffer: %v", err)
	}

	offers, err := store.GetOffers(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetOffers: %v", err)
	}
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}
	if offers[0].BuyerID != "buyer1" {
		t.Errorf("expected buyer_id buyer1, got %s", offers[0].BuyerID)
	}
	if offers[0].MaxPrice != 9.00 {
		t.Errorf("expected max_price 9.00, got %f", offers[0].MaxPrice)
	}
	if offers[0].Seats != 10 {
		t.Errorf("expected seats 10, got %d", offers[0].Seats)
	}
}

func TestAddOffer_MultipleOffers(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	o1 := &Offer{ListingID: l.ID, BuyerID: "buyer1", Seats: 5, MaxPrice: 8.00}
	o2 := &Offer{ListingID: l.ID, BuyerID: "buyer2", Seats: 10, MaxPrice: 9.50}
	o3 := &Offer{ListingID: l.ID, BuyerID: "buyer3", Seats: 3, MaxPrice: 7.00}

	for _, o := range []*Offer{o1, o2, o3} {
		if err := store.AddOffer(ctx, o); err != nil {
			t.Fatalf("AddOffer: %v", err)
		}
	}

	offers, err := store.GetOffers(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetOffers: %v", err)
	}
	if len(offers) != 3 {
		t.Fatalf("expected 3 offers, got %d", len(offers))
	}
	// Should be ordered by max_price DESC
	if offers[0].MaxPrice != 9.50 {
		t.Errorf("expected first offer 9.50 (highest), got %f", offers[0].MaxPrice)
	}
	if offers[1].MaxPrice != 8.00 {
		t.Errorf("expected second offer 8.00, got %f", offers[1].MaxPrice)
	}
	if offers[2].MaxPrice != 7.00 {
		t.Errorf("expected third offer 7.00, got %f", offers[2].MaxPrice)
	}
}

func TestUpdateListingStatus_Marketplace(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)

	if err := store.UpdateListingStatus(ctx, l.ID, "completed"); err != nil {
		t.Fatalf("UpdateListingStatus: %v", err)
	}

	got, err := store.GetListing(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("expected status completed, got %s", got.Status)
	}
}

func TestUpdateOfferStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l := createTestListing(t, store, ctx)
	o := &Offer{ListingID: l.ID, BuyerID: "buyer1", Seats: 5, MaxPrice: 8.00}
	if err := store.AddOffer(ctx, o); err != nil {
		t.Fatalf("AddOffer: %v", err)
	}

	if err := store.UpdateOfferStatus(ctx, o.ID, "accepted"); err != nil {
		t.Fatalf("UpdateOfferStatus: %v", err)
	}

	offers, err := store.GetOffers(ctx, l.ID)
	if err != nil {
		t.Fatalf("GetOffers: %v", err)
	}
	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}
	if offers[0].Status != "accepted" {
		t.Errorf("expected offer status accepted, got %s", offers[0].Status)
	}
}

func TestCreateTransaction(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	txn := &Transaction{
		ListingID:    "listing-1",
		Vendor:       "Slack",
		SKU:          "pro-seat",
		Seats:        10,
		PricePerSeat: 9.00,
		Total:        90.00,
		PlatformFee:  4.50,
		SellerID:     "seller1",
		BuyerID:      "buyer1",
		Status:       "completed",
	}
	if err := store.CreateTransaction(ctx, txn); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	txns, err := store.GetRecentTransactions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentTransactions: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
	if txns[0].Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", txns[0].Vendor)
	}
	if txns[0].Total != 90.00 {
		t.Errorf("expected total 90.00, got %f", txns[0].Total)
	}
	if txns[0].PlatformFee != 4.50 {
		t.Errorf("expected platform_fee 4.50, got %f", txns[0].PlatformFee)
	}
}

func TestGetActiveListings(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	l1 := createTestListing(t, store, ctx)

	// Completed listing
	l2 := &Listing{
		Vendor: "GitHub", SKU: "team-seat", Seats: 20,
		OrigPrice: 10.00, AskPrice: 8.00, MinPrice: 5.00,
		Status: "completed", SellerID: "seller2",
		CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(168 * time.Hour),
	}
	if err := store.CreateListing(ctx, l2); err != nil {
		t.Fatalf("CreateListing l2: %v", err)
	}

	active, err := store.GetActiveListings(ctx)
	if err != nil {
		t.Fatalf("GetActiveListings: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active listing, got %d", len(active))
	}
	if active[0].ID != l1.ID {
		t.Errorf("expected active listing %s, got %s", l1.ID, active[0].ID)
	}
}

func TestGetRecentTransactions_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	txns, err := store.GetRecentTransactions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentTransactions: %v", err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txns))
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

func TestEngine_ListSeats(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	if listing.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", listing.Vendor)
	}
	if listing.SKU != "pro-seat" {
		t.Errorf("expected SKU pro-seat, got %s", listing.SKU)
	}
	if listing.Seats != 50 {
		t.Errorf("expected seats 50, got %d", listing.Seats)
	}
	if listing.OrigPrice != 15.00 {
		t.Errorf("expected orig_price 15.00, got %f", listing.OrigPrice)
	}
	if listing.AskPrice != 10.00 {
		t.Errorf("expected ask_price 10.00, got %f", listing.AskPrice)
	}
	if listing.MinPrice != 7.00 {
		t.Errorf("expected min_price 7.00, got %f", listing.MinPrice)
	}
	if listing.Status != "active" {
		t.Errorf("expected status active, got %s", listing.Status)
	}

	// Verify persisted
	got, err := store.GetListing(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if got.Vendor != "Slack" {
		t.Errorf("expected persisted vendor Slack, got %s", got.Vendor)
	}
}

func TestEngine_ListSeats_AskPriceMustBeLessThanOrigPrice(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 15.00, 7.00, 72)
	if err == nil {
		t.Fatal("expected error when ask_price >= orig_price")
	}

	_, err = eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 20.00, 7.00, 72)
	if err == nil {
		t.Fatal("expected error when ask_price > orig_price")
	}
}

func TestEngine_ListSeats_InvalidSeats(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.ListSeats(ctx, "Slack", "pro-seat", 0, 15.00, 10.00, 7.00, 72)
	if err == nil {
		t.Fatal("expected error for zero seats")
	}
}

func TestEngine_Search(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}
	_, err = eng.ListSeats(ctx, "GitHub", "team-seat", 20, 10.00, 8.00, 5.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	results, err := eng.Search(ctx, "Slack", "", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 Slack listing, got %d", len(results))
	}
}

func TestEngine_MakeOffer_Pending(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	// Offer below ask price -> should remain pending
	offer, err := eng.MakeOffer(ctx, listing.ID, "buyer1", 10, 9.00)
	if err != nil {
		t.Fatalf("MakeOffer: %v", err)
	}
	if offer.Status != "pending" {
		t.Errorf("expected offer status pending, got %s", offer.Status)
	}
	if offer.MaxPrice != 9.00 {
		t.Errorf("expected max_price 9.00, got %f", offer.MaxPrice)
	}
}

func TestEngine_MakeOffer_AutoAccept(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	// Offer at or above ask price -> should auto-accept
	offer, err := eng.MakeOffer(ctx, listing.ID, "buyer1", 10, 10.00)
	if err != nil {
		t.Fatalf("MakeOffer: %v", err)
	}
	if offer.Status != "accepted" {
		t.Errorf("expected offer status accepted, got %s", offer.Status)
	}

	// Listing should be completed
	listing, err = eng.store.GetListing(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if listing.Status != "completed" {
		t.Errorf("expected listing status completed, got %s", listing.Status)
	}

	// Transaction should exist
	txns, err := eng.store.GetRecentTransactions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentTransactions: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}
	if txns[0].Total != 100.00 {
		t.Errorf("expected total 100.00, got %f", txns[0].Total)
	}
	if txns[0].PlatformFee != 5.00 {
		t.Errorf("expected platform_fee 5.00, got %f", txns[0].PlatformFee)
	}
	if txns[0].BuyerID != "buyer1" {
		t.Errorf("expected buyer_id buyer1, got %s", txns[0].BuyerID)
	}
}

func TestEngine_MakeOffer_ExceedsSeats(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	_, err = eng.MakeOffer(ctx, listing.ID, "buyer1", 100, 9.00)
	if err == nil {
		t.Fatal("expected error for exceeding available seats")
	}
}

func TestEngine_MakeOffer_ListingNotActive(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	// Mark as completed
	if err := store.UpdateListingStatus(ctx, listing.ID, "completed"); err != nil {
		t.Fatalf("UpdateListingStatus: %v", err)
	}

	_, err = eng.MakeOffer(ctx, listing.ID, "buyer1", 10, 10.00)
	if err == nil {
		t.Fatal("expected error for completed listing")
	}
}

func TestEngine_AcceptOffer(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	offer, err := eng.MakeOffer(ctx, listing.ID, "buyer1", 10, 9.00)
	if err != nil {
		t.Fatalf("MakeOffer: %v", err)
	}

	// Seller accepts the pending offer
	txn, err := eng.AcceptOffer(ctx, listing.ID, offer.ID)
	if err != nil {
		t.Fatalf("AcceptOffer: %v", err)
	}

	if txn.Total != 90.00 {
		t.Errorf("expected total 90.00 (9.00 * 10), got %f", txn.Total)
	}
	if txn.PlatformFee != 4.50 {
		t.Errorf("expected platform_fee 4.50 (5%% of 90), got %f", txn.PlatformFee)
	}
	if txn.Status != "completed" {
		t.Errorf("expected transaction status completed, got %s", txn.Status)
	}
	if txn.PricePerSeat != 9.00 {
		t.Errorf("expected price_per_seat 9.00, got %f", txn.PricePerSeat)
	}

	// Listing should be completed
	listing, err = eng.store.GetListing(ctx, listing.ID)
	if err != nil {
		t.Fatalf("GetListing: %v", err)
	}
	if listing.Status != "completed" {
		t.Errorf("expected listing status completed, got %s", listing.Status)
	}
}

func TestEngine_AcceptOffer_AlreadyCompleted(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	offer, err := eng.MakeOffer(ctx, listing.ID, "buyer1", 10, 10.00)
	if err != nil {
		t.Fatalf("MakeOffer: %v", err)
	}
	_ = offer // auto-accepted, listing is completed

	_, err = eng.AcceptOffer(ctx, listing.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for already completed listing")
	}
}

func TestEngine_AcceptOffer_OfferNotFound(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	listing, err := eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	_, err = eng.AcceptOffer(ctx, listing.ID, "nonexistent-offer")
	if err == nil {
		t.Fatal("expected error for nonexistent offer")
	}
}

func TestEngine_Overview(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// No data yet
	overview, err := eng.Overview(ctx)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}

	activeCount, _ := overview["active_listings_count"].(int)
	if activeCount != 0 {
		t.Errorf("expected 0 active listings, got %d", activeCount)
	}

	txnCount, _ := overview["transaction_count"].(int)
	if txnCount != 0 {
		t.Errorf("expected 0 transactions, got %d", txnCount)
	}

	// Add a listing and do an auto-accept to create a transaction
	_, err = eng.ListSeats(ctx, "Slack", "pro-seat", 50, 15.00, 10.00, 7.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	// Since the first listing is completed, we need to add another active one
	listing2, err := eng.ListSeats(ctx, "GitHub", "team-seat", 20, 10.00, 8.00, 5.00, 72)
	if err != nil {
		t.Fatalf("ListSeats: %v", err)
	}

	// Auto-accept the active listing
	_, err = eng.MakeOffer(ctx, listing2.ID, "buyer1", 10, 8.00)
	if err != nil {
		t.Fatalf("MakeOffer: %v", err)
	}

	overview, err = eng.Overview(ctx)
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}

	activeCount, _ = overview["active_listings_count"].(int)
	if activeCount != 1 {
		t.Errorf("expected 1 active listing, got %d", activeCount)
	}

	txnCount, _ = overview["transaction_count"].(int)
	if txnCount != 1 {
		t.Errorf("expected 1 transaction, got %d", txnCount)
	}
}
