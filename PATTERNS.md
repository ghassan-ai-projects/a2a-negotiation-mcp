# Qwen Pattern Cookbook — a2a-negotiation-mcp

> Inject 2-3 relevant patterns into each Qwen ACP task prompt to enforce consistent senior-level Go. Updated as quality improves.

---

## Pattern A: In-Memory Engine (stateless / minimal state)

**Use when:** Tool doesn't need persistence, just computation or simple in-memory state.

```go
package dataretention

import (
    "fmt"
    "sync"
    "time"
)

var validDataTypes = map[string]bool{...}  // validation consts

type Engine struct {
    mu       sync.Mutex
    policies map[string]RetentionPolicy
}

func NewEngine() *Engine {
    return &Engine{
        policies: make(map[string]RetentionPolicy),
    }
}

func (e *Engine) SetPolicy(dataType string, retentionDays int, action string) error {
    if !validDataTypes[dataType] {
        return fmt.Errorf("invalid data type: %s", dataType)
    }
    e.mu.Lock()
    defer e.mu.Unlock()
    e.policies[dataType] = RetentionPolicy{
        DataType:      dataType,
        RetentionDays: retentionDays,
        Action:        action,
        UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
    }
    return nil
}

func (e *Engine) GetPolicies() []RetentionPolicy {
    e.mu.Lock()
    defer e.mu.Unlock()
    result := make([]RetentionPolicy, 0, len(e.policies))
    for _, p := range e.policies {
        result = append(result, p)
    }
    return result
}
```

**Key rules:**
- `sync.Mutex` for ALL map access (never bare map writes)
- Constructor pre-allocates the map
- `Lock()` at method entry, `defer Unlock()` immediately
- Return copies, never raw map references
- Validation in a separate step before locking

---

## Pattern B: SQLite Store (shared DB connection)

**Use when:** Tool needs persistence. Shares the same SQLite DB via `pricingStore.DB()`.

```go
package history

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
    _ "modernc.org/sqlite"
)

type Store struct {
    db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
    s := &Store{db: db}
    if err := s.migrate(); err != nil {
        return nil, fmt.Errorf("migrate history: %w", err)
    }
    return s, nil
}

func (s *Store) migrate() error {
    schema := `CREATE TABLE IF NOT EXISTS ...`
    _, err := s.db.Exec(schema)
    return err
}
```

**Key rules:**
- Constructor takes `*sql.DB` (not `dbPath` — use `pricingStore.DB()`)
- `migrate()` runs CREATE TABLE IF NOT EXISTS on init
- Always wrap errors with context: `fmt.Errorf("save report: %w", err)`
- Use `context` parameter on public methods
- Use `modernc.org/sqlite` driver (`_ "modernc.org/sqlite"`)

---

## Pattern C: Tool Registration + Handler

**Use when:** Wiring a new tool into the server.

```go
// In registerTools():
ns.mcpServer.AddTool(mcp.NewTool("negotiate_save_report",
    mcp.WithDescription("Save an industry research report."),
    mcp.WithString("title", mcp.Required(), mcp.Description("Report title")),
    mcp.WithString("category", mcp.Required(), mcp.Description("Report category")),
    mcp.WithString("content", mcp.Required(), mcp.Description("Report content")),
    mcp.WithString("source", mcp.Required(), mcp.Description("Source URL or name")),
), ns.handleSaveReport)

// Handler method:
func (ns *NegotiationServer) handleSaveReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()
    title, _ := req.RequireString("title")
    category, _ := req.RequireString("category")

    ns.logger.Info("negotiate_save_report called", "title", title, "category", category)

    if ns.industryReportsStore == nil {
        return mcp.NewToolResultError("Industry reports store is not available"), nil
    }

    report, err := ns.industryReportsStore.SaveReport(ctx, title, category, content, source)
    if err != nil {
        ns.logger.Warn("negotiate_save_report failed", "error", err.Error())
        return mcp.NewToolResultError("Save report failed: " + err.Error()), nil
    }

    result, _ := json.Marshal(report)
    ns.logger.Info("negotiate_save_report succeeded",
        "report_id", report.ID,
        "elapsed", time.Since(start).String())
    return mcp.NewToolResultText(string(result)), nil
}
```

**Key rules:**
- Log entry + exit with elapsed time
- Nil-guard every engine/store before use
- Parse params with `req.RequireString()` / `req.RequireNumber()`
- Marshal results to JSON string
- Return errors as `mcp.NewToolResultError()`, not as Go errors
- Follow existing handler ordering in registerTools()

---

## Pattern D: Struct field + Constructor param

**Use when:** Adding a new engine/store to NegotiationServer.

```go
// In struct definition (alphabetical order among peers):
trainingEng          *training.Engine

// In constructor signature (ADD BEFORE logger):
trainingEng *training.Engine

// In constructor body:
trainingEng: trainingEng,

// In main.go:
trainingEng := training.NewEngine()
```

**Key rules:**
- Add BEFORE `logger *slog.Logger` (last param)
- NEVER after logger (it changes all existing callers)
- Always nil-check in handlers

---

## Pattern E: Tests (table-driven)

**Use when:** Testing engine/store methods.

```go
func TestSimulate_StandardParams(t *testing.T) {
    eng := NewEngine()
    result, err := eng.Simulate("Acme Corp", "competitive", 100000, 3)
    if err != nil {
        t.Fatalf("Simulate() returned unexpected error: %v", err)
    }
    if len(result.Rounds) != 3 {
        t.Errorf("expected 3 rounds, got %d", len(result.Rounds))
    }
    if result.Vendor != "Acme Corp" {
        t.Errorf("expected vendor Acme Corp, got %s", result.Vendor)
    }
    if result.TotalDiscount <= 0 {
        t.Errorf("expected positive discount, got %f", result.TotalDiscount)
    }
}

func TestSimulate_InvalidStrategy(t *testing.T) {
    eng := NewEngine()
    _, err := eng.Simulate("Acme Corp", "fake_strategy", 100000, 3)
    if err == nil {
        t.Fatal("expected error for invalid strategy, got nil")
    }
}
```

**Key rules:**
- One test file per package (`package <name>`, not `package <name>_test`)
- `NewEngine()` in every test (no shared state between tests)
- `t.Fatalf()` for setup failures, `t.Errorf()` for assertion failures
- Test happy path + all error paths + edge cases (empty, nil, zero, negative)
- Use table-driven tests for parameter variations
