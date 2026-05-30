# A2A Negotiation MCP Server

**Agent-to-Agent SaaS pricing negotiation server** — an MCP-compliant server that gives AI agents the ability to query market pricing, run multi-round negotiations with strategy profiles, manage buying groups, track contracts and renewals, analyze vendor quotes, manage SLA breaches, and broker unused-seat marketplace transactions.

Serves **34 MCP tools** and **5 MCP resources** over stdio (MCP protocol) and optionally exposes A2A HTTP endpoints for agent-to-agent communication.

---

## Quick Start

### Build

```bash
go build -o bin/a2a-negotiation ./cmd/server
```

### Run with seed data

```bash
./bin/a2a-negotiation -seed data/seeds/saas_pricing.csv
```

Starts the MCP server on stdio, ready to accept connections from any MCP client.

### Test with the CLI

```bash
# Terminal 1: start server with HTTP
./bin/a2a-negotiation -http :8080 -seed data/seeds/saas_pricing.csv

# Terminal 2: use the CLI client
go run ./cmd/cli -- query Slack
go run ./cmd/cli -- negotiate Slack Pro --strategy balanced --budget 7
go run ./cmd/cli -- discover
go run ./cmd/cli -- health Salesforce
go run ./cmd/cli -- sla add Acme CRM --uptime 99.9 --credit 10 --max-credit 25 --spend 5000
go run ./cmd/cli -- strategies
```

### Run all tests

```bash
go test ./... -v -count=1
```

---

## MCP Tools Reference

All 34 tools are registered via the MCP protocol and accessible from any MCP client over stdio. (`*` = required parameter)

### Pricing & Discovery

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_query_price` | Query fair market price range for a SaaS vendor's product | `vendor`*, `sku`, `quantity`, `term_months` |
| `negotiate_calculate_savings` | Estimate potential savings based on current spend and market data | `vendor`*, `current_spend`* |
| `negotiate_strategies` | List available negotiation strategy profiles | *(none)* |
| `negotiate_strategy_recommend` | Get the best strategy recommendation based on past outcomes | `vendor`* |
| `negotiate_learning_insights` | Get global learning insights across all vendors | *(none)* |

### Negotiation Sessions

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_create_session` | Start a new negotiation session with a strategy profile | `vendor`*, `sku`, `strategy`*, `budget`, `constraints` |
| `negotiate_run` | Execute the multi-round negotiation loop | `session_id`*, `auto_approve_threshold` |
| `negotiate_run_parallel` | Run parallel negotiations across multiple sessions | `sessions`*, `strategy`*, `timeout` |
| `negotiate_history` | View negotiation history and performance metrics | `vendor`, `period` |

### Buying Groups

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_create_group` | Create a collective buying group | `target_vendor`*, `target_sku`*, `min_members`, `expires_in_hours` |
| `negotiate_join_group` | Join an existing buying group | `group_id`*, `user_id`*, `quantity`, `max_price` |
| `negotiate_compute_offer` | Compute a consolidated group offer | `group_id`* |
| `negotiate_group_status` | View group details and member list | `group_id`* |

### Contract & Renewal Management

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_add_contract` | Register a new SaaS contract for renewal tracking | `vendor`*, `sku`*, `seats`*, `current_price_per_unit`*, `renewal_date`* |
| `negotiate_list_contracts` | List registered contracts with optional filters | `vendor`, `status`, `expiring_soon` |
| `negotiate_check_renewals` | Check upcoming renewals with urgency classification | `days_ahead`* |
| `negotiate_trigger_renewal` | Trigger automatic negotiation for a renewing contract | `contract_id`* |

### Used-Seat Marketplace

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_list_unused_seats` | List unused SaaS seats for sale | `vendor`*, `sku`*, `seats`*, `orig_price`*, `ask_price`*, `min_price`, `expires_in_hours` |
| `negotiate_search_used` | Search for unused seat listings | `vendor`*, `sku`, `max_seats` |
| `negotiate_offer_seats` | Place a buy offer on a used-seats listing | `listing_id`*, `buyer_id`*, `seats`*, `max_price`* |
| `negotiate_accept_offer` | Accept a pending offer (5% platform fee) | `listing_id`*, `offer_id`* |
| `negotiate_marketplace_overview` | Get marketplace: active listings, recent transactions | *(none)* |

### Quote Analysis

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_analyze_quote` | Analyze a vendor quote email, extract fields, cross-reference market data | `raw_text`*, `vendor`, `sku` |
| `negotiate_generate_counter` | Generate formatted counter-offer text from analysis JSON | `analysis_json`* |

