package playbook

// PlaybookSection represents a named section of items in a negotiation playbook.
type PlaybookSection struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

// Playbook is the complete negotiation playbook with markdown content and structured sections.
type Playbook struct {
	Content  string            `json:"content"`
	Sections []PlaybookSection `json:"sections"`
}
