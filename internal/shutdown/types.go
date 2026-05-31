package shutdown

// ShutdownResult holds the outcome of a graceful shutdown operation.
type ShutdownResult struct {
	Status           string   `json:"status"`
	ResourcesCleaned []string `json:"resources_cleaned"`
	DurationMs       int64    `json:"duration_ms"`
}
