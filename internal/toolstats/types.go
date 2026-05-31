package toolstats

// ToolUsage represents usage counts for a single tool.
type ToolUsage struct {
	ToolName  string `json:"tool_name"`
	CallCount int    `json:"call_count"`
}

// UsageReport summarizes tool usage over a period.
type UsageReport struct {
	Period      string      `json:"period"`
	TotalCalls  int         `json:"total_calls"`
	UniqueTools int         `json:"unique_tools"`
	TopTools    []ToolUsage `json:"top_tools"`
	BottomTools []ToolUsage `json:"bottom_tools"`
}
