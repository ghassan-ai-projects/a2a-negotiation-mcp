package apidocs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Engine generates API documentation from an MCP server's registered tools.
type Engine struct {
	mcpServer *mcpserver.MCPServer
}

// NewEngine creates a new apidocs engine.
func NewEngine(mcpServer *mcpserver.MCPServer) *Engine {
	return &Engine{mcpServer: mcpServer}
}

// collectDocs enumerates all tools registered on the MCP server and returns
// an APIDoc.
func (e *Engine) collectDocs() *APIDoc {
	toolMap := e.mcpServer.ListTools()
	names := make([]string, 0, len(toolMap))
	for name := range toolMap {
		names = append(names, name)
	}
	sort.Strings(names)

	docs := make([]ToolDoc, 0, len(names))
	for _, name := range names {
		st := toolMap[name]
		t := st.Tool

		params := make([]ParamDoc, 0)
		requiredSet := make(map[string]bool)
		for _, r := range t.InputSchema.Required {
			requiredSet[r] = true
		}
		propNames := make([]string, 0, len(t.InputSchema.Properties))
		for pn := range t.InputSchema.Properties {
			propNames = append(propNames, pn)
		}
		sort.Strings(propNames)

		for _, pn := range propNames {
			raw, _ := t.InputSchema.Properties[pn].(map[string]any)
			pt := ""
			if raw != nil {
				if tv, ok := raw["type"]; ok {
					pt = fmt.Sprintf("%v", tv)
				}
			}
			desc := ""
			if raw != nil {
				if dv, ok := raw["description"]; ok {
					desc = fmt.Sprintf("%v", dv)
				}
			}
			params = append(params, ParamDoc{
				Name:        pn,
				Required:    requiredSet[pn],
				ParamType:   pt,
				Description: desc,
			})
		}

		docs = append(docs, ToolDoc{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}

	return &APIDoc{
		Tools:     docs,
		ToolCount: len(docs),
	}
}

// GenerateMarkdown returns a human-readable Markdown string documenting all
// registered tools.
func (e *Engine) GenerateMarkdown() string {
	doc := e.collectDocs()

	var b strings.Builder
	b.WriteString("# API Documentation\n\n")
	b.WriteString(fmt.Sprintf("Total tools: **%d**\n\n", doc.ToolCount))

	for _, tool := range doc.Tools {
		b.WriteString(fmt.Sprintf("## %s\n\n", tool.Name))
		if tool.Description != "" {
			b.WriteString(fmt.Sprintf("%s\n\n", tool.Description))
		}
		if len(tool.Parameters) == 0 {
			b.WriteString("_No parameters._\n\n")
			continue
		}
		b.WriteString("| Parameter | Type | Required | Description |\n")
		b.WriteString("|-----------|------|----------|-------------|\n")
		for _, p := range tool.Parameters {
			req := "No"
			if p.Required {
				req = "Yes"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s |\n", p.Name, p.ParamType, req, p.Description))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// GenerateJSON returns a JSON representation of the API documentation.
func (e *Engine) GenerateJSON() ([]byte, error) {
	doc := e.collectDocs()
	return json.MarshalIndent(doc, "", "  ")
}
