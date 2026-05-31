package contribguide

import (
	"os"
	"path/filepath"
	"strings"
)

// Engine generates CONTRIBUTING.md based on project structure.
type Engine struct{}

// NewEngine creates a new contribguide Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Generate produces a ContributionGuide by inspecting the project at basePath.
func (e *Engine) Generate(basePath string) (*ContributionGuide, error) {
	sections := e.discoverSections(basePath)
	content := e.buildMarkdown(basePath, sections)
	return &ContributionGuide{
		Content:  content,
		Sections: sections,
	}, nil
}

// discoverSections walks the project to identify structural patterns.
func (e *Engine) discoverSections(basePath string) []string {
	var sections []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return []string{"Development Setup", "Building", "Testing", "Package Conventions", "MCP Tool Registration", "PR Workflow"}
	}

	hasGoMod := false
	hasMakefile := false
	hasDockerfile := false
	hasInternal := false

	for _, entry := range entries {
		switch entry.Name() {
		case "go.mod":
			hasGoMod = true
		case "Makefile", "makefile":
			hasMakefile = true
		case "Dockerfile":
			hasDockerfile = true
		}
		if entry.IsDir() && entry.Name() == "internal" {
			hasInternal = true
		}
	}

	if hasGoMod {
		sections = append(sections, "Development Setup")
	}
	if hasGoMod {
		sections = append(sections, "Building")
	}
	if hasGoMod || hasMakefile {
		sections = append(sections, "Testing")
	}
	if hasInternal {
		sections = append(sections, "Package Conventions")
	}
	sections = append(sections, "MCP Tool Registration")
	sections = append(sections, "PR Workflow")

	if hasDockerfile {
		sections = append(sections, "Docker")
	}

	return sections
}

// buildMarkdown composes the CONTRIBUTING.md content.
func (e *Engine) buildMarkdown(basePath string, sections []string) string {
	var b strings.Builder

	b.WriteString("# Contributing\n\n")
	b.WriteString("Thank you for your interest in contributing to this project!\n\n")

	sectionMap := map[string]string{
		"Development Setup":     e.developmentSetupSection(basePath),
		"Building":              e.buildingSection(basePath),
		"Testing":               e.testingSection(basePath),
		"Package Conventions":   e.packageConventionsSection(basePath),
		"MCP Tool Registration": e.mcpToolSection(basePath),
		"PR Workflow":           e.prWorkflowSection(basePath),
		"Docker":                e.dockerSection(basePath),
	}

	for _, s := range sections {
		if content, ok := sectionMap[s]; ok {
			b.WriteString(content)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (e *Engine) developmentSetupSection(basePath string) string {
	return `## Development Setup

### Prerequisites

- **Go 1.26** or later
- **SQLite** (bundled via modernc.org/sqlite — no CGO required)

### Getting Started

1. Clone the repository:
   ` + "```" + `bash
   git clone <repository-url>
   cd ` + filepath.Base(basePath) + `
   ` + "```" + `

2. Download dependencies:
   ` + "```" + `bash
   go mod download
   ` + "```" + `
`
}

func (e *Engine) buildingSection(basePath string) string {
	return `## Building

Build the server binary:

` + "```" + `bash
go build ./cmd/server
` + "```" + `

The resulting binary can be run directly:
` + "```" + `bash
./server -db ./data/negotiation.db
` + "```" + `
`
}

func (e *Engine) testingSection(basePath string) string {
	return `## Testing

Run all tests:

` + "```" + `bash
go test ./...
` + "```" + `

Run tests with race detection:
` + "```" + `bash
go test -race ./...
` + "```" + `

Run tests with coverage:
` + "```" + `bash
go test -cover ./...
` + "```" + `
`
}

func (e *Engine) packageConventionsSection(basePath string) string {
	return `## Package Conventions

Each internal package follows a consistent file layout:

| File | Purpose |
|------|---------|
| ` + "`types.go`" + ` | Data structures, request/response types |
| ` + "`engine.go`" + ` | Business logic (pure Go, no MCP dependency) |
| ` + "`store.go`" + ` | SQLite persistence layer (if applicable) |
| ` + "`engine_test.go`" + ` | Unit tests for the engine |
| ` + "`store_test.go`" + ` | Unit tests for the store (if applicable) |

### Naming Conventions

- Package names are lowercase, single word (e.g. ` + "`coverage`" + `, ` + "`dependency`" + `)
- Test functions use ` + "`TestFunctionName`" + ` pattern
- Constructors use ` + "`NewEngine`" + ` or ` + "`NewStore`" + `
`
}

func (e *Engine) mcpToolSection(basePath string) string {
	return `## MCP Tool Registration Pattern

Tools are registered in ` + "`internal/server/tools.go`" + `:

1. Add the engine field to the ` + "`NegotiationServer`" + ` struct
2. Add the parameter to ` + "`NewNegotiationServer`" + ` constructor
3. Register the tool with ` + "`mcpServer.AddTool(mcp.NewTool(...))" + ` in ` + "`registerTools()`" + `
4. Implement the handler method on ` + "`NegotiationServer`" + `

### Handler Pattern

` + "```" + `go
func (ns *NegotiationServer) handleToolName(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()
    param, _ := req.RequireString("param")
    ns.logger.Debug("negotiate_toolname called", "param", param)
    result, err := ns.someEng.SomeMethod(ctx, param)
    if err != nil {
        ns.logger.Warn("negotiate_toolname failed", "error", err.Error())
        return mcp.NewToolResultError("Failed: " + err.Error()), nil
    }
    resp := map[string]any{
        "field":  result.Field,
        "duration_ms": time.Since(start).Milliseconds(),
    }
    return ns.jsonResult(resp)
}
` + "```" + `
`
}

func (e *Engine) prWorkflowSection(basePath string) string {
	return `## PR Workflow Guidelines

1. **Branch naming**: Use ` + "`feature/description`" + ` or ` + "`fix/description`" + `
2. **Commit messages**: Reference issues (e.g. "Implement coverage report tool (#68)")
3. **Before submitting**:
   - Run ` + "`go build ./...`" + ` — must compile cleanly
   - Run ` + "`go test ./...`" + ` — all tests must pass
   - Ensure no new lint warnings
4. **Review**: At least one maintainer review required before merge
5. **Squash merge**: Commits should be squashed into a single logical commit per feature
`
}

func (e *Engine) dockerSection(basePath string) string {
	return `## Docker

Build the Docker image:

` + "```" + `bash
docker build -t a2a-negotiation-mcp .
` + "```" + `

Run with Docker Compose:
` + "```" + `bash
docker-compose up
` + "```" + `
`
}
