package healthcheck

import (
	"context"
	"database/sql"
	"os"
	"time"
)

// Engine performs health checks against the database and MCP server.
type Engine struct {
	db        *sql.DB
	toolCount int
	startTime time.Time
	dbPath    string
}

// NewEngine creates a new healthcheck engine.
// dbPath is the filesystem path to the SQLite DB file (used for size check; may be empty).
func NewEngine(db *sql.DB, toolCount int, startTime time.Time, dbPath string) *Engine {
	return &Engine{
		db:        db,
		toolCount: toolCount,
		startTime: startTime,
		dbPath:    dbPath,
	}
}

// Check performs a health check and returns the result.
func (e *Engine) Check(ctx context.Context) *HealthResult {
	dbOK := e.pingDB(ctx)

	var dbSize int64
	if e.dbPath != "" {
		if fi, err := os.Stat(e.dbPath); err == nil {
			dbSize = fi.Size()
		}
	}

	uptime := int64(time.Since(e.startTime).Seconds())

	status := "healthy"
	if !dbOK {
		status = "degraded"
	}

	return &HealthResult{
		Status:      status,
		DatabaseOK:  dbOK,
		ToolCount:   e.toolCount,
		DBSizeBytes: dbSize,
		UptimeSecs:  uptime,
		StartedAt:   e.startTime.UTC().Format(time.RFC3339),
	}
}

func (e *Engine) pingDB(ctx context.Context) bool {
	if e.db == nil {
		return false
	}
	if err := e.db.PingContext(ctx); err != nil {
		return false
	}
	// Also verify a simple query works
	var dummy int
	if err := e.db.QueryRowContext(ctx, "SELECT 1").Scan(&dummy); err != nil {
		return false
	}
	return true
}

// ToolCount returns the current tool count.
func (e *Engine) ToolCount() int {
	return e.toolCount
}

// SetToolCount updates the tool count (e.g., after tools are registered).
func (e *Engine) SetToolCount(n int) {
	e.toolCount = n
}
