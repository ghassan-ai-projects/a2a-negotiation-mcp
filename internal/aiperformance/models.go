package aiperformance

// AIPerformanceLog represents a single AI agent performance log entry.
type AIPerformanceLog struct {
	ID              int    `json:"id"`
	Model           string `json:"model"`
	ToolName        string `json:"tool_name"`
	LatencyMs       int    `json:"latency_ms"`
	TokensUsed      int    `json:"tokens_used"`
	Success         bool   `json:"success"`
	NegotiationType string `json:"negotiation_type"`
	CreatedAt       string `json:"created_at"`
}

// ProviderSummary represents aggregated performance metrics by model.
type ProviderSummary struct {
	Model        string  `json:"model"`
	TotalCalls   int     `json:"total_calls"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	TotalTokens  int     `json:"total_tokens"`
	AvgCost      float64 `json:"avg_cost"`
}
