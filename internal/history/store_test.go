package history

import (
	"context"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupHistoryTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestSaveAndGetSession(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	sess := &SessionRecord{
		ID: "test-session-1", Vendor: "Slack", SKU: "Pro",
		Strategy: "balanced", Budget: 1000, Status: "active",
		CurrentOffer: 7.00, ListPrice: 8.75, RoundsComplete: 0,
		Outcome: "", CreatedAt: now, UpdatedAt: now,
	}

	err := s.SaveSession(ctx, sess)
	if err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	got, err := s.GetSession(ctx, "test-session-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", got.Vendor)
	}
	if got.ID != "test-session-1" {
		t.Errorf("expected session ID test-session-1, got %s", got.ID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	_, err := s.GetSession(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestUpdateSession(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	sess := &SessionRecord{
		ID: "test-update", Vendor: "GitHub", SKU: "Team",
		Strategy: "aggressive", Status: "active",
		CurrentOffer: 3.20, ListPrice: 4.00, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	sess.Status = "completed"
	sess.Outcome = "accepted"
	sess.CurrentOffer = 2.80
	sess.UpdatedAt = time.Now().UTC()
	if err := s.UpdateSession(ctx, sess); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	got, err := s.GetSession(ctx, "test-update")
	if err != nil {
		t.Fatalf("GetSession after update: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("expected status completed, got %s", got.Status)
	}
	if got.Outcome != "accepted" {
		t.Errorf("expected outcome accepted, got %s", got.Outcome)
	}
	if got.CurrentOffer != 2.80 {
		t.Errorf("expected offer 2.80, got %f", got.CurrentOffer)
	}
}

func TestSaveAndGetRounds(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	sess := &SessionRecord{
		ID: "test-rounds", Vendor: "Salesforce", SKU: "Enterprise",
		Strategy: "conservative", Status: "active",
		CurrentOffer: 150.00, ListPrice: 165.00, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	rounds := []RoundRecord{
		{SessionID: "test-rounds", RoundNumber: 1, Offer: 148.50, DiscountPct: 0.10, Counterparty: "buyer", Note: "Initial offer", CreatedAt: now},
		{SessionID: "test-rounds", RoundNumber: 2, Offer: 145.00, DiscountPct: 0.12, Counterparty: "seller", Note: "Counter", CreatedAt: now},
	}
	if err := s.SaveRounds(ctx, rounds); err != nil {
		t.Fatalf("SaveRounds: %v", err)
	}

	got, err := s.GetRounds(ctx, "test-rounds")
	if err != nil {
		t.Fatalf("GetRounds: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rounds, got %d", len(got))
	}
	if got[0].RoundNumber != 1 {
		t.Errorf("expected round 1, got %d", got[0].RoundNumber)
	}
}

func TestSaveDealAndHistory(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	deal := &DealOutcome{
		Vendor: "Slack", SKU: "Pro", ListPrice: 8.75,
		FinalPrice: 7.00, DiscountPct: 0.20,
		Seats: 100, TermMonths: 12, Strategy: "balanced",
		SessionID: "deal-session-1", CreatedAt: now,
	}
	if err := s.SaveDealOutcome(ctx, deal); err != nil {
		t.Fatalf("SaveDealOutcome: %v", err)
	}

	// Save another deal
	deal2 := &DealOutcome{
		Vendor: "GitHub", SKU: "Enterprise", ListPrice: 21.00,
		FinalPrice: 16.80, DiscountPct: 0.20,
		Seats: 50, TermMonths: 24, Strategy: "aggressive",
		SessionID: "deal-session-2", CreatedAt: now,
	}
	if err := s.SaveDealOutcome(ctx, deal2); err != nil {
		t.Fatalf("SaveDealOutcome: %v", err)
	}

	// Test history
	summary, err := s.GetHistory(ctx, "", "all")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if summary.TotalDeals != 2 {
		t.Errorf("expected 2 deals, got %d", summary.TotalDeals)
	}
	if summary.TotalSavings <= 0 {
		t.Errorf("expected positive total savings, got %f", summary.TotalSavings)
	}

	// Filter by vendor
	slackSummary, err := s.GetHistory(ctx, "Slack", "all")
	if err != nil {
		t.Fatalf("GetHistory(Slack): %v", err)
	}
	if slackSummary.TotalDeals != 1 {
		t.Errorf("expected 1 Slack deal, got %d", slackSummary.TotalDeals)
	}
}

func TestGetSimilarDeals(t *testing.T) {
	s := setupHistoryTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		deal := &DealOutcome{
			Vendor: "Slack", SKU: "Pro", ListPrice: 8.75,
			FinalPrice: 7.00 - float64(i)*0.5, DiscountPct: 0.20 + float64(i)*0.05,
			Seats: 100, TermMonths: 12, Strategy: "balanced",
			SessionID: "sim-deal", CreatedAt: now,
		}
		if err := s.SaveDealOutcome(ctx, deal); err != nil {
			t.Fatalf("SaveDealOutcome: %v", err)
		}
	}

	deals, err := s.GetSimilarDeals(ctx, "Slack", 2)
	if err != nil {
		t.Fatalf("GetSimilarDeals: %v", err)
	}
	if len(deals) != 2 {
		t.Errorf("expected 2 deals, got %d", len(deals))
	}
}
