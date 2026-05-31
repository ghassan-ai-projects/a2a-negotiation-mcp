package prompts

// PromptTemplate represents a stored negotiation prompt template.
type PromptTemplate struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"created_at"`
}