### Contract Parsing

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_parse_contract` | Parse contract text for key terms with per-field confidence | `raw_text`*, `vendor`, `sku` |
| `negotiate_parse_and_calendar` | Parse contract AND auto-populate renewal calendar | `raw_text`*, `vendor`, `sku` |

### SLA Management

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_add_sla` | Register an SLA contract | `vendor`*, `service`*, `uptime_pct`*, `credit_pct`*, `max_credit_pct`*, `monthly_spend`* |
| `negotiate_record_breach` | Record an SLA breach | `vendor`*, `service`*, `date`*, `duration_mins`* |
| `negotiate_file_claim` | File an SLA breach claim for credit | `breach_id`* |
| `negotiate_sla_report` | Get SLA report for a given month | `month`* |

### Vendor Health

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_vendor_health` | Get vendor health and leverage assessment | `vendor`* |
| `negotiate_record_signal` | Record a health signal for a vendor | `vendor`*, `type`*, `detail`*, `weight` |
| `negotiate_health_overview` | Get health overview of all tracked vendors | *(none)* |
| `negotiate_discover_opportunities` | Discover negotiation opportunities by industry | `industry` |

### Slack Integration

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_configure_slack` | Configure Slack webhook URL for alerts | `webhook_url`* |
| `negotiate_slack_status` | Check Slack integration status | *(none)* |

### Authentication & Rate Limiting (require `-api-keys` flag)

| Tool | Description | Parameters |
|------|-------------|------------|
| `negotiate_generate_api_key` | Generate a new API key (returned exactly once) | `owner`* |
| `negotiate_rate_limit_status` | Check API key count and rate limit configuration | *(none)* |

---

## MCP Resources

| URI | Description |
|-----|-------------|
| `negotiate://pricing/{vendor}/{sku}` | Current market pricing data for a specific vendor product |
| `negotiate://session/{session_id}` | Full negotiation history and current state for a session |
| `negotiate://history/{period}` | Aggregated performance statistics (30d, 90d, 1y, all) |
| `negotiate://strategies` | List of all available negotiation strategy profiles |
| `negotiate://opportunities/{industry}` | Top negotiation opportunities for a given industry vertical |

---

## Architecture

### Package Map

```
cmd/server/                  Entry point, MCP stdio + A2A HTTP server setup
cmd/cli/                     CLI client (talks to A2A HTTP endpoints)

internal/
  a2a/                       A2A protocol: HTTP router, auth, rate limiting, agent card
  calendar/                  Renewal calendar engine + store
  contract/                  Contract text parsing + field extraction
  group/                     Collective buying group management
  health/                    Vendor health scoring + leverage assessment
  history/                   Negotiation session history + aggregate stats
  ierrors/                   Domain-specific error types
  learning/                  Cross-vendor strategy learning + insights
  marketplace/               Used-seat marketplace: listings, offers, transactions
  miner/                     Opportunity discovery engine
  negotiation/               Core negotiation engine, strategies, cultural profiles, templates
  parallel/                  Parallel session negotiation engine
  pricing/                   SQLite-backed pricing data store + models
  quote/                     Quote email analysis + counter-offer generation
  sell/                      Sell-side listing management
  server/                    MCP server: tool + resource registration and handlers
  sla/                       SLA contract management, breach tracking, claim filing
  slack/                     Slack Block Kit message builder + webhook client
  webhooks/                  Generic webhook dispatch engine

scripts/
  scrape-pricing.py          SaaS pricing web scraper

data/
  seeds/saas_pricing.csv     ~50 common SaaS products with known pricing ranges
```

### Package Dependency Graph

```
cmd/server/main.go
  |-- internal/server/          <- MCP tool/resource registration, all handler logic
  |   |-- internal/pricing/     <- SQLite CRUD for pricing data
  |   |-- internal/negotiation/ <- Strategy profiles, negotiation loop
  |   |-- internal/history/     <- Session + round persistence
  |   |-- internal/miner/       <- Opportunity discovery
  |   |-- internal/group/       <- Collective buying groups
  |   |-- internal/sell/        <- Sell-side listing management
  |   |-- internal/calendar/    <- Renewal calendar
  |   |-- internal/health/      <- Vendor health scores + leverage
  |   |-- internal/marketplace/ <- Used-seat marketplace
  |   |-- internal/sla/         <- SLA contracts + breach tracking
  |   |-- internal/learning/    <- Cross-vendor learning engine
  |   |-- internal/quote/       <- Quote analysis
  |   |-- internal/contract/    <- Contract parsing
  |   |-- internal/webhooks/    <- Webhook dispatch
  |   |-- internal/slack/       <- Slack notifications
  |-- internal/a2a/             <- A2A HTTP router, auth, rate limiting
```

