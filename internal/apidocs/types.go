package apidocs

// ParamDoc describes a single parameter for a tool.
type ParamDoc struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	ParamType   string `json:"param_type"`
	Description string `json:"description"`
}

// ToolDoc describes a registered MCP tool.
type ToolDoc struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  []ParamDoc `json:"parameters"`
}

// APIDoc contains documentation for all registered tools.
type APIDoc struct {
	Tools     []ToolDoc `json:"tools"`
	ToolCount int       `json:"tool_count"`
}
