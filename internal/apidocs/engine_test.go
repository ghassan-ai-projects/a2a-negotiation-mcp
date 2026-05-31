package apidocs

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func newTestServer() (*mcpserver.MCPServer, *Engine) {
	s := mcpserver.NewMCPServer("test", "1.0.0", mcpserver.WithToolCapabilities(true))
	s.AddTool(mcp.NewTool("test_tool_one",
		mcp.WithDescription("First test tool"),
		mcp.WithString("name", mcp.Required(), mcp.Description("A name parameter")),
		mcp.WithNumber("count", mcp.Description("A count parameter")),
	), nil)
	s.AddTool(mcp.NewTool("test_tool_two",
		mcp.WithDescription("Second test tool"),
		mcp.WithString("vendor", mcp.Required(), mcp.Description("Vendor name")),
	), nil)
	s.AddTool(mcp.NewTool("test_tool_no_params",
		mcp.WithDescription("Tool with no parameters"),
	), nil)
	e := NewEngine(s)
	return s, e
}

func TestGenerateMarkdown(t *testing.T) {
	_, e := newTestServer()
	md := e.GenerateMarkdown()

	if !strings.Contains(md, "API Documentation") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "test_tool_one") {
		t.Error("markdown should contain test_tool_one")
	}
	if !strings.Contains(md, "test_tool_two") {
		t.Error("markdown should contain test_tool_two")
	}
	if !strings.Contains(md, "test_tool_no_params") {
		t.Error("markdown should contain test_tool_no_params")
	}
	if !strings.Contains(md, "First test tool") {
		t.Error("markdown should contain description of test_tool_one")
	}
	if !strings.Contains(md, "| `name` |") {
		t.Error("markdown should contain name parameter in table")
	}
	if !strings.Contains(md, "| `count` |") {
		t.Error("markdown should contain count parameter in table")
	}
	if !strings.Contains(md, "Total tools: **3**") {
		t.Error("markdown should show correct tool count")
	}
	if !strings.Contains(md, "No parameters") {
		t.Error("markdown should indicate tools with no parameters")
	}
}

func TestGenerateJSON(t *testing.T) {
	_, e := newTestServer()
	data, err := e.GenerateJSON()
	if err != nil {
		t.Fatalf("GenerateJSON failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "test_tool_one") {
		t.Error("JSON should contain test_tool_one")
	}
	if !strings.Contains(jsonStr, "test_tool_two") {
		t.Error("JSON should contain test_tool_two")
	}
	if !strings.Contains(jsonStr, "First test tool") {
		t.Error("JSON should contain description")
	}
	if !strings.Contains(jsonStr, `"tool_count": 3`) {
		t.Error("JSON should have tool_count of 3")
	}
}
