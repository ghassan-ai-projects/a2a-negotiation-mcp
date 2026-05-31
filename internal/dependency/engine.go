package dependency

import (
	"bufio"
	"strings"
)

// Engine parses go.mod to produce dependency reports.
type Engine struct{}

// NewEngine creates a new dependency Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Parse parses go.mod content and returns a DependencyReport.
func (e *Engine) Parse(data []byte) (*DependencyReport, error) {
	var direct []DepEntry
	var indirect []DepEntry

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect the require block
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		// Single-line require e.g. require dep v1.0.0
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			entry := parseRequireLine(strings.TrimPrefix(line, "require "))
			if entry != nil {
				direct = append(direct, *entry)
			}
			continue
		}

		if inRequire {
			entry := parseRequireLine(line)
			if entry != nil {
				if strings.Contains(line, "// indirect") {
					indirect = append(indirect, *entry)
				} else {
					direct = append(direct, *entry)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	total := len(direct) + len(indirect)

	return &DependencyReport{
		Direct:     direct,
		Indirect:   indirect,
		TotalCount: total,
	}, nil
}

// parseRequireLine extracts module and version from a require line.
// Input examples:
//   github.com/google/uuid v1.6.0
//   github.com/mark3labs/mcp-go v0.54.1 // indirect
func parseRequireLine(line string) *DepEntry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	module := parts[0]
	version := parts[1]

	// Check for go version directive e.g. "go 1.26.3"
	if module == "go" && len(parts) == 2 {
		return nil
	}

	return &DepEntry{
		Module:  module,
		Version: version,
	}
}
