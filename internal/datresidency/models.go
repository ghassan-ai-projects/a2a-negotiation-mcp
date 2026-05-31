package datresidency

// ResidencyRule represents a data residency rule for a region.
type ResidencyRule struct {
	ID        int    `json:"id"`
	Region    string `json:"region"`
	Allowed   bool   `json:"allowed"`
	UpdatedAt string `json:"updated_at"`
}

// VendorResidencyCheck represents the result of a vendor residency compliance check.
type VendorResidencyCheck struct {
	Vendor    string `json:"vendor"`
	Region    string `json:"region"`
	Compliant bool   `json:"compliant"`
	RuleFound bool   `json:"rule_found"`
}
