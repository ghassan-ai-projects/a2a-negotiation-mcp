package dataimport

// ImportType is the type of data being imported.
type ImportType string

const (
	ImportTypeDeals   ImportType = "deals"
	ImportTypePricing ImportType = "pricing"
)

// ImportMode is the import mode.
type ImportMode string

const (
	ImportModeValidate ImportMode = "validate"
	ImportModeImport   ImportMode = "import"
)

// ImportRequest represents a data import request.
type ImportRequest struct {
	Type   ImportType `json:"type"`
	Data   string     `json:"data"`
	Mode   ImportMode `json:"mode"`
	DryRun bool       `json:"dry_run"`
}

// ImportResult represents the result of an import operation.
type ImportResult struct {
	ValidCount    int      `json:"valid_count"`
	ImportedCount int      `json:"imported_count"`
	SkippedCount  int      `json:"skipped_count"`
	Errors        []string `json:"errors"`
	Summary       string   `json:"summary"`
}

// dealRecord represents a single deal in JSON import data.
type dealRecord struct {
	Vendor      string  `json:"vendor"`
	SKU         string  `json:"sku"`
	ListPrice   float64 `json:"list_price"`
	FinalPrice  float64 `json:"final_price"`
	DiscountPct float64 `json:"discount_percentage"`
	Seats       int     `json:"seats"`
	TermMonths  int     `json:"term_months"`
	Strategy    string  `json:"strategy"`
}

// pricingRecord represents a single pricing entry in JSON import data.
type pricingRecord struct {
	Vendor      string  `json:"vendor"`
	Category    string  `json:"category"`
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	ListPrice   float64 `json:"list_price"`
	MinObserved float64 `json:"min_observed"`
	MaxObserved float64 `json:"max_observed"`
	TypicalPct  float64 `json:"typical_pct"`
	Unit        string  `json:"unit"`
}
