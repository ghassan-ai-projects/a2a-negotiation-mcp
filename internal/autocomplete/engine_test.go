package autocomplete_test

import (
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/autocomplete"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func newTestEngine(t *testing.T) *autocomplete.Engine {
	t.Helper()
	srv := mcpserver.NewMCPServer("test", "1.0.0", mcpserver.WithToolCapabilities(true))
	srv.AddTool(mcp.NewTool("negotiate_query_price",
		mcp.WithDescription("test"),
	), nil)
	srv.AddTool(mcp.NewTool("negotiate_create_session",
		mcp.WithDescription("test"),
	), nil)
	srv.AddTool(mcp.NewTool("negotiate_history",
		mcp.WithDescription("test"),
	), nil)
	return autocomplete.NewEngine(srv)
}

func TestGenerateBash(t *testing.T) {
	eng := newTestEngine(t)
	script := eng.Generate("bash")

	if script.Shell != "bash" {
		t.Errorf("expected shell 'bash', got %q", script.Shell)
	}
	if script.Content == "" {
		t.Fatal("expected non-empty bash completion script")
	}
	if !contains(script.Content, "#!/bin/bash") {
		t.Error("expected bash shebang")
	}
	if !contains(script.Content, "complete -F _a2a_cli a2a-cli") {
		t.Error("expected complete directive")
	}
	if !contains(script.Content, "negotiate_query_price") {
		t.Error("expected tool name in completion list")
	}
	if !contains(script.Content, "negotiate_create_session") {
		t.Error("expected tool name in completion list")
	}
	if !contains(script.Content, "negotiate_history") {
		t.Error("expected tool name in completion list")
	}
}

func TestGenerateZSH(t *testing.T) {
	eng := newTestEngine(t)
	script := eng.Generate("zsh")

	if script.Shell != "zsh" {
		t.Errorf("expected shell 'zsh', got %q", script.Shell)
	}
	if script.Content == "" {
		t.Fatal("expected non-empty zsh completion script")
	}
	if !contains(script.Content, "#compdef a2a-cli") {
		t.Error("expected zsh compdef directive")
	}
	if !contains(script.Content, "negotiate_query_price") {
		t.Error("expected tool name in completion list")
	}
	if !contains(script.Content, "negotiate_create_session") {
		t.Error("expected tool name in completion list")
	}
}

func TestGenerateDefaultShell(t *testing.T) {
	eng := newTestEngine(t)
	script := eng.Generate("unknown")

	// Should default to bash
	if script.Shell != "bash" {
		t.Errorf("expected default shell 'bash', got %q", script.Shell)
	}
	if !contains(script.Content, "complete -F _a2a_cli a2a-cli") {
		t.Error("expected bash completion directive")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
