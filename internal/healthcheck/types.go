package healthcheck

// HealthResult represents the result of a health check.
type HealthResult struct {
	Status      string `json:"status"`
	DatabaseOK  bool   `json:"database_ok"`
	ToolCount   int    `json:"tool_count"`
	DBSizeBytes int64  `json:"db_size_bytes"`
	UptimeSecs  int64  `json:"uptime_seconds"`
	StartedAt   string `json:"started_at"`
}
