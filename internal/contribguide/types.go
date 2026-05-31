package contribguide

// ContributionGuide represents a generated CONTRIBUTING.md document.
type ContributionGuide struct {
	Content  string   `json:"content"`
	Sections []string `json:"sections"`
}
