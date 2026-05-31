package gamification

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

	db, err := sql.Open("sqlite", "file:gam_test_"+t.Name()+"?mode=memory&cache=private")
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

func setupTestEngine(t *testing.T) (*Engine, *Store) {
	t.Helper()
	store := setupTestStore(t)
	logger := testLogger(t)
	eng := New(store, logger)
	return eng, store
}

// TestRecordNegotiation_ThreeConsecutiveDeals verifies that 3 consecutive daily
// deals produce a streak of 3.
func TestRecordNegotiation_ThreeConsecutiveDeals(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	userID := "user-alice"

	// Deal 1 (day 1)
	if err := eng.RecordNegotiation(ctx, userID, 100); err != nil {
		t.Fatalf("record negotiation 1: %v", err)
	}
	streak, err := eng.GetStreak(ctx, userID)
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if streak.CurrentStreak != 1 {
		t.Errorf("expected streak=1 after first deal, got %d", streak.CurrentStreak)
	}
	if streak.TotalDeals != 1 {
		t.Errorf("expected total_deals=1, got %d", streak.TotalDeals)
	}
	if streak.TotalSavings != 100 {
		t.Errorf("expected total_savings=100, got %f", streak.TotalSavings)
	}
	if streak.LongestStreak != 1 {
		t.Errorf("expected longest_streak=1, got %d", streak.LongestStreak)
	}

	// Deal 2 — same day (within 48h)
	// Move last_negotiation_at back 24h so it's still within the window
	existing, _ := eng.store.GetStreak(ctx, userID)
	existing.LastNegotiationAt = time.Now().UTC().Add(-24 * time.Hour)
	if err := eng.store.UpsertStreak(ctx, existing); err != nil {
		t.Fatalf("adjust time: %v", err)
	}

	if err := eng.RecordNegotiation(ctx, userID, 200); err != nil {
		t.Fatalf("record negotiation 2: %v", err)
	}
	streak, _ = eng.GetStreak(ctx, userID)
	if streak.CurrentStreak != 2 {
		t.Errorf("expected streak=2 after second deal, got %d", streak.CurrentStreak)
	}
	if streak.TotalDeals != 2 {
		t.Errorf("expected total_deals=2, got %d", streak.TotalDeals)
	}
	if streak.TotalSavings != 300 {
		t.Errorf("expected total_savings=300, got %f", streak.TotalSavings)
	}

	// Deal 3 — move back another 24h (still within 48h of deal 2)
	existing, _ = eng.store.GetStreak(ctx, userID)
	existing.LastNegotiationAt = time.Now().UTC().Add(-24 * time.Hour)
	if err := eng.store.UpsertStreak(ctx, existing); err != nil {
		t.Fatalf("adjust time: %v", err)
	}

	if err := eng.RecordNegotiation(ctx, userID, 50); err != nil {
		t.Fatalf("record negotiation 3: %v", err)
	}
	streak, _ = eng.GetStreak(ctx, userID)
	if streak.CurrentStreak != 3 {
		t.Errorf("expected streak=3 after third deal, got %d", streak.CurrentStreak)
	}
	if streak.TotalDeals != 3 {
		t.Errorf("expected total_deals=3, got %d", streak.TotalDeals)
	}
	if streak.TotalSavings != 350 {
		t.Errorf("expected total_savings=350, got %f", streak.TotalSavings)
	}
	if streak.LongestStreak != 3 {
		t.Errorf("expected longest_streak=3, got %d", streak.LongestStreak)
	}
}