### Transport Architecture

```
 MCP Client (Claude/Cline)        CLI Client         Other AI Agents
       |                              |                    |
       | stdio (MCP JSON-RPC)         | HTTP               | A2A Protocol
       v                              v                    v
+--------------+           +------------------+
|  MCP Server  |           |  A2A HTTP Server  |
|  (mark3labs)  |           |  :8080 (optional)  |
|  stdio only   |           +--------+---------+
+------+-------+                    |
       |                            |
       +----------+-----------------+
                  v
        NegotiationServer
     (tools.go, resources.go)
                  |
      +-----------+-----------+
      v           v           v
  pricing/   negotiation/   history/   ... 18 packages
```

---

## CLI Usage

Build: `go build -o bin/a2a-cli ./cmd/cli`

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-server` | `localhost:8080` | MCP/A2A server address |
| `-json` | `false` | Output raw JSON instead of formatted text |

### Commands

| Command | Description |
|---------|-------------|
| `query` | Query fair market price for a vendor/product |
| `negotiate` | Start and run a new negotiation session |
| `discover` | Discover negotiation opportunities |
| `health` | Get vendor health and leverage assessment |
| `sla` | Manage SLA contracts and breaches |
| `group` | Manage buying groups |
| `marketplace` | Browse the unused-seats marketplace |
| `contracts` | List and manage vendor contracts |
| `quote` | Analyze a vendor quote email |
| `strategies` | List available negotiation strategies |
| `help` | Show detailed help |

### Examples

```bash
# Query pricing
a2a-cli query Slack
a2a-cli query Slack Pro --seats 50 --term 12

# Run a negotiation
a2a-cli negotiate Slack Pro --seats 50 --strategy balanced --budget 10

# Discover opportunities
a2a-cli discover --industry fintech

# Vendor health
a2a-cli health Salesforce

# SLA management
a2a-cli sla add AcmeCRM Support --uptime 99.9 --credit 10 --max-credit 25 --spend 5000
a2a-cli sla breach AcmeCRM Support --duration 45
a2a-cli sla report --month 2026-06-01T00:00:00Z

# Buying groups
a2a-cli group create Slack --sku Pro --min-members 5
a2a-cli group join <group-id> --user alice --qty 20

# Marketplace
a2a-cli marketplace list
a2a-cli marketplace search Slack --sku Pro --max-seats 100

# Contracts
a2a-cli contracts list --vendor Slack --status active
a2a-cli contracts list --expiring 90

# Quote analysis
a2a-cli quote "Annual subscription: \$8.75/user/month for 100 users." --vendor Slack

# List strategies
a2a-cli strategies
```

---

## Configuration

### Server Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-db` | `~/.a2a-negotiation/negotiations.db` | Path to SQLite database |
| `-seed` | `""` | Path to seed CSV file (optional) |
| `-log` | `json` | Log format: `json` or `text` |
| `-http` | `""` | HTTP listen address for A2A endpoints (e.g. `:8080`) |
| `-slack-webhook` | `""` | Slack webhook URL for negotiation alerts |
| `-api-keys` | `""` | Path to JSON file of API keys (enables auth) |
| `-rate-limit` | `0` | Max requests per minute per key (0 = unlimited) |

### Examples

```bash
# stdio only
./bin/a2a-negotiation

# With seed data
./bin/a2a-negotiation -seed data/seeds/saas_pricing.csv

# HTTP + stdio
./bin/a2a-negotiation -http :8080 -seed data/seeds/saas_pricing.csv

# Custom DB + text logging
./bin/a2a-negotiation -db /tmp/negotiations.db -log text

# Full-featured
./bin/a2a-negotiation -http :8080 \
  -seed data/seeds/saas_pricing.csv \
  -slack-webhook https://hooks.slack.com/services/T00/B00/xxx \
  -api-keys /path/to/api-keys.json \
  -rate-limit 60
```

### MCP Settings (Claude Code / Cline)

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

---

## Seed Data

`data/seeds/saas_pricing.csv` contains ~50 common SaaS products with known pricing ranges:

