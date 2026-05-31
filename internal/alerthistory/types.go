package alerthistory

// AlertEntry represents a single alert record merged from multiple sources.
type AlertEntry struct {
	ID        int64  `json:"id"`
	AlertType string `json:"alert_type"`
	Vendor    string `json:"vendor"`
	Message   string `json:"message"`
	Level     string `json:"level"`
	CreatedAt string `json:"created_at"`
}

// AlertFeed wraps a list of alert entries grouped by type.
type AlertFeed struct {
	Entries []AlertEntry            `json:"entries"`
	Grouped map[string][]AlertEntry `json:"grouped"`
}