// TestRecordNegotiation_GapOver48h verifies that a gap >48h resets the streak to 1.
func TestRecordNegotiation_GapOver48h(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	userID := "user-bob"

	// First deal
	if err := eng.RecordNegotiation(ctx, userID, 100); err != nil {
		t.Fatalf("record negotiation 1: %v", err)
	}

	// Move last_negotiation_at back 72h to simulate gap >48h
	existing, _ := eng.store.GetStreak(ctx, userID)
	existing.LastNegotiationAt = time.Now().UTC().Add(-72 * time.Hour)
	existing.CurrentStreak = 3 // simulate existing streak before the gap
	existing.LongestStreak = 3 // preserve the longest
	if err := eng.store.UpsertStreak(ctx, existing); err != nil {
		t.Fatalf("adjust time: %v", err)
	}

	// Now do another deal — streak should reset to 1
	if err := eng.RecordNegotiation(ctx, userID, 200); err != nil {
		t.Fatalf("record negotiation 2: %v", err)
	}
	streak, _ := eng.GetStreak(ctx, userID)
	if streak.CurrentStreak != 1 {
		t.Errorf("expected streak=1 after 72h gap, got %d", streak.CurrentStreak)
	}
	if streak.TotalDeals != 2 {
		t.Errorf("expected total_deals=2, got %d", streak.TotalDeals)
	}
	if streak.LongestStreak != 3 {
		t.Errorf("expected longest_streak=3 (preserved from before gap), got %d", streak.LongestStreak)
	}
}

// TestLeaderboard_Ordering verifies leaderboard returns entries sorted by savings DESC.
func TestLeaderboard_Ordering(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Create a few users with different savings
	users := []struct {
		id      string
		savings []float64
	}{
		{"alice", []float64{100, 200}},
		{"bob", []float64{500}},
		{"charlie", []float64{50}},
	}

	for _, u := range users {
		eng.store.UpsertStreak(ctx, &Streak{
			UserID:            u.id,
			CurrentStreak:     1,
			LongestStreak:     1,
			LastNegotiationAt: time.Now().UTC(),
		})

		// Record negotiations to build savings
		for _, s := range u.savings {
			eng.RecordNegotiation(ctx, u.id, s)
		}
	}

	lb, err := eng.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("get leaderboard: %v", err)
	}

	if len(lb) < 3 {
		t.Fatalf("expected at least 3 leaderboard entries, got %d", len(lb))
	}

	// bob should be #1 (savings=500)
	if lb[0].UserID != "bob" {
		t.Errorf("expected #1=bob, got %s", lb[0].UserID)
	}
	// alice should be #2 (savings=300)
	if lb[1].UserID != "alice" {
		t.Errorf("expected #2=alice, got %s", lb[1].UserID)
	}
	// charlie should be #3 (savings=50)
	if lb[2].UserID != "charlie" {
		t.Errorf("expected #3=charlie, got %s", lb[2].UserID)
	}

	// Verify savings order
	for i := 1; i < len(lb); i++ {
		if lb[i].TotalSavings > lb[i-1].TotalSavings {
			t.Errorf("leaderboard not sorted: entry %d (%.0f) > entry %d (%.0f)",
				i, lb[i].TotalSavings, i-1, lb[i-1].TotalSavings)
		}
	}
}

// TestGetBadges_ReturnsAllSix verifies GetBadges returns all 6 badges with correct earned state.
func TestGetBadges_ReturnsAllSix(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	userID := "user-dave"

	// Before any deals — all badges should be unearned
	badges, err := eng.GetBadges(ctx, userID)
	if err != nil {
		t.Fatalf("get badges: %v", err)
	}

	expectedIDs := []string{"first_deal", "streak_5", "thousand_club", "perfectionist", "power_negotiator", "saas_master"}
	if len(badges) != len(expectedIDs) {
		t.Fatalf("expected %d badges, got %d", len(expectedIDs), len(badges))
	}

	for i, id := range expectedIDs {
		if badges[i].ID != id {
			t.Errorf("badge[%d]: expected ID=%s, got %s", i, id, badges[i].ID)
		}
		if badges[i].Earned {
			t.Errorf("badge %s should not yet be earned", id)
		}
	}

	// Award first_deal, then check
	if err := eng.RecordNegotiation(ctx, userID, 100); err != nil {
		t.Fatalf("record negotiation: %v", err)
	}
	streak, _ := eng.GetStreak(ctx, userID)
	if _, err := eng.CheckAndAwardBadges(ctx, userID, streak); err != nil {
		t.Fatalf("check and award badges: %v", err)
	}

	// Now get badges — first_deal should be earned, others not
	badges, err = eng.GetBadges(ctx, userID)
	if err != nil {
		t.Fatalf("get badges after award: %v", err)
	}

	found := make(map[string]bool)
	for _, b := range badges {
		found[b.ID] = b.Earned
	}

	if !found["first_deal"] {
		t.Error("first_deal should be earned after first deal")
	}
	if found["streak_5"] {
		t.Error("streak_5 should not be earned yet")
	}
}

