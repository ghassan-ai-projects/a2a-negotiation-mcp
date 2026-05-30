package parallel_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/parallel"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/google/uuid"
)

// setupParallelTest creates infrastructure for parallel tests.
// Returns the parallel engine, pricing store, history store, and negotiation engine.
func setupParallelTest(t *testing.T) (*parallel.Engine, *pricing.Store, *history.Store, *negotiation.Engine) {
	t.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pstore.Close() })

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	// Seed test pricing data
	ctx := context.Background()
	vendors := []struct {
		name, category, sku, desc string
		listPrice, minObs, maxObs float64
		typicalPct                float64
		unit                      string
	}{
		{"VendorA", "Category", "Standard", "Standard plan", 100.00, 70.00, 95.00, 15, "per_seat_month"},
		{"VendorB", "Category", "Standard", "Standard plan", 200.00, 140.00, 190.00, 20, "per_seat_month"},
		{"VendorC", "Category", "Standard", "Standard plan", 50.00, 35.00, 48.00, 10, "per_seat_month"},
	}
	for _, v := range vendors {
		_, err := pstore.DB().ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", v.name, v.category)
		if err != nil {
			t.Fatalf("insert vendor %s: %v", v.name, err)
		}
		var vid int64
		err = pstore.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", v.name).Scan(&vid)
		if err != nil {
			t.Fatalf("get vendor id %s: %v", v.name, err)
		}
		_, err = pstore.DB().ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
		`, vid, v.sku, v.desc, v.listPrice, v.minObs, v.maxObs, v.typicalPct, v.unit)
		if err != nil {
			t.Fatalf("insert pricing %s/%s: %v", v.name, v.sku, err)
		}
	}

	negEng := negotiation.NewEngine(pstore)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parEng := parallel.NewEngine(negEng, hstore, pstore, logger)
	return parEng, pstore, hstore, negEng
}

// createAndSaveSession creates a negotiation session and saves it to history.
// Returns the session ID.
func createAndSaveSession(t *testing.T, ctx context.Context, negEng *negotiation.Engine, hstore *history.Store, vendor, sku, strategy string) string {
	t.Helper()

	session, err := negEng.CreateSession(ctx, vendor, sku, strategy, 0, nil)
	if err != nil {
		t.Fatalf("CreateSession(%s): %v", vendor, err)
	}

	session.ID = uuid.New().String()

	rec := &history.SessionRecord{
		ID:           session.ID,
		Vendor:       session.Vendor,
		SKU:          session.SKU,
		Strategy:     session.Strategy,
		Budget:       session.Budget,
		Status:       session.Status,
		CurrentOffer: session.CurrentOffer,
		ListPrice:    session.ListPrice,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}
	if err := hstore.SaveSession(ctx, rec); err != nil {
		t.Fatalf("SaveSession(%s): %v", session.ID, err)
	}

	return session.ID
}

// ─── RunParallel Tests ───

func TestRunParallel_TwoSessions_BestPrice(t *testing.T) {
	parEng, _, hstore, negEng := setupParallelTest(t)
	ctx := context.Background()

	// VendorB has higher list price (200.00) so balanced strategy @ 20% off = 160.00
	// VendorA has lower list price (100.00) so balanced strategy @ 20% off = 80.00
	idA := createAndSaveSession(t, ctx, negEng, hstore, "VendorA", "Standard", "balanced")
	idB := createAndSaveSession(t, ctx, negEng, hstore, "VendorB", "Standard", "balanced")

	result, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{idA, idB},
		Strategy:   "best_price",
		Timeout:    10,
	})
	if err != nil {
		t.Fatalf("RunParallel: %v", err)
	}

	if result.WinnerSessionID != idA {
		t.Errorf("expected winner %s (VendorA, lower price), got %s (%s)", idA, result.WinnerSessionID, result.WinnerVendor)
	}
	if result.WinnerVendor != "VendorA" {
		t.Errorf("expected winner VendorA, got %s", result.WinnerVendor)
	}
	if result.WinnerOffer <= 0 {
		t.Errorf("expected positive WinnerOffer, got %f", result.WinnerOffer)
	}
	if result.Strategy != "best_price" {
		t.Errorf("expected strategy best_price, got %s", result.Strategy)
	}
	if len(result.AllResults) != 2 {
		t.Errorf("expected 2 AllResults, got %d", len(result.AllResults))
	}
	if result.DurationMs < 0 {
		t.Errorf("expected positive DurationMs, got %d", result.DurationMs)
	}
}

func TestRunParallel_TwoSessions_BestDiscount(t *testing.T) {
	parEng, _, hstore, negEng := setupParallelTest(t)
	ctx := context.Background()

	// VendorC: aggressive strategy, 30% initial, goes up to 45%, list price 50.00
	// VendorB: balanced strategy, 20% initial, goes up to 35%, list price 200.00
	// The aggressive strategy on VendorC will yield higher discount percentage
	idC := createAndSaveSession(t, ctx, negEng, hstore, "VendorC", "Standard", "aggressive")
	idB := createAndSaveSession(t, ctx, negEng, hstore, "VendorB", "Standard", "balanced")

	result, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{idC, idB},
		Strategy:   "best_discount",
		Timeout:    10,
	})
	if err != nil {
		t.Fatalf("RunParallel: %v", err)
	}

	// Aggressive strategy yields higher discount than balanced
	if result.WinnerSessionID != idC {
		t.Errorf("expected winner %s (aggressive, higher discount), got %s (%s)", idC, result.WinnerSessionID, result.WinnerVendor)
	}
	if result.WinnerVendor != "VendorC" {
		t.Errorf("expected winner VendorC, got %s", result.WinnerVendor)
	}
	if result.WinnerDiscount <= 0 {
		t.Errorf("expected positive WinnerDiscount, got %f", result.WinnerDiscount)
	}
	if result.Strategy != "best_discount" {
		t.Errorf("expected strategy best_discount, got %s", result.Strategy)
	}
	if len(result.AllResults) != 2 {
		t.Errorf("expected 2 AllResults, got %d", len(result.AllResults))
	}
}

func TestRunParallel_EmptySessions(t *testing.T) {
	parEng, _, _, _ := setupParallelTest(t)
	ctx := context.Background()

	_, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{},
		Strategy:   "best_price",
		Timeout:    10,
	})
	if err == nil {
		t.Fatal("expected error for empty sessions, got nil")
	}
}

func TestRunParallel_InvalidSession(t *testing.T) {
	parEng, _, _, _ := setupParallelTest(t)
	ctx := context.Background()

	_, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{"nonexistent-session-id"},
		Strategy:   "best_price",
		Timeout:    10,
	})
	if err == nil {
		t.Fatal("expected error for invalid session, got nil")
	}
}

// ─── selectWinner Unit Tests ───

func TestSelectWinner_BestPrice(t *testing.T) {
	// Use reflection to test private selectWinner via the public API
	parEng, _, hstore, negEng := setupParallelTest(t)
	ctx := context.Background()

	idA := createAndSaveSession(t, ctx, negEng, hstore, "VendorA", "Standard", "balanced") // 80.00
	idB := createAndSaveSession(t, ctx, negEng, hstore, "VendorB", "Standard", "balanced") // 160.00

	result, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{idA, idB},
		Strategy:   "best_price",
		Timeout:    10,
	})
	if err != nil {
		t.Fatalf("RunParallel: %v", err)
	}

	if result.WinnerSessionID != idA {
		t.Errorf("best_price: expected winner %s (lower price), got %s (%s)", idA, result.WinnerSessionID, result.WinnerVendor)
	}

	// Verify AllResults offers
	for _, r := range result.AllResults {
		if r.SessionID == idA && r.Offer >= 100 {
			t.Errorf("VendorA offer %f should be less than list price 100", r.Offer)
		}
		if r.SessionID == idB && r.Offer >= 200 {
			t.Errorf("VendorB offer %f should be less than list price 200", r.Offer)
		}
	}
}

func TestSelectWinner_BestDiscount(t *testing.T) {
	parEng, _, hstore, negEng := setupParallelTest(t)
	ctx := context.Background()

	idC := createAndSaveSession(t, ctx, negEng, hstore, "VendorC", "Standard", "aggressive")   // highest discount strategy
	idB := createAndSaveSession(t, ctx, negEng, hstore, "VendorB", "Standard", "conservative") // lowest discount strategy

	result, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{idC, idB},
		Strategy:   "best_discount",
		Timeout:    10,
	})
	if err != nil {
		t.Fatalf("RunParallel: %v", err)
	}

	// Agressive should have higher discount than conservative
	if result.WinnerSessionID != idC {
		t.Errorf("best_discount: expected winner %s (aggressive -> higher discount), got %s (%s)", idC, result.WinnerSessionID, result.WinnerVendor)
	}
}

func TestSelectWinner_Fastest(t *testing.T) {
	parEng, _, hstore, negEng := setupParallelTest(t)
	ctx := context.Background()

	// Conservative has 3 max rounds, aggressive has 5 max rounds
	// With no auto-approve threshold, the negotiation runs to completion
	// The session that finishes in fewer rounds wins (conservative = 3, aggressive = 5)
	idCons := createAndSaveSession(t, ctx, negEng, hstore, "VendorA", "Standard", "conservative")
	idAgg := createAndSaveSession(t, ctx, negEng, hstore, "VendorB", "Standard", "aggressive")

	result, err := parEng.RunParallel(ctx, parallel.ParallelConfig{
		SessionIDs: []string{idCons, idAgg},
		Strategy:   "fastest",
		Timeout:    10,
	})
	if err != nil {
		t.Fatalf("RunParallel: %v", err)
	}

	// Conservative has 3 max rounds (less than aggressive's 5)
	if result.WinnerSessionID != idCons {
		t.Errorf("fastest: expected winner %s (conservative, fewer rounds), got %s (%s)", idCons, result.WinnerSessionID, result.WinnerVendor)
	}

	// Verify that conservative indeed has fewer rounds
	for _, r := range result.AllResults {
		if r.SessionID == idCons && r.Rounds > 4 {
			t.Errorf("conservative should have <= 4 rounds (3 buyer + 1 seller accept), got %d", r.Rounds)
		}
		if r.SessionID == idAgg && r.Rounds <= 4 {
			t.Errorf("aggressive should have > 4 rounds (5 buyer + 1 seller accept), got %d", r.Rounds)
		}
	}
}
