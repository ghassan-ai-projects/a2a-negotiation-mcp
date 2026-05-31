package autocomplete

// CompletionScript holds the generated shell completion script content
// and the target shell name.
type CompletionScript struct {
	Content string `json:"content"`
	Shell   string `json:"shell"`
}