- **Productivity:** Slack, Notion, Linear
- **DevTools:** GitHub, GitLab, JetBrains, VS Code Copilot
- **Analytics:** Datadog, New Relic, Splunk, Grafana, Sentry
- **CRM/Sales:** Salesforce, HubSpot, Outreach, SalesLoft
- **Collaboration:** Zoom, Figma, Atlassian, Miro, Asana, Monday.com, Coda
- **Cloud:** AWS, Google Cloud, Microsoft 365, Cloudflare, Fastly, Vercel, Netlify
- **Security:** Okta, 1Password, CrowdStrike, Snyk, Wiz
- **DevOps:** PagerDuty, CircleCI, LaunchDarkly, HashiCorp, Databricks
- **Support:** Zendesk, Intercom, Freshdesk
- **Design:** Canva, Adobe
- **HR:** Workday, Rippling, Gusto, Lattice, Culture Amp

Columns: `vendor, sku, description, list_price, min_observed, max_observed, typical_pct, unit, category`

Update pricing data:

```bash
pip install -r requirements-scrape.txt
python scripts/scrape-pricing.py
```

---

## Development

### Build

```bash
go build -o bin/a2a-negotiation ./cmd/server
go build -o bin/a2a-cli ./cmd/cli
```

### Test

```bash
go test ./... -v -count=1
go test ./internal/negotiation/... -v
go test ./internal/a2a/... -v
go test ./internal/server/... -v
```

Covers: price query, session creation, negotiation loop (accept/budget/walk-away), history CRUD, parallel negotiation, buying groups, contracts, marketplace, SLA management, quote analysis, contract parsing, vendor health, A2A routing, authentication, rate limiting, webhooks.

### Add a Package

1. Create `internal/<package>/` with Go files
2. Add the dependency to `NegotiationServer` in `internal/server/tools.go`
3. Wire it in `cmd/server/main.go` (init store/engine, pass to `NewNegotiationServer`)
4. Add MCP tool(s) in `registerTools()` in `internal/server/tools.go`
5. Add tests in `<package>_test.go`
6. Run `go test ./... -v -count=1`

### Strategy Profiles

| Strategy | Start | Rounds | Style |
|----------|-------|--------|-------|
| **Aggressive** | 30% below asking | 4-5 | Concede slowly, hard budget limit |
| **Balanced** | 20% below asking | 3-4 | Moderate concessions, data-driven |
| **Conservative** | 10% below asking | 2-3 | Quick to accept, relationship-preserving |

---

## Docker

### Build

```bash
docker build -t a2a-negotiation-mcp .
```

### Run

```bash
# stdio only
docker run -i --rm a2a-negotiation-mcp

# With seed data
docker run -i --rm \
  -v $(pwd)/data:/app/data \
  a2a-negotiation-mcp \
  ./bin/a2a-negotiation -seed /app/data/seeds/saas_pricing.csv

# With HTTP
docker run -i --rm -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  a2a-negotiation-mcp \
  ./bin/a2a-negotiation -http :8080 -seed /app/data/seeds/saas_pricing.csv
```

### Docker Compose

```bash
docker compose up
```

---

## A2A HTTP Endpoints

When `-http` is set, the server serves A2A (Agent-to-Agent) protocol endpoints alongside stdio.

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/a2a/task` | Dispatch a task (`query_price`, `mandate_create`, `mandate_settle`, `mandate_cancel`) |
| `GET` | `/a2a/task/{id}` | Session status and result for a previously submitted task |
| `POST` | `/a2a/negotiate` | Create a mandate + negotiation session |
| `GET` | `/.well-known/agent-card.json` | A2A Agent Card describing capabilities |

### Examples

```bash
# Query price
curl -X POST http://localhost:8080/a2a/task \
  -H 'Content-Type: application/json' \
  -d '{"task":"query_price","params":{"vendor":"Slack","sku":"Pro"}}'

# Negotiate
curl -X POST http://localhost:8080/a2a/negotiate \
  -H 'Content-Type: application/json' \
  -d '{"vendor":"Slack","sku":"Pro","strategy":"balanced","budget":7.00}'

# Session status
curl http://localhost:8080/a2a/task/<session-id>

# Agent Card
curl http://localhost:8080/.well-known/agent-card.json
```

### Modes

| Mode | Command | Transport |
|------|---------|-----------|
| stdio only | `./bin/a2a-negotiation` | MCP JSON-RPC over stdin/stdout |
| HTTP + stdio | `./bin/a2a-negotiation -http :8080` | Both concurrently |

---

## Tech Stack

- **Go 1.26** — Modern Go with `slog` for structured JSON logging
- **modernc.org/sqlite** — CGo-free SQLite (shared across all stores)
- **github.com/mark3labs/mcp-go** — MCP protocol (JSON-RPC over stdio)
- **Slack Block Kit** — Rich message formatting for Slack alerts

---

## License

MIT
