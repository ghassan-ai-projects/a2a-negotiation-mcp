package compliance

// ComplianceFlag represents a single regulatory compliance flag.
type ComplianceFlag struct {
	RuleID         string `json:"rule_id"`
	Description    string `json:"description"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation"`
}

// ComplianceResult holds the overall compliance check outcome.
type ComplianceResult struct {
	Terms         string            `json:"terms"`
	Jurisdiction  string            `json:"jurisdiction"`
	OverallStatus string            `json:"overall_status"`
	Flags         []ComplianceFlag  `json:"flags"`
	PassCount     int               `json:"pass_count"`
	FlagCount     int               `json:"flag_count"`
}
