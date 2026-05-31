package commlog

// CommEntry represents a single vendor communication record.
type CommEntry struct {
	ID        int64  `json:"id"`
	Vendor    string `json:"vendor"`
	CommType  string `json:"comm_type"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail,omitempty"`
	CreatedAt string `json:"created_at"`
}

// CommLogResult wraps a list of communication entries with a total count.
type CommLogResult struct {
	Entries    []CommEntry `json:"entries"`
	TotalCount int         `json:"total_count"`
}
