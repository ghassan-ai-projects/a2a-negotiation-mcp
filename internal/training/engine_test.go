package training

import (
	"testing"
)

func TestSimulate_SuccessfulWithStandardParams(t *testing.T) {
	eng := NewEngine()
	result, err := eng.Simulate("AcmeCorp", "collaborative", 10000.0, 5)
	if err != nil {
		t.Fatalf("Simulate() returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Simulate() returned nil result")
	}
	if result.Vendor != "AcmeCorp" {
		t.Errorf("expected vendor AcmeCorp, got %s", result.Vendor)
	}
	if result.Strategy != "collaborative" {
		t.Errorf("expected strategy collaborative, got %s", result.Strategy)
	}
	if result.Budget != 10000.0 {
		t.Errorf("expected budget 10000, got %.2f", result.Budget)
	}
	if result.TotalRounds != 5 {
		t.Errorf("expected total_rounds 5, got %d", result.TotalRounds)
	}
	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if result.FinalOutcome == "" {
		t.Error("expected non-empty final_outcome")
	}
	if len(result.Lessons) < 2 {
		t.Errorf("expected at least 2 lessons, got %d", len(result.Lessons))
	}
}

func TestSimulate_InvalidStrategy(t *testing.T) {
	eng := NewEngine()
	_, err := eng.Simulate("AcmeCorp", "invalid", 10000.0, 3)
	if err == nil {
		t.Fatal("expected error for invalid strategy, got nil")
	}
}

func TestSimulate_EmptyVendor(t *testing.T) {
	eng := NewEngine()
	_, err := eng.Simulate("", "competitive", 10000.0, 3)
	if err == nil {
		t.Fatal("expected error for empty vendor, got nil")
	}
}

func TestSimulate_RoundsOutOfRange(t *testing.T) {
	eng := NewEngine()

	// Zero rounds
	_, err := eng.Simulate("AcmeCorp", "competitive", 10000.0, 0)
	if err == nil {
		t.Fatal("expected error for 0 rounds, got nil")
	}

	// More than 10 rounds
	_, err = eng.Simulate("AcmeCorp", "competitive", 10000.0, 11)
	if err == nil {
		t.Fatal("expected error for 11 rounds, got nil")
	}
}

func TestSimulate_BudgetZero(t *testing.T) {
	eng := NewEngine()
	_, err := eng.Simulate("AcmeCorp", "competitive", 0, 3)
	if err == nil {
		t.Fatal("expected error for zero budget, got nil")
	}

	_, err = eng.Simulate("AcmeCorp", "competitive", -100, 3)
	if err == nil {
		t.Fatal("expected error for negative budget, got nil")
	}
}

func TestSimulate_ResultsContainExpectedRounds(t *testing.T) {
	eng := NewEngine()

	for _, rounds := range []int{1, 3, 7, 10} {
		result, err := eng.Simulate("TestVendor", "competitive", 5000.0, rounds)
		if err != nil {
			t.Fatalf("Simulate() with %d rounds returned error: %v", rounds, err)
		}
		if len(result.Rounds) != rounds {
			t.Errorf("expected %d rounds, got %d", rounds, len(result.Rounds))
		}
		for i, r := range result.Rounds {
			if r.RoundNumber != i+1 {
				t.Errorf("round %d: expected round_number %d, got %d", i, i+1, r.RoundNumber)
			}
			if r.Offer <= 0 {
				t.Errorf("round %d: expected positive offer, got %.2f", i, r.Offer)
			}
			if r.Counterparty != "vendor" && r.Counterparty != "buyer" {
				t.Errorf("round %d: expected counterparty vendor or buyer, got %s", i, r.Counterparty)
			}
			if r.Note == "" {
				t.Errorf("round %d: expected non-empty note", i)
			}
		}
	}
}

func TestSimulate_AllStrategiesProduceValidResults(t *testing.T) {
	eng := NewEngine()
	strategies := []string{"competitive", "collaborative", "aggressive", "concessionary", "principled"}

	for _, s := range strategies {
		result, err := eng.Simulate("VendorX", s, 10000.0, 4)
		if err != nil {
			t.Fatalf("Simulate() with strategy %q returned error: %v", s, err)
		}
		if len(result.Rounds) != 4 {
			t.Errorf("strategy %q: expected 4 rounds, got %d", s, len(result.Rounds))
		}
		if result.TotalDiscount < 0 || result.TotalDiscount > 100 {
			t.Errorf("strategy %q: discount %.2f out of range [0, 100]", s, result.TotalDiscount)
		}
	}
}
