package coverage

// PackageCoverage represents coverage data for a single package.
type PackageCoverage struct {
	Name        string  `json:"name"`
	CoveragePct float64 `json:"coverage_pct"`
	TestCount   int     `json:"test_count"`
}

// CoverageReport is the top-level report returned by Run().
type CoverageReport struct {
	OverallPct       float64           `json:"overall_pct"`
	Packages         []PackageCoverage `json:"packages"`
	TotalTests       int               `json:"total_tests"`
	UntestedPackages []string          `json:"untested_packages"`
	Recommendation   string            `json:"recommendation"`
}
