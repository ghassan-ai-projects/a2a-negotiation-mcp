package contract

// ContractTerms holds the extracted key terms from a parsed contract text.
type ContractTerms struct {
	Vendor            string `json:"vendor"`
	SKU               string `json:"sku,omitempty"`
	StartDate         string `json:"start_date,omitempty"`
	EndDate           string `json:"end_date"`
	RenewalTermDays   int    `json:"renewal_term_days"` // days before renewal to act
	AutoRenew         bool   `json:"auto_renew"`
	PriceLockPeriod   string `json:"price_lock_period,omitempty"`
	TerminationNotice int    `json:"termination_notice_days"`
	AnnualContract    bool   `json:"annual_contract"`
	DataPortability   bool   `json:"data_portability,omitempty"`
	Confidence        string `json:"confidence"` // "high", "medium", "low" per field
}

// PerFieldConfidence holds confidence levels for individual extracted fields.
type PerFieldConfidence struct {
	EndDate           string `json:"end_date"`
	AutoRenew         string `json:"auto_renew"`
	TerminationNotice string `json:"termination_notice"`
	PriceLockPeriod   string `json:"price_lock_period"`
	AnnualContract    string `json:"annual_contract"`
	Pricing           string `json:"pricing,omitempty"`
}

// ContractParseResult is the full output of parsing a contract text.
type ContractParseResult struct {
	RawText       string             `json:"raw_text"`
	Terms         ContractTerms      `json:"terms"`
	FieldConf     PerFieldConfidence `json:"field_confidence"`
	Warnings      []string           `json:"warnings,omitempty"`
	AutoPopulated bool               `json:"auto_populated"`
}

// PricingExtract holds price-related information parsed from contract text.
type PricingExtract struct {
	Amount     float64 `json:"amount"`
	Unit       string  `json:"unit"` // "per_seat_month", "per_user_year", "flat_monthly", etc.
	Confidence string  `json:"confidence"`
}