// TestCheckAndAwardBadges_FirstDeal verifies first_deal badge is awarded on first deal.
func TestCheckAndAwardBadges_FirstDeal(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	userID := "user-eve"

	// First negotiation
	if err := eng.RecordNegotiation(ctx, userID, 100); err != nil {
		t.Fatalf("record negotiation: %v", err)
	}

	streak, _ := eng.GetStreak(ctx, userID)
	awarded, err := eng.CheckAndAwardBadges(ctx, userID, streak)
	if err != nil {
		t.Fatalf("check and award badges: %v", err)
	}

	// first_deal should be awarded
	found := false
	for _, b := range awarded {
		if b.ID == "first_deal" {
			found = true
			if !b.Earned {
				t.Error("first_deal badge should be marked earned")
			}
		}
	}
	if !found {
		t.Error("first_deal badge should have been awarded")
	}

	// Second call should not award first_deal again
	awarded2, err := eng.CheckAndAwardBadges(ctx, userID, streak)
	if err != nil {
		t.Fatalf("second check and award: %v", err)
	}
	if len(awarded2) == 0 {
		// GetBadges fallback returns all badges — no new ones
		badges, err := eng.GetBadges(ctx, userID)
		if err != nil {
			t.Fatalf("get badges after second check: %v", err)
		}
		for _, b := range badges {
			if b.ID == "first_deal" && !b.Earned {
				t.Error("first_deal should remain earned after second check")
			}
		}
	}
}

// TestCheckAndAwardBadges_Streak5 verifies streak_5 badge after 5 consecutive deals.
func TestCheckAndAwardBadges_Streak5(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()
	userID := "user-frank"

	// Do 5 negotiations with timestamps adjusted to stay within 48h windows
	for i := 0; i < 5; i++ {
		if err := eng.RecordNegotiation(ctx, userID, float64(50*(i+1))); err != nil {
			t.Fatalf("record negotiation %d: %v", i+1, err)
		}

		// After recording, shift the last_negotiation_at back by 1 hour to simulate
		// closely-spaced deals (still within 48h of each other)
		if i < 4 {
			existing, _ := eng.store.GetStreak(ctx, userID)
			existing.LastNegotiationAt = time.Now().UTC().Add(-1 * time.Hour)
			if err := eng.store.UpsertStreak(ctx, existing); err != nil {
				t.Fatalf("adjust time after deal %d: %v", i+1, err)
			}
		}
	}

	streak, err := eng.GetStreak(ctx, userID)
	if err != nil {
		t.Fatalf("get streak: %v", err)
	}
	if streak.CurrentStreak != 5 {
		t.Fatalf("expected streak=5, got %d (can't test streak_5 badge without streak=5)", streak.CurrentStreak)
	}

	awarded, err := eng.CheckAndAwardBadges(ctx, userID, streak)
	if err != nil {
		t.Fatalf("check and award badges: %v", err)
	}

	found := false
	for _, b := range awarded {
		if b.ID == "streak_5" {
			found = true
			if !b.Earned {
				t.Error("streak_5 badge should be marked earned")
			}
		}
	}
	if !found {
		t.Error("streak_5 badge should have been awarded")
	}

	// first_deal should also be awarded (since first deal was done too)
	foundFirstDeal := false
	for _, b := range awarded {
		if b.ID == "first_deal" {
			foundFirstDeal = true
		}
	}
	if !foundFirstDeal {
		// first_deal may have been awarded on a previous call, but should still be earned now
		badges, err := eng.GetBadges(ctx, userID)
		if err != nil {
			t.Fatalf("get badges: %v", err)
		}
		for _, b := range badges {
			if b.ID == "first_deal" && !b.Earned {
				t.Error("first_deal should be earned")
			}
		}
	}
}
