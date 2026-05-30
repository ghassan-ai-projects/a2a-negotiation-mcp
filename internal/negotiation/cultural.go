package negotiation

// CulturalProfile defines how negotiation parameters should be adjusted
// for a given cultural region.
type CulturalProfile struct {
	Region            string `json:"region"`
	Directness        string `json:"directness"`         // "direct", "indirect", "moderate"
	Pace              string `json:"pace"`               // "fast", "moderate", "slow"
	Formality         string `json:"formality"`          // "formal", "casual"
	RelationshipFirst bool   `json:"relationship_first"` // whether relationship building precedes business
	AdjustmentDesc    string `json:"adjustment_desc"`    // human-readable description of adjustments
}

// CulturalProfiles maps region names to their cultural negotiation profiles.
var CulturalProfiles = map[string]CulturalProfile{
	"germany": {
		Region: "germany", Directness: "direct", Pace: "fast",
		Formality: "formal", RelationshipFirst: false,
		AdjustmentDesc: "Direct offers, fewer rounds, smaller initial discount",
	},
	"japan": {
		Region: "japan", Directness: "indirect", Pace: "slow",
		Formality: "formal", RelationshipFirst: true,
		AdjustmentDesc: "More rounds, smaller concessions, relationship-building pauses",
	},
	"us": {
		Region: "us", Directness: "direct", Pace: "fast",
		Formality: "casual", RelationshipFirst: false,
		AdjustmentDesc: "Fast pace, direct offers, standard concession curve",
	},
	"france": {
		Region: "france", Directness: "moderate", Pace: "moderate",
		Formality: "formal", RelationshipFirst: true,
		AdjustmentDesc: "Moderate pace, formal language, relationship-first",
	},
	"uk": {
		Region: "uk", Directness: "indirect", Pace: "moderate",
		Formality: "formal", RelationshipFirst: false,
		AdjustmentDesc: "Polite indirectness, formal tone, standard rounds",
	},
	"brazil": {
		Region: "brazil", Directness: "indirect", Pace: "slow",
		Formality: "casual", RelationshipFirst: true,
		AdjustmentDesc: "Slow pace, warm relationship building, flexible terms",
	},
	"china": {
		Region: "china", Directness: "indirect", Pace: "slow",
		Formality: "formal", RelationshipFirst: true,
		AdjustmentDesc: "Face-saving indirectness, guanxi relationship-first",
	},
	"uae": {
		Region: "uae", Directness: "moderate", Pace: "moderate",
		Formality: "formal", RelationshipFirst: true,
		AdjustmentDesc: "Relationship before business, formal, moderate pace",
	},
}

// GetCulturalProfile returns the profile for a region. Returns nil if not found.
func GetCulturalProfile(region string) *CulturalProfile {
	p, ok := CulturalProfiles[region]
	if !ok {
		return nil
	}
	return &p
}

// ListCulturalProfiles returns all available profiles.
func ListCulturalProfiles() []CulturalProfile {
	profiles := make([]CulturalProfile, 0, len(CulturalProfiles))
	for _, p := range CulturalProfiles {
		profiles = append(profiles, p)
	}
	return profiles
}

// ApplyCulturalAdjustment modifies strategy parameters based on culture.
// It adjusts MaxRounds, ConcessionPerRound, and MaxConcessions to reflect
// cultural negotiation norms.
func ApplyCulturalAdjustment(strategy *Strategy, culture string) {
	if strategy == nil {
		return
	}
	profile := GetCulturalProfile(culture)
	if profile == nil {
		return
	}

	switch profile.Region {
	case "japan":
		// More rounds, smaller concessions, relationship-building pauses
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.4)
		if strategy.MaxRounds < 5 {
			strategy.MaxRounds = 5
		}
		strategy.ConcessionPerRound *= 0.67
		strategy.MaxConcessions = int(float64(strategy.MaxConcessions) * 1.5)
		if strategy.MaxConcessions < strategy.MaxRounds {
			strategy.MaxConcessions = strategy.MaxRounds
		}

	case "germany":
		// Direct offers, fewer rounds, smaller initial discount
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 0.8)
		if strategy.MaxRounds < 2 {
			strategy.MaxRounds = 2
		}
		strategy.ConcessionPerRound *= 0.8

	case "us":
		// Fast pace, standard concessions (baseline — minimal change)
		// Default values are already US-aligned

	case "france":
		// Moderate pace, formal approach
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.15)
		strategy.ConcessionPerRound *= 0.9

	case "uk":
		// Polite indirectness, slightly more rounds
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.1)
		strategy.ConcessionPerRound *= 0.9

	case "brazil":
		// Slow pace, warm relationship building, flexible terms
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.3)
		if strategy.MaxRounds < 4 {
			strategy.MaxRounds = 4
		}
		strategy.ConcessionPerRound *= 0.75
		strategy.MaxConcessions = int(float64(strategy.MaxConcessions) * 1.25)

	case "china":
		// Face-saving indirectness, relationship-first
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.35)
		if strategy.MaxRounds < 5 {
			strategy.MaxRounds = 5
		}
		strategy.ConcessionPerRound *= 0.7
		strategy.MaxConcessions = int(float64(strategy.MaxConcessions) * 1.4)
		if strategy.MaxConcessions < strategy.MaxRounds {
			strategy.MaxConcessions = strategy.MaxRounds
		}

	case "uae":
		// Relationship before business, formal, moderate pace
		strategy.MaxRounds = int(float64(strategy.MaxRounds) * 1.2)
		strategy.ConcessionPerRound *= 0.85
	}
}
