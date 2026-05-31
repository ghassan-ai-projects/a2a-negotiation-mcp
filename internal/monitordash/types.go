package monitordash

type LiveDashboard struct {
	ActiveNegotiations int            `json:"active_negotiations"`
	SystemHealth       string         `json:"system_health"`
	LastToolCalls      []ToolCallEntry `json:"last_tool_calls"`
	ErrorRate5Min      float64        `json:"error_rate_5min"`
	UptimeSeconds      int64          `json:"uptime_seconds"`
	TotalTools         int            `json:"total_tools"`
	Timestamp          string         `json:"timestamp"`
}

type ToolCallEntry struct {
	ToolName   string `json:"tool_name"`
	DurationMs int    `json:"duration_ms"`
	Success    bool   `json:"success"`
	Timestamp  string `json:"timestamp"`
}
