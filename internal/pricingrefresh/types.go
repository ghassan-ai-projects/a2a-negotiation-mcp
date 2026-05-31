package pricingrefresh

// RefreshInput defines which vendors to refresh and the data source.
type RefreshInput struct {
	Vendors []string `json:"vendors"`
	Source  string   `json:"source"`
}

// RefreshResult summarizes the refresh operation.
type RefreshResult struct {
	VendorsRefreshed int   `json:"vendors_refreshed"`
	RecordsUpdated   int   `json:"records_updated"`
	DurationMs       int64 `json:"duration_ms"`
}
