package learning

import (
	"context"
	"fmt"
	"time"
)

// Autopsy records the details of a failed negotiation.
type Autopsy struct {
	SessionID     string  `json:"session_id"`
	Vendor        string  `json:"vendor"`
	SKU           string  `json:"sku"`
	Strategy      string  `json:"strategy"`
	FailureReason string  `json:"failure_reason"` // "price_too_high", "vendor_refused", "budget_exceeded", "timeout", "counter_too_low"
	FinalOffer    float64 `json:"final_offer"`
	VendorBest    float64 `json:"vendor_best"`
	Gap           float64 `json:"gap"`
	TacticUsed    string  `json:"tactic_used"`
}

// FailurePattern describes a recurring failure mode for a vendor.
type FailurePattern struct {
	Vendor       string `json:"vendor"`
	Pattern      string `json:"pattern"`       // e.g., "aggressive tactics fail with Salesforce"
	FailCount    int    `json:"fail_count"`
	SuggestedFix string `json:"suggested_fix"` // e.g., "switch to balanced strategy"
}

// RecordFailure saves an autopsy of a failed negotiation.
func (e *Engine) RecordFailure(ctx context.Context, a Autopsy) error {
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO failure_autopsies (id, session_id, vendor, sku, strategy, failure_reason, final_offer, vendor_best, gap, tactic_used, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fmt.Sprintf("%s-%d", a.SessionID, time.Now().UnixNano()), a.SessionID, a.Vendor, a.SKU,
		a.Strategy, a.FailureReason, a.FinalOffer, a.VendorBest, a.Gap, a.TacticUsed, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("record failure: %w", err)
	}
	e.logger.Debug("recorded failure autopsy",
		"session_id", a.SessionID, "vendor", a.Vendor, "reason", a.FailureReason)
	return nil
}

// AnalyzeFailures returns failure patterns for a specific vendor, grouped by strategy+failure_reason.
func (e *Engine) AnalyzeFailures(ctx context.Context, vendor string) ([]FailurePattern, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT strategy, failure_reason, COUNT(*) as cnt
		FROM failure_autopsies
		WHERE vendor = ?
		GROUP BY strategy, failure_reason
		ORDER BY cnt DESC
	`, vendor)
	if err != nil {
		return nil, fmt.Errorf("analyze failures: %w", err)
	}
	defer rows.Close()

	var patterns []FailurePattern
	for rows.Next() {
		var strategy, reason string
		var count int
		if err := rows.Scan(&strategy, &reason, &count); err != nil {
			return nil, fmt.Errorf("scan failure pattern: %w", err)
		}

		suggestedFix := suggestFix(strategy, reason)
		patternLabel := patternLabel(strategy, reason)

		patterns = append(patterns, FailurePattern{
			Vendor:       vendor,
			Pattern:      patternLabel,
			FailCount:    count,
			SuggestedFix: suggestedFix,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

// CommonFailureModes returns the most common failure patterns across all vendors.
func (e *Engine) CommonFailureModes(ctx context.Context, limit int) ([]FailurePattern, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := e.db.QueryContext(ctx, `
		SELECT vendor, strategy, failure_reason, COUNT(*) as cnt
		FROM failure_autopsies
		GROUP BY vendor, strategy, failure_reason
		ORDER BY cnt DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("common failure modes: %w", err)
	}
	defer rows.Close()

	var patterns []FailurePattern
	for rows.Next() {
		var vendor, strategy, reason string
		var count int
		if err := rows.Scan(&vendor, &strategy, &reason, &count); err != nil {
			return nil, fmt.Errorf("scan common failure: %w", err)
		}

		suggestedFix := suggestFix(strategy, reason)
		patternLabel := patternLabel(strategy, reason)

		patterns = append(patterns, FailurePattern{
			Vendor:       vendor,
			Pattern:      patternLabel,
			FailCount:    count,
			SuggestedFix: suggestedFix,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

// suggestFix returns a human-readable suggested fix based on strategy and failure reason.
func suggestFix(strategy, reason string) string {
	switch {
	case reason == "vendor_refused":
		return "reduce aggressiveness — try balanced or conservative strategy"
	case reason == "counter_too_low":
		return "start with a higher initial offer or use balanced strategy"
	case reason == "price_too_high":
		return "try a more aggressive strategy to secure better pricing"
	case reason == "budget_exceeded":
		return "use balanced strategy to stay within budget constraints"
	case reason == "timeout":
		return "reduce max rounds or use conservative strategy for faster closes"
	default:
		return "switch to balanced strategy"
	}
}

// patternLabel returns a human-readable pattern label.
func patternLabel(strategy, reason string) string {
	return fmt.Sprintf("%s tactics fail — %s", strategy, reason)
}
