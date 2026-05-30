package export

// ExportRequest describes the parameters for a data export operation.
type ExportRequest struct {
	Format   string `json:"format"`            // "csv" or "json"
	Type     string `json:"type"`              // "deals", "sessions", "analytics", "all"
	Vendor   string `json:"vendor,omitempty"`
	DateFrom string `json:"date_from,omitempty"`
	DateTo   string `json:"date_to,omitempty"`
}

// ExportResult is the result of a data export operation.
type ExportResult struct {
	ExportID    int64  `json:"export_id"`
	Format      string `json:"format"`
	ExportType  string `json:"export_type"`
	RecordCount int    `json:"record_count"`
	Data        string `json:"data"`
	Filename    string `json:"filename"`
	GeneratedAt string `json:"generated_at"`
}

// exportRecord is the internal database row for a saved export.
type exportRecord struct {
	ID          int64
	UserID      string
	Format      string
	ExportType  string
	RecordCount int
	CreatedAt   string
}
