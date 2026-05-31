package toolbilling

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages tool usage billing in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a ToolBillingStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate toolbilling: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tool_billing_prices (
		tool_name TEXT PRIMARY KEY,
		price_per_call REAL NOT NULL DEFAULT 0.01
	);
	CREATE TABLE IF NOT EXISTS tool_usage_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_id TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		call_time TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetToolPrice sets or updates the price per call for a tool.
func (s *Store) SetToolPrice(ctx context.Context, toolName string, pricePerCall float64) (*ToolPrice, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO tool_billing_prices (tool_name, price_per_call)
		VALUES (?, ?)
	`, toolName, pricePerCall)
	if err != nil {
		return nil, fmt.Errorf("set tool price: %w", err)
	}
	return &ToolPrice{
		ToolName:     toolName,
		PricePerCall: pricePerCall,
	}, nil
}

// GetBillingReport returns a billing report for a key over the given period.
func (s *Store) GetBillingReport(ctx context.Context, keyID, from, to string) (*BillingReport, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.tool_name, COUNT(*) as call_count, COALESCE(p.price_per_call, 0.01) as price
		FROM tool_usage_log l
		LEFT JOIN tool_billing_prices p ON l.tool_name = p.tool_name
		WHERE l.key_id = ? AND l.call_time >= ? AND l.call_time <= ?
		GROUP BY l.tool_name
	`, keyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get billing report query: %w", err)
	}
	defer rows.Close()

	report := &BillingReport{
		KeyID:      keyID,
		PeriodFrom: from,
		PeriodTo:   to,
		PerTool:    make(map[string]int),
	}

	var totalCost float64
	var totalCalls int

	for rows.Next() {
		var toolName string
		var callCount int
		var price float64
		if err := rows.Scan(&toolName, &callCount, &price); err != nil {
			return nil, fmt.Errorf("scan billing report: %w", err)
		}
		report.PerTool[toolName] = callCount
		totalCalls += callCount
		totalCost += float64(callCount) * price
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	report.TotalCalls = totalCalls
	report.TotalCost = totalCost
	return report, nil
}

// LogUsage records a tool usage event. This is called when a tool is invoked.
func (s *Store) LogUsage(ctx context.Context, keyID, toolName string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tool_usage_log (key_id, tool_name, call_time)
		VALUES (?, ?, ?)
	`, keyID, toolName, now)
	if err != nil {
		return fmt.Errorf("log usage: %w", err)
	}
	return nil
}

// GetUsageTier returns the current usage tier for a key.
func (s *Store) GetUsageTier(ctx context.Context, keyID string) (*UsageTier, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startStr := startOfMonth.Format(time.RFC3339)
	endStr := now.Format(time.RFC3339)

	var callsThisMonth int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tool_usage_log
		WHERE key_id = ? AND call_time >= ? AND call_time <= ?
	`, keyID, startStr, endStr).Scan(&callsThisMonth)
	if err != nil {
		return nil, fmt.Errorf("get usage tier count: %w", err)
	}

	tier := &UsageTier{
		KeyID:          keyID,
		CallsThisMonth: callsThisMonth,
	}

	switch {
	case callsThisMonth <= 100:
		tier.CurrentTier = "tier1"
		tier.TierLimit = 100
	case callsThisMonth <= 1000:
		tier.CurrentTier = "tier2"
		tier.TierLimit = 1000
	default:
		tier.CurrentTier = "tier3"
		tier.TierLimit = 0 // no limit
	}

	return tier, nil
}
