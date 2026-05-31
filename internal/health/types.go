package health

import "time"

// VendorHealth represents a vendor's financial/market health assessment.
type VendorHealth struct {
	Vendor      string    `json:"vendor"`
	Score       int       `json:"score"`    // 1-100, lower = struggling = easier to negotiate
	Category    string    `json:"category"` // "struggling", "stable", "growing"
	Signals     []Signal  `json:"signals"`
	LastUpdated time.Time `json:"last_updated"`
}

// Signal is an individual data point that influences vendor health.
type Signal struct {
	ID     int64  `json:"id,omitempty"`
	Type   string `json:"type"`   // "layoff", "funding", "growth", "acquisition", "lawsuit", "ipo"
	Source string `json:"source"` // "crunchbase", "news", "manual"
	Detail string `json:"detail"`
	Weight int    `json:"weight"` // -20 to +20 (negative = worse for vendor)
	Date   string `json:"date"`
}

// NegotiationLeverage is the derived advice from vendor health data.
type NegotiationLeverage struct {
	Vendor     string       `json:"vendor"`
	Health     VendorHealth `json:"health"`
	Leverage   string       `json:"leverage"`   // "high" (score <30), "medium" (30-60), "low" (>60)
	Suggestion string       `json:"suggestion"` // e.g., "Push hard — they just had layoffs"
}

// SignalTypeWeights defines the default weight for each signal type.
var SignalTypeWeights = map[string]int{
	"layoff":      -15,
	"lawsuit":     -20,
	"funding":     +10,
	"growth":      +15,
	"ipo":         +20,
	"acquisition": +5,
}
