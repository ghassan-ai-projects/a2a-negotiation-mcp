package a2a

// TaskRequest is the payload for POST /a2a/task.
type TaskRequest struct {
	Task   string         `json:"task"`
	Params map[string]any `json:"params,omitempty"`
}

// TaskResponse is the response from POST /a2a/task and GET /a2a/task/{id}.
type TaskResponse struct {
	TaskID    string         `json:"task_id"`
	Status    string         `json:"status"`
	SessionID string         `json:"session_id,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// NegotiateRequest is the payload for POST /a2a/negotiate.
type NegotiateRequest struct {
	Vendor   string         `json:"vendor"`
	SKU      string         `json:"sku,omitempty"`
	Strategy string         `json:"strategy"`
	Budget   float64        `json:"budget,omitempty"`
	Terms    map[string]any `json:"terms,omitempty"`
}

// NegotiateResponse is the response from POST /a2a/negotiate.
type NegotiateResponse struct {
	MandateID  string         `json:"mandate_id"`
	TaskID     string         `json:"task_id"`
	SessionID  string         `json:"session_id"`
	Status     string         `json:"status"`
	Offer      float64        `json:"offer"`
	ListPrice  float64        `json:"list_price"`
	Strategy   string         `json:"strategy"`
	Mandate    *Mandate       `json:"mandate,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
}
