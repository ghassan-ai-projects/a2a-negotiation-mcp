package dataretention

import (
	"fmt"
	"sync"
	"time"
)

var validDataTypes = map[string]bool{
	"sessions":    true,
	"outcomes":    true,
	"alerts":      true,
	"audit_log":   true,
	"usage_stats": true,
}

var validActions = map[string]bool{
	"delete":  true,
	"archive": true,
}

// Engine manages data retention policies in-memory.
type Engine struct {
	mu       sync.Mutex
	policies map[string]RetentionPolicy
}

// NewEngine creates a new Engine with no policies.
func NewEngine() *Engine {
	return &Engine{
		policies: make(map[string]RetentionPolicy),
	}
}

// SetPolicy creates or updates a retention policy for the given data type.
// dataType must be one of: sessions, outcomes, alerts, audit_log, usage_stats.
// action must be one of: delete, archive.
func (e *Engine) SetPolicy(dataType string, retentionDays int, action string) error {
	if !validDataTypes[dataType] {
		return fmt.Errorf("invalid data type %q: must be one of sessions, outcomes, alerts, audit_log, usage_stats", dataType)
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action %q: must be one of delete, archive", action)
	}
	if retentionDays < 1 {
		return fmt.Errorf("retention_days must be at least 1")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	existing, found := e.policies[dataType]
	if found {
		e.policies[dataType] = RetentionPolicy{
			DataType:      dataType,
			RetentionDays: retentionDays,
			Action:        action,
			CreatedAt:     existing.CreatedAt,
			UpdatedAt:     now,
		}
	} else {
		e.policies[dataType] = RetentionPolicy{
			DataType:      dataType,
			RetentionDays: retentionDays,
			Action:        action,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	}

	return nil
}

// GetPolicies returns all current retention policies.
func (e *Engine) GetPolicies() []RetentionPolicy {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]RetentionPolicy, 0, len(e.policies))
	for _, p := range e.policies {
		result = append(result, p)
	}
	return result
}

// PurgeOldData simulates purging old data according to each policy.
// When dryRun is true, the RecordsDeleted field is an estimate of how many
// records would be deleted (simulated as retentionDays / 10).
// When dryRun is false, the count is the same (since this is an in-memory
// simulation) but the semantics indicate a real purge would have occurred.
func (e *Engine) PurgeOldData(dryRun bool) []PurgeResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	results := make([]PurgeResult, 0, len(e.policies))
	for _, p := range e.policies {
		// Simulated: assume 100 records exist, (retentionDays / 365) fraction are old
		simulatedDeleted := 0
		if p.RetentionDays < 365 {
			simulatedDeleted = 100 * (365 - p.RetentionDays) / 365
			if simulatedDeleted < 0 {
				simulatedDeleted = 0
			}
		}
		results = append(results, PurgeResult{
			DataType:       p.DataType,
			RecordsDeleted: simulatedDeleted,
			DryRun:         dryRun,
		})
	}
	return results
}
