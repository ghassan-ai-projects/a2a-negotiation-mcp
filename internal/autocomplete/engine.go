package autocomplete

import (
	"fmt"
	"sort"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Engine generates shell completion scripts for registered MCP tools.
type Engine struct {
	mcpServer *mcpserver.MCPServer
}

// NewEngine creates an autocomplete engine backed by an MCPServer.
func NewEngine(mcpServer *mcpserver.MCPServer) *Engine {
	return &Engine{mcpServer: mcpServer}
}

// Generate produces a CompletionScript for the given shell ("bash" or "zsh").
func (e *Engine) Generate(shell string) CompletionScript {
	tools := e.mcpServer.ListTools()
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)

	switch strings.ToLower(shell) {
	case "zsh":
		return e.generateZSH(names)
	default:
		return e.generateBash(names)
	}
}

func (e *Engine) generateBash(toolNames []string) CompletionScript {
	var b strings.Builder

	b.WriteString("#!/bin/bash\n\n")
	b.WriteString("_a2a_cli() {\n")
	b.WriteString("\tlocal cur prev opts\n")
	b.WriteString("\tCOMPREPLY=()\n")
	b.WriteString("\tcur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("\tprev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	b.WriteString("\topts=\"")
	for i, name := range toolNames {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(name)
	}
	b.WriteString("\"\n\n")
	b.WriteString("\tif [[ ${cur} == * ]] ; then\n")
	b.WriteString("\t\tCOMPREPLY=( $(compgen -W \"${opts}\" -- ${cur}) )\n")
	b.WriteString("\t\treturn 0\n")
	b.WriteString("\tfi\n")
	b.WriteString("}\n\n")
	b.WriteString("complete -F _a2a_cli a2a-cli\n")

	return CompletionScript{
		Content: b.String(),
		Shell:   "bash",
	}
}

func (e *Engine) generateZSH(toolNames []string) CompletionScript {
	var b strings.Builder

	b.WriteString("#compdef a2a-cli\n\n")
	b.WriteString("local -a _1st_arguments\n")
	b.WriteString("_1st_arguments=(\n")
	for _, name := range toolNames {
		b.WriteString(fmt.Sprintf("\t%s\n", name))
	}
	b.WriteString(")\n\n")
	b.WriteString("_arguments \\\n")
	b.WriteString("\t'1: :->command' \\\n")
	b.WriteString("\t'*:: :->args'\n\n")
	b.WriteString("case $state in\n")
	b.WriteString("\tcommand)\n")
	b.WriteString("\t\t_describe -t commands \"a2a-cli commands\" _1st_arguments\n")
	b.WriteString("\t\t;;\n")
	b.WriteString("esac\n")

	return CompletionScript{
		Content: b.String(),
		Shell:   "zsh",
	}
}
