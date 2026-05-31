package sandbox

// SandboxExecution represents a single tool execution attempt in the sandbox.
type SandboxExecution struct {
	ID        int    `json:"id"`
	ToolName  string `json:"tool_name"`
	Params    string `json:"params"`
	Result    string `json:"result"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// SandboxTemplate represents a predefined template for trying a tool.
type SandboxTemplate struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ToolName      string `json:"tool_name"`
	ExampleParams string `json:"example_params"`
}
