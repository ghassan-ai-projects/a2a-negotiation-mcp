package workflow

// Workflow represents a custom automation workflow with conditional logic.
type Workflow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StepsJSON string `json:"steps_json"`
	CreatedAt string `json:"created_at"`
}

// WorkflowStep represents a single step within a workflow.
type WorkflowStep struct {
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Params    string `json:"params"`
}

// WorkflowLogEntry represents a single execution log for a workflow.
type WorkflowLogEntry struct {
	ID         int    `json:"id"`
	WorkflowID int    `json:"workflow_id"`
	RunAt      string `json:"run_at"`
	Status     string `json:"status"`
	Result     string `json:"result"`
}
