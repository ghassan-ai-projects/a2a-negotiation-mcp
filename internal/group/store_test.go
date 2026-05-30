package group

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

// setupTestStore creates an in-memory SQLite store and seeds pricing data.
func setupTestStore(t *testing.T) (*Store, *pricing.Store) {
	t.Helper()

	// Use a unique in-memory database to avoid cross-test contamination
	db, err := sql.Open("sqlite", "file:group_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	gstore, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("group NewStore: %v", err)
	}

	seedPricing(t, pstore)
	return gstore, pstore
}

func seedPricing(t *testing.T, pstore *pricing.Store) {
	t.Helper()
	ctx := context.Background()

	_, err := pstore.DB().ExecContext(ctx, "INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", "TestVendor", "Testing")
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	var vid int64
	err = pstore.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", "TestVendor").Scan(&vid)
	if err != nil {
		t.Fatalf("get vendor id: %v", err)
	}
	_, err = pstore.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
	`, vid, "TestSKU", "Test product", 100.00, 70.00, 95.00, 20, "per_seat_month")
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}
}

func createTestGroup(t *testing.T, gstore *Store, ctx context.Context) *BuyingGroup {
	t.Helper()
	g := &BuyingGroup{
		TargetVendor: "TestVendor",
		TargetSKU:    "TestSKU",
		MinMembers:   2,
		Status:       "forming",
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(72 * time.Hour),
	}
	if err := gstore.CreateGroup(ctx, g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	return g
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ─── Store Tests ───

func TestCreateGroup(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	g := createTestGroup(t, gstore, ctx)

	got, err := gstore.GetGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.TargetVendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", got.TargetVendor)
	}
	if got.TargetSKU != "TestSKU" {
		t.Errorf("expected SKU TestSKU, got %s", got.TargetSKU)
	}
	if got.MinMembers != 2 {
		t.Errorf("expected MinMembers 2, got %d", got.MinMembers)
	}
	if got.Status != "forming" {
		t.Errorf("expected status forming, got %s", got.Status)
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

func TestJoinGroup(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	g := createTestGroup(t, gstore, ctx)

	m := &GroupMember{
		GroupID:  g.ID,
		UserID:   "user1",
		Quantity: 5,
		MaxPrice: 90.00,
	}
	if err := gstore.JoinGroup(ctx, m); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	members, err := gstore.GetMembers(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].UserID != "user1" {
		t.Errorf("expected user_id user1, got %s", members[0].UserID)
	}
	if members[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", members[0].Quantity)
	}
	if members[0].MaxPrice != 90.00 {
		t.Errorf("expected MaxPrice 90.00, got %f", members[0].MaxPrice)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	_, err := gstore.GetGroup(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
}

func TestJoinGroup_DuplicateError(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	g := createTestGroup(t, gstore, ctx)

	m1 := &GroupMember{
		GroupID:  g.ID,
		UserID:   "user1",
		Quantity: 5,
	}
	if err := gstore.JoinGroup(ctx, m1); err != nil {
		t.Fatalf("first JoinGroup: %v", err)
	}

	// Second member should join successfully
	m2 := &GroupMember{
		GroupID:  g.ID,
		UserID:   "user2",
		Quantity: 3,
	}
	if err := gstore.JoinGroup(ctx, m2); err != nil {
		t.Fatalf("second JoinGroup: %v", err)
	}

	count, err := gstore.GetMemberCount(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetMemberCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 members, got %d", count)
	}
}

func TestJoinGroup_ExpiredGroupRejection(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	g := createTestGroup(t, gstore, ctx)

	if err := gstore.UpdateGroupStatus(ctx, g.ID, "completed"); err != nil {
		t.Fatalf("UpdateGroupStatus: %v", err)
	}

	m := &GroupMember{
		GroupID:  g.ID,
		UserID:   "user1",
		Quantity: 5,
	}
	err := gstore.JoinGroup(ctx, m)
	if err == nil {
		t.Fatal("expected error joining completed group")
	}
}

func TestGetActiveGroups(t *testing.T) {
	gstore, _ := setupTestStore(t)
	ctx := context.Background()

	g1 := createTestGroup(t, gstore, ctx)

	g2 := &BuyingGroup{
		TargetVendor: "OtherVendor",
		TargetSKU:    "OtherSKU",
		MinMembers:   1,
		Status:       "forming",
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(48 * time.Hour),
	}
	if err := gstore.CreateGroup(ctx, g2); err != nil {
		t.Fatalf("CreateGroup g2: %v", err)
	}

	if err := gstore.UpdateGroupStatus(ctx, g1.ID, "completed"); err != nil {
		t.Fatalf("UpdateGroupStatus: %v", err)
	}

	active, err := gstore.GetActiveGroups(ctx)
	if err != nil {
		t.Fatalf("GetActiveGroups: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active group, got %d", len(active))
	}
	if active[0].ID != g2.ID {
		t.Errorf("expected active group %s, got %s", g2.ID, active[0].ID)
	}
}

// ─── Engine Tests ───

func TestComputeOffer_QuantityScaling(t *testing.T) {
		gstore, pstore := setupTestStore(t)
	ctx := context.Background()
	logger := testLogger(t)
	eng := NewEngine(gstore, pstore, logger)

	g := createTestGroup(t, gstore, ctx)

	m1 := &GroupMember{GroupID: g.ID, UserID: "user1", Quantity: 1}
	if err := gstore.JoinGroup(ctx, m1); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	offer, err := eng.ComputeOffer(ctx, g.ID)
	if err != nil {
		t.Fatalf("ComputeOffer: %v", err)
	}

	if offer.DiscountPct != 20.0 {
		t.Errorf("expected discount 20%%, got %.2f%%", offer.DiscountPct)
	}
	if offer.OfferPrice != 80.00 {
		t.Errorf("expected offer 80.00, got %.2f", offer.OfferPrice)
	}
	if offer.MemberCount != 1 {
		t.Errorf("expected member count 1, got %d", offer.MemberCount)
	}
	if offer.TotalDemand != 1 {
		t.Errorf("expected total demand 1, got %d", offer.TotalDemand)
	}
}

func TestComputeOffer_QuantityScaling_Tier2(t *testing.T) {
		gstore, pstore := setupTestStore(t)
	ctx := context.Background()
	logger := testLogger(t)
	eng := NewEngine(gstore, pstore, logger)

	g := createTestGroup(t, gstore, ctx)

	m1 := &GroupMember{GroupID: g.ID, UserID: "user1", Quantity: 25}
	if err := gstore.JoinGroup(ctx, m1); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	offer, err := eng.ComputeOffer(ctx, g.ID)
	if err != nil {
		t.Fatalf("ComputeOffer: %v", err)
	}

	expectedDiscount := 20.0 * 1.2 // 24%
	if math.Abs(offer.DiscountPct-expectedDiscount) > 0.01 {
		t.Errorf("expected discount ~%.2f%%, got %.2f%%", expectedDiscount, offer.DiscountPct)
	}
}

func TestComputeOffer_MemberScaling(t *testing.T) {
		gstore, pstore := setupTestStore(t)
	ctx := context.Background()
	logger := testLogger(t)
	eng := NewEngine(gstore, pstore, logger)

	g := createTestGroup(t, gstore, ctx)

	for i := 1; i <= 5; i++ {
		m := &GroupMember{GroupID: g.ID, UserID: fmt.Sprintf("user%d", i), Quantity: 1}
		if err := gstore.JoinGroup(ctx, m); err != nil {
			t.Fatalf("JoinGroup user%d: %v", i, err)
		}
	}

	offer, err := eng.ComputeOffer(ctx, g.ID)
	if err != nil {
		t.Fatalf("ComputeOffer: %v", err)
	}

	expectedDiscount := 20.0 * 1.25 // 25%
	if math.Abs(offer.DiscountPct-expectedDiscount) > 0.01 {
		t.Errorf("expected discount ~%.2f%%, got %.2f%%", expectedDiscount, offer.DiscountPct)
	}
	if offer.MemberCount != 5 {
		t.Errorf("expected 5 members, got %d", offer.MemberCount)
	}
	if offer.TotalDemand != 5 {
		t.Errorf("expected total demand 5, got %d", offer.TotalDemand)
	}
}

func TestComputeOffer_CapAt50Percent(t *testing.T) {
	_, pstore := setupTestStore(t)
	ctx := context.Background()

	_, err := pstore.DB().ExecContext(ctx, "INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", "DiscountVendor", "Testing")
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	var vid int64
	err = pstore.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", "DiscountVendor").Scan(&vid)
	if err != nil {
		t.Fatalf("get vendor id: %v", err)
	}
	_, err = pstore.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
	`, vid, "HighDiscount", "High discount product", 100.00, 30.00, 90.00, 45, "per_seat_month")
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	logger := testLogger(t)

	g := &BuyingGroup{
		TargetVendor: "DiscountVendor",
		TargetSKU:    "HighDiscount",
		MinMembers:   1,
		Status:       "forming",
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(72 * time.Hour),
	}
	gstore2, _ := NewStore(pstore.DB())
	_ = gstore2.CreateGroup(ctx, g)

	for i := 1; i <= 10; i++ {
		m := &GroupMember{GroupID: g.ID, UserID: fmt.Sprintf("user%d", i), Quantity: 20}
		if err := gstore2.JoinGroup(ctx, m); err != nil {
			t.Fatalf("JoinGroup user%d: %v", i, err)
		}
	}

	eng := NewEngine(gstore2, pstore, logger)
	offer, err := eng.ComputeOffer(ctx, g.ID)
	if err != nil {
		t.Fatalf("ComputeOffer: %v", err)
	}

	if offer.DiscountPct > 50.0 {
		t.Errorf("expected discount capped at 50%%, got %.2f%%", offer.DiscountPct)
	}
	if math.Abs(offer.DiscountPct-50.0) > 0.01 {
		t.Errorf("expected discount exactly 50%% (capped), got %.2f%%", offer.DiscountPct)
	}
	if math.Abs(offer.OfferPrice-50.00) > 0.01 {
		t.Errorf("expected offer price 50.00, got %.2f", offer.OfferPrice)
	}
}

func TestComputeOffer_EmptyGroupError(t *testing.T) {
		gstore, pstore := setupTestStore(t)
	ctx := context.Background()
	logger := testLogger(t)
	eng := NewEngine(gstore, pstore, logger)

	g := createTestGroup(t, gstore, ctx)

	_, err := eng.ComputeOffer(ctx, g.ID)
	if err == nil {
		t.Fatal("expected error for empty group")
	}
}
