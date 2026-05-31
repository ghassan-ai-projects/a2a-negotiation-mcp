package vendorknowledge

// KnowledgeDoc represents a stored vendor knowledge document.
type KnowledgeDoc struct {
	ID        int    `json:"id"`
	Vendor    string `json:"vendor"`
	Content   string `json:"content"`
	DocType   string `json:"doc_type"`
	CreatedAt string `json:"created_at"`
}
