package dependency

// DepEntry represents a single dependency entry.
type DepEntry struct {
	Module    string `json:"module"`
	Version   string `json:"version"`
	GoVersion string `json:"go_version,omitempty"`
}

// DependencyReport is the top-level report returned by Parse().
type DependencyReport struct {
	Direct     []DepEntry `json:"direct"`
	Indirect   []DepEntry `json:"indirect"`
	TotalCount int        `json:"total_count"`
}
