package contracttemplates

// ContractTemplate is a reusable contract template stored in SQLite.
type ContractTemplate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// GeneratedContract is the result of rendering a template with vendor and custom parameters.
type GeneratedContract struct {
	TemplateID    string   `json:"template_id"`
	VendorName    string   `json:"vendor_name"`
	Content       string   `json:"content"`
	VariablesUsed []string `json:"variables_used"`
}
