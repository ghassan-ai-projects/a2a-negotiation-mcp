package batchnegotiation

// BatchRequest is the input for a batch negotiation.
type BatchRequest struct {
	Items []BatchItem `json:"items"`
}

// BatchItem is a single item in a batch negotiation.
type BatchItem struct {
	Vendor   string  `json:"vendor"`
	SKU      string  `json:"sku"`
	Strategy string  `json:"strategy"`
	Budget   float64 `json:"budget"`
}

// BatchResult is the output of a batch negotiation.
type BatchResult struct {
	BatchID      string            `json:"batch_id"`
	Results      []BatchItemResult `json:"results"`
	TotalSavings float64           `json:"total_savings"`
	DurationMs   int64             `json:"duration_ms"`
	CreatedAt    string            `json:"created_at"`
}

// BatchItemResult is the result of a single item in a batch.
type BatchItemResult struct {
	Vendor    string  `json:"vendor"`
	SessionID string  `json:"session_id"`
	Status    string  `json:"status"`
	Savings   float64 `json:"savings"`
}
