package calendar

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

// setupTestStore creates an in-memory SQLite store and seeds pricing data.
func setupTestStore(t *testing.T) (*Store, *pricing.Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:calendar_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("calendar NewStore: %v", err)
	}

	seedPricing(t, pstore)
	return store, pstore
}

// setupTestEngine creates a fully initialized Engine for integration tests.
func setupTestEngine(t *testing.T) (*Engine, *Store, *pricing.Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:calendar_engine_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("calendar NewStore: %v", err)
	}

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history NewStore: %v", err)
	}

	seedPricing(t, pstore)
	seedAdditionalPricing(t, pstore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	negEng := negotiation.NewEngine(pstore)
	eng := NewEngine(store, negEng, hstore, pstore, logger)

	return eng, store, pstore
}

func seedPricing(t *testing.T, pstore *pricing.Store) {
	t.Helper()
	ctx := context.Background()

	_, err := pstore.DB().ExecContext(ctx,
		"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", "TestVendor", "Testing")
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	var vid int64
	err = pstore.DB().QueryRowContext(ctx,
		"SELECT id FROM vendors WHERE name = ?", "TestVendor").Scan(&vid)
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

func seedAdditionalPricing(t *testing.T, pstore *pricing.Store) {
	t.Helper()
	ctx := context.Background()

	_, err := pstore.DB().ExecContext(ctx,
		"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", "Slack", "Communication")
	if err != nil {
		t.Fatalf("insert vendor Slack: %v", err)
	}
	var vid int64
	err = pstore.DB().QueryRowContext(ctx,
		"SELECT id FROM vendors WHERE name = ?", "Slack").Scan(&vid)
	if err != nil {
		t.Fatalf("get Slack vendor id: %v", err)
	}
	_, err = pstore.DB().ExecContext(ctx, `
		INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
	`, vid, "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month")
	if err != nil {
		t.Fatalf("insert Slack pricing: %v", err)
	}
}

func createTestContract(t *testing.T, store *Store, ctx context.Context) *Contract {
	t.Helper()
	c := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user1",
		Seats:        10,
		CurrentPrice: 100.00,
		RenewalDate:  time.Now().UTC().Add(30 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	return c
}

// ─── Store Tests ───

func TestCreateAndGetContract(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	c := createTestContract(t, store, ctx)

	got, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Vendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", got.Vendor)
	}
	if got.SKU != "TestSKU" {
		t.Errorf("expected SKU TestSKU, got %s", got.SKU)
	}
	if got.Seats != 10 {
		t.Errorf("expected seats 10, got %d", got.Seats)
	}
	if got.CurrentPrice != 100.00 {
		t.Errorf("expected price 100.00, got %f", got.CurrentPrice)
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
	if got.RenewalDate.IsZero() {
		t.Error("expected non-zero RenewalDate")
	}
}

func TestGetContract_NotFound(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetContract(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent contract")
	}
}

func TestListContracts_All(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	c1 := createTestContract(t, store, ctx)

	c2 := &Contract{
		Vendor:       "OtherVendor",
		SKU:          "OtherSKU",
		UserID:       "user2",
		Seats:        5,
		CurrentPrice: 50.00,
		RenewalDate:  time.Now().UTC().Add(60 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c2); err != nil {
		t.Fatalf("CreateContract c2: %v", err)
	}

	contracts, err := store.ListContracts(ctx, ContractFilter{})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(contracts))
	}

	ids := map[string]bool{c1.ID: true, c2.ID: true}
	for _, c := range contracts {
		if !ids[c.ID] {
			t.Errorf("unexpected contract id: %s", c.ID)
		}
		delete(ids, c.ID)
	}
}

func TestListContracts_FilteredByVendor(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	_ = createTestContract(t, store, ctx)

	c2 := &Contract{
		Vendor:       "OtherVendor",
		SKU:          "OtherSKU",
		UserID:       "user2",
		Seats:        5,
		CurrentPrice: 50.00,
		RenewalDate:  time.Now().UTC().Add(60 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c2); err != nil {
		t.Fatalf("CreateContract c2: %v", err)
	}

	contracts, err := store.ListContracts(ctx, ContractFilter{Vendor: "TestVendor"})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].Vendor != "TestVendor" {
		t.Errorf("expected TestVendor, got %s", contracts[0].Vendor)
	}
}

func TestListContracts_FilteredByStatus(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	c1 := createTestContract(t, store, ctx)

	c2 := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user2",
		Seats:        5,
		CurrentPrice: 50.00,
		RenewalDate:  time.Now().UTC().Add(60 * 24 * time.Hour),
		Status:       "renewed",
	}
	if err := store.CreateContract(ctx, c2); err != nil {
		t.Fatalf("CreateContract c2: %v", err)
	}

	// Filter by 'active'
	contracts, err := store.ListContracts(ctx, ContractFilter{Status: "active"})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 active contract, got %d", len(contracts))
	}
	if contracts[0].ID != c1.ID {
		t.Errorf("expected c1 id, got %s", contracts[0].ID)
	}

	// Filter by 'renewed'
	contracts, err = store.ListContracts(ctx, ContractFilter{Status: "renewed"})
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 renewed contract, got %d", len(contracts))
	}
	if contracts[0].ID != c2.ID {
		t.Errorf("expected c2 id, got %s", contracts[0].ID)
	}
}

