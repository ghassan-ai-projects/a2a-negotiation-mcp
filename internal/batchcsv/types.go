package batchcsv

// CSVRow represents a single row from the uploaded CSV.
type CSVRow struct {
	Vendor      string  `json:"vendor"`
	Strategy    string  `json:"strategy"`
	Budget      float64 `json:"budget"`
	TargetPrice float64 `json:"target_price"`
	Notes       string  `json:"notes"`
}

// BatchUploadResult holds the outcome of processing a CSV upload.
type BatchUploadResult struct {
	CreatedCount int      `json:"created_count"`
	RowCount     int      `json:"row_count"`
	Errors       []string `json:"errors"`
}
