package reports

// ReportRequest defines the sections and filters for a custom report.
type ReportRequest struct {
	Sections []string `json:"sections"`          // savings, vendor_breakdown, win_loss, benchmarks, budget, trends
	Period   string   `json:"period"`            // 30d, 90d, 1y, all
	Vendor   string   `json:"vendor,omitempty"`
}

// ReportResult holds the generated report data.
type ReportResult struct {
	Sections     map[string]any `json:"sections"`
	GeneratedAt  string         `json:"generated_at"`
	SectionCount int            `json:"section_count"`
}