func TestUpdateContract(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	c := createTestContract(t, store, ctx)

	c.Status = "renewed"
	c.CurrentPrice = 85.00
	c.LastNegotiatedPrice = 85.00
	c.Seats = 15

	if err := store.UpdateContract(ctx, c); err != nil {
		t.Fatalf("UpdateContract: %v", err)
	}

	got, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract after update: %v", err)
	}
	if got.Status != "renewed" {
		t.Errorf("expected status renewed, got %s", got.Status)
	}
	if got.CurrentPrice != 85.00 {
		t.Errorf("expected price 85.00, got %f", got.CurrentPrice)
	}
	if got.LastNegotiatedPrice != 85.00 {
		t.Errorf("expected last_negotiated_price 85.00, got %f", got.LastNegotiatedPrice)
	}
	if got.Seats != 15 {
		t.Errorf("expected seats 15, got %d", got.Seats)
	}
}

func TestGetContractsExpiringSoon(t *testing.T) {
	store, _ := setupTestStore(t)
	ctx := context.Background()

	// Contract renewing in 10 days (should appear in 30-day lookahead)
	c1 := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user1",
		Seats:        10,
		CurrentPrice: 100.00,
		RenewalDate:  time.Now().UTC().Add(10 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c1); err != nil {
		t.Fatalf("CreateContract c1: %v", err)
	}

	// Contract renewing in 60 days (should appear in 90-day but not 30-day lookahead)
	c2 := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user2",
		Seats:        5,
		CurrentPrice: 50.00,
		RenewalDate:  time.Now().UTC().Add(60 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c2); err != nil {
		t.Fatalf("CreateContract c2: %v", err)
	}

	// Contract already expired (should not appear)
	c3 := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user3",
		Seats:        3,
		CurrentPrice: 30.00,
		RenewalDate:  time.Now().UTC().Add(-5 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c3); err != nil {
		t.Fatalf("CreateContract c3: %v", err)
	}

	// Contract with non-active status renewing soon (should not appear)
	c4 := &Contract{
		Vendor:       "TestVendor",
		SKU:          "TestSKU",
		UserID:       "user4",
		Seats:        3,
		CurrentPrice: 30.00,
		RenewalDate:  time.Now().UTC().Add(10 * 24 * time.Hour),
		Status:       "renewed",
	}
	if err := store.CreateContract(ctx, c4); err != nil {
		t.Fatalf("CreateContract c4: %v", err)
	}

	// 30-day lookahead should return only c1
	contracts, err := store.GetContractsExpiringSoon(ctx, 30)
	if err != nil {
		t.Fatalf("GetContractsExpiringSoon(30): %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract in 30-day lookahead, got %d", len(contracts))
	}
	if contracts[0].ID != c1.ID {
		t.Errorf("expected c1, got %s", contracts[0].ID)
	}

	// 90-day lookahead should return c1 and c2 (not c3 - expired, not c4 - renewed)
	contracts, err = store.GetContractsExpiringSoon(ctx, 90)
	if err != nil {
		t.Fatalf("GetContractsExpiringSoon(90): %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts in 90-day lookahead, got %d", len(contracts))
	}
}

func TestTriggerNegotiation(t *testing.T) {
	eng, store, _ := setupTestEngine(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create a contract with a vendor/SKU that has pricing data
	c := &Contract{
		Vendor:       "Slack",
		SKU:          "Pro",
		UserID:       "user1",
		Seats:        50,
		CurrentPrice: 8.75,
		RenewalDate:  now.Add(15 * 24 * time.Hour),
		Status:       "active",
	}
	if err := store.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}

	session, err := eng.TriggerNegotiation(ctx, c.ID)
	if err != nil {
		t.Fatalf("TriggerNegotiation: %v", err)
	}

	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if session.Status != "completed" {
		t.Errorf("expected session status completed, got %s", session.Status)
	}
	if session.Outcome != "accepted" {
		t.Errorf("expected session outcome accepted, got %s", session.Outcome)
	}

	// Verify contract was updated
	updated, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract after trigger: %v", err)
	}
	if updated.Status != "renewed" {
		t.Errorf("expected contract status renewed, got %s", updated.Status)
	}
	if updated.LastNegotiatedPrice <= 0 {
		t.Errorf("expected last_negotiated_price > 0, got %f", updated.LastNegotiatedPrice)
	}
}

func TestTriggerNegotiation_NonActive(t *testing.T) {
	eng, store, _ := setupTestEngine(t)
	ctx := context.Background()

	now := time.Now().UTC()

	c := &Contract{
		Vendor:       "Slack",
		SKU:          "Pro",
		UserID:       "user1",
		Seats:        50,
		CurrentPrice: 8.75,
		RenewalDate:  now.Add(15 * 24 * time.Hour),
		Status:       "renewed",
	}
	if err := store.CreateContract(ctx, c); err != nil {
		t.Fatalf("CreateContract: %v", err)
	}

	_, err := eng.TriggerNegotiation(ctx, c.ID)
	if err == nil {
		t.Fatal("expected error for non-active contract")
	}
}

func TestTriggerNegotiation_NotFound(t *testing.T) {
	eng, _, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.TriggerNegotiation(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent contract")
	}
}
