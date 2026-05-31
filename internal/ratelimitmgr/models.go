package ratelimitmgr

type RateLimitConfig struct {
	ToolName      string `json:"tool_name"`
	MaxCalls      int    `json:"max_calls"`
	WindowSeconds int    `json:"window_seconds"`
	UpdatedAt     string `json:"updated_at"`
}

type RateLimitHit struct {
	ID        int    `json:"id"`
	ToolName  string `json:"tool_name"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}
