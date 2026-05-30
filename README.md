# A2A Negotiation MCP Server

Headless MCP server for A2A-compliant agent-to-agent SaaS pricing negotiation.

## Overview

This MCP server provides pricing intelligence and negotiation capabilities that any AI agent can use via the MCP protocol. It registers 6 tools and 4 resources for discovering fair market prices, calculating savings opportunities, running multi-round negotiations, and tracking deal history — all via stdio MCP transport.

## MCP Tools

| Tool | Description |
|------|-------------|
| `negotiate_query_price` | Query fair market price range for a SaaS vendor's product |
| `negotiate_calculate_savings` | Estimate potential savings based on current spend |
| `negotiate_create_session` | Start a new negotiation session with a strategy profile |
| `negotiate_run` | Execute the multi-round negotiation loop |
| `negotiate_history` | View negotiation history and performance metrics |
| `negotiate_strategies` | List available negotiation strategy profiles |

## MCP Resources

| URI | Description |
|-----|-------------|
| `negotiate://pricing/{vendor}/{sku}` | Current market pricing for a specific product |
| `negotiate://session/{session_id}` | Full negotiation history, all rounds |
| `negotiate://history/{period}` | Aggregated performance stats (30d, 90d, 1y, all) |
| `negotiate://strategies` | Available strategies and descriptions |

## Strategy Profiles

- **Aggressive** — Start at 30% below asking, concede slowly, 4-5 rounds
- **Balanced** — Start at 20% below asking, 3-4 rounds, moderate concessions
- **Conservative** — Start at 10% below asking, 2-3 rounds, quick to accept

## Usage

```bash
# Build
go build -o bin/a2a-negotiation ./cmd/server

# Run with seed data (stdio MCP transport)
./bin/a2a-negotiation -seed data/seeds/saas_pricing.csv

# Run with custom DB path
./bin/a2a-negotiation -db /path/to/custom.db -seed data/seeds/saas_pricing.csv

# Text logging (default: JSON)
./bin/a2a-negotiation -log text

# Connect via MCP client (example using mcp-cli)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./bin/a2a-negotiation
```

### Configuring with Claude Code / Cline

Add to your MCP settings:

```json
{
  "mcpServers": {
    "a2a-negotiation": {
      "command": "/path/to/a2a-negotiation-mcp/bin/a2a-negotiation",
      "args": ["-seed", "/path/to/a2a-negotiation-mcp/data/seeds/saas_pricing.csv"],
      "env": {}
    }
  }
}
```

## Seed Data

Includes ~50 common SaaS products with known pricing ranges: Slack, Notion, GitHub, Salesforce, Zoom, Figma, Atlassian, Datadog, AWS, Google Cloud, Microsoft 365, Jira, Confluence, Sentry, PagerDuty, Monday.com.

## Architecture

```
cmd/server/main.go          — Entry point, MCP server setup
internal/
  server/tools.go           — MCP tool handlers (6 tools)
  server/resources.go       — MCP resource handlers (4 resources)
  pricing/db.go             — SQLite pricing DB operations
  pricing/models.go         — Pricing data models
  negotiation/engine.go     — Core negotiation strategy engine
  negotiation/strategies.go — Strategy profiles
  negotiation/templates.go  — Counter-offer templates
  history/store.go          — SQLite sessions + history
  history/models.go         — History data models
ierrors/errors.go           — Domain-specific error types
data/seeds/saas_pricing.csv — Initial public SaaS pricing data
```

## Tech Stack

- **Go 1.26** — Modern Go with slog for structured JSON logging
- **modernc.org/sqlite** — CGo-free SQLite for storage
- **github.com/mark3labs/mcp-go** — MCP protocol implementation

## Tests

```bash
go test ./... -v -count=1
```

Covers: price query, session creation, negotiation loop (accept/budget/walk-away), history CRUD, edge cases (empty DB, unknown vendor, concurrent sessions).
