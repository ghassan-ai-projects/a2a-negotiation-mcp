package vendorcomparison

// ComparisonRequest is the input for a multi-vendor comparison.
type ComparisonRequest struct {
	Category string   `json:"category"`
	Features []string `json:"features,omitempty"`
	Seats    int      `json:"seats"`
}

// VendorComparison is a single vendor entry in a comparison result.
type VendorComparison struct {
	Vendor           string  `json:"vendor"`
	SKU              string  `json:"sku"`
	ListPrice        float64 `json:"list_price"`
	TypicalPrice     float64 `json:"typical_price"`
	AnnualCost       float64 `json:"annual_cost"`
	SavingsPotential float64 `json:"savings_potential"`
	Category         string  `json:"category"`
	Score            int     `json:"score"`
}

// ComparisonResult is the output of a multi-vendor comparison.
type ComparisonResult struct {
	Category    string             `json:"category"`
	Comparisons []VendorComparison `json:"comparisons"`
	TopPick     string             `json:"top_pick"`
	AvgPrice    float64            `json:"avg_price"`
}
