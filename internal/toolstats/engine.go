package toolstats

import (
	"context"
	"fmt"
	"time"
)

// Engine provides tool usage statistics.
type Engine struct {
	store *Store
}

// NewEngine creates a new toolstats engine.
func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

// parsePeriod converts a period string ("24h", "7d", "30d") to a time.Duration.
func parsePeriod(period string) (time.Duration, error) {
	switch period {
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported period: %q (use 24h, 7d, or 30d)", period)
	}
}

// GetReport returns a usage report for the given period.
func (e *Engine) GetReport(ctx context.Context, period string) (*UsageReport, error) {
	dur, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}
	since := time.Now().Add(-dur)

	top, err := e.store.GetTopTools(ctx, since, 10)
	if err != nil {
		return nil, fmt.Errorf("get top tools: %w", err)
	}

	all, err := e.store.CountByTool(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("count by tool: %w", err)
	}

	total, err := e.store.TotalCalls(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("total calls: %w", err)
	}

	uniqueTools := len(all)

	// Bottom tools: reverse of all (take up to 10)
	var bottom []ToolUsage
	if len(all) > 10 {
		bottom = make([]ToolUsage, 10)
		for i := 0; i < 10; i++ {
			bottom[i] = all[len(all)-1-i]
		}
	} else {
		bottom = make([]ToolUsage, len(all))
		for i := 0; i < len(all); i++ {
			bottom[i] = all[len(all)-1-i]
		}
	}

	return &UsageReport{
		Period:      period,
		TotalCalls:  total,
		UniqueTools: uniqueTools,
		TopTools:    top,
		BottomTools: bottom,
	}, nil
}

// LogCall records a tool call for usage tracking.
func (e *Engine) LogCall(ctx context.Context, toolName string) error {
	return e.store.LogCall(ctx, toolName)
}
