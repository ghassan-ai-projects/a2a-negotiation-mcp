package negotiation

import (
	"context"
	"testing"
)

func TestGetCulturalProfile_KnownRegions(t *testing.T) {
	tests := []struct {
		region   string
		expected string // directness
	}{
		{"germany", "direct"},
		{"japan", "indirect"},
		{"us", "direct"},
		{"france", "moderate"},
		{"uk", "indirect"},
		{"brazil", "indirect"},
		{"china", "indirect"},
		{"uae", "moderate"},
	}

	for _, tc := range tests {
		t.Run(tc.region, func(t *testing.T) {
			p := GetCulturalProfile(tc.region)
			if p == nil {
				t.Fatalf("GetCulturalProfile(%q) returned nil", tc.region)
			}
			if p.Directness != tc.expected {
				t.Errorf("GetCulturalProfile(%q).Directness = %q, want %q", tc.region, p.Directness, tc.expected)
			}
			if p.Region != tc.region {
				t.Errorf("GetCulturalProfile(%q).Region = %q", tc.region, p.Region)
			}
		})
	}
}

func TestGetCulturalProfile_UnknownReturnsNil(t *testing.T) {
	p := GetCulturalProfile("nonexistent")
	if p != nil {
		t.Errorf("expected nil for unknown region, got %+v", p)
	}
}

func TestListCulturalProfiles(t *testing.T) {
	profiles := ListCulturalProfiles()
	if len(profiles) != 8 {
		t.Errorf("expected 8 profiles, got %d", len(profiles))
	}

	// Verify all regions are present in the list
	seen := make(map[string]bool)
	for _, p := range profiles {
		seen[p.Region] = true
	}
	expectedRegions := []string{"germany", "japan", "us", "france", "uk", "brazil", "china", "uae"}
	for _, r := range expectedRegions {
		if !seen[r] {
			t.Errorf("expected region %q in list", r)
		}
	}
}

func TestApplyCulturalAdjustment_Japan(t *testing.T) {
	strategy := &Strategy{
		Name:               "balanced",
		MaxRounds:          4,
		MaxConcessions:     3,
		ConcessionPerRound: 0.04,
	}

	ApplyCulturalAdjustment(strategy, "japan")

	// Japan: slower pace, more rounds, smaller concessions
	if strategy.MaxRounds < 5 {
		t.Errorf("Japan MaxRounds: expected >= 5, got %d", strategy.MaxRounds)
	}
	if strategy.ConcessionPerRound >= 0.04 {
		t.Errorf("Japan ConcessionPerRound: expected < 0.04, got %f", strategy.ConcessionPerRound)
	}
	if strategy.MaxConcessions < strategy.MaxRounds {
		t.Errorf("Japan MaxConcessions (%d) should be >= MaxRounds (%d)", strategy.MaxConcessions, strategy.MaxRounds)
	}
}

func TestApplyCulturalAdjustment_Germany(t *testing.T) {
	strategy := &Strategy{
		Name:               "balanced",
		MaxRounds:          4,
		ConcessionPerRound: 0.04,
	}

	ApplyCulturalAdjustment(strategy, "germany")

	// Germany: fewer rounds, smaller concessions
	if strategy.MaxRounds > 4 {
		t.Errorf("Germany MaxRounds: expected <= 4, got %d", strategy.MaxRounds)
	}
	if strategy.ConcessionPerRound >= 0.04 {
		t.Errorf("Germany ConcessionPerRound: expected < 0.04, got %f", strategy.ConcessionPerRound)
	}
}

func TestApplyCulturalAdjustment_NilStrategy(t *testing.T) {
	// Should not panic
	ApplyCulturalAdjustment(nil, "japan")
}

func TestApplyCulturalAdjustment_UnknownCulture(t *testing.T) {
	strategy := &Strategy{
		Name:               "balanced",
		MaxRounds:          4,
		ConcessionPerRound: 0.04,
	}

	// Should not modify strategy for unknown culture
	originalMaxRounds := strategy.MaxRounds
	originalConcession := strategy.ConcessionPerRound

	ApplyCulturalAdjustment(strategy, "mars")

	if strategy.MaxRounds != originalMaxRounds {
		t.Errorf("expected no change for unknown culture, MaxRounds changed from %d to %d", originalMaxRounds, strategy.MaxRounds)
	}
	if strategy.ConcessionPerRound != originalConcession {
		t.Errorf("expected no change for unknown culture, ConcessionPerRound changed from %f to %f", originalConcession, strategy.ConcessionPerRound)
	}
}

func TestCreateSession_WithCulture(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	session, err := engine.CreateSession(ctx, "Slack", "Pro", "balanced", 0, nil, "japan")
	if err != nil {
		t.Fatalf("CreateSession with culture failed: %v", err)
	}

	if session.Culture != "japan" {
		t.Errorf("expected culture 'japan', got %q", session.Culture)
	}
	if session.Status != "active" {
		t.Errorf("expected status active, got %s", session.Status)
	}
}

func TestCreateSession_WithUSCulture(t *testing.T) {
	engine, _ := setupTest(t)
	ctx := context.Background()

	session, err := engine.CreateSession(ctx, "Slack", "Pro", "balanced", 0, nil, "us")
	if err != nil {
		t.Fatalf("CreateSession with US culture failed: %v", err)
	}

	if session.Culture != "us" {
		t.Errorf("expected culture 'us', got %q", session.Culture)
	}

	// US culture should produce standard balanced strategy (20% off 8.75 = 7.00)
	expectedOffer := 8.75 * 0.80
	if session.CurrentOffer != expectedOffer {
		t.Errorf("expected offer %f, got %f", expectedOffer, session.CurrentOffer)
	}
}
