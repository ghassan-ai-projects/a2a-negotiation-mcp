package dataretention

import (
	"testing"
	"time"
)

func TestSetPolicy(t *testing.T) {
	eng := NewEngine()

	err := eng.SetPolicy("sessions", 90, "delete")
	if err != nil {
		t.Fatalf("SetPolicy(sessions, 90, delete) failed: %v", err)
	}

	policies := eng.GetPolicies()
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].DataType != "sessions" {
		t.Errorf("expected data_type sessions, got %s", policies[0].DataType)
	}
	if policies[0].RetentionDays != 90 {
		t.Errorf("expected retention_days 90, got %d", policies[0].RetentionDays)
	}
	if policies[0].Action != "delete" {
		t.Errorf("expected action delete, got %s", policies[0].Action)
	}
	if policies[0].CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
	if policies[0].UpdatedAt == "" {
		t.Error("expected non-empty UpdatedAt")
	}
}

func TestSetPolicy_InvalidDataType(t *testing.T) {
	eng := NewEngine()
	err := eng.SetPolicy("invalid_type", 90, "delete")
	if err == nil {
		t.Fatal("expected error for invalid data type")
	}
}

func TestSetPolicy_InvalidAction(t *testing.T) {
	eng := NewEngine()
	err := eng.SetPolicy("sessions", 90, "compress")
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestSetPolicy_NegativeRetention(t *testing.T) {
	eng := NewEngine()
	err := eng.SetPolicy("sessions", 0, "delete")
	if err == nil {
		t.Fatal("expected error for zero retention_days")
	}
}

func TestUpdateExistingPolicy(t *testing.T) {
	eng := NewEngine()

	err := eng.SetPolicy("alerts", 30, "delete")
	if err != nil {
		t.Fatalf("initial SetPolicy failed: %v", err)
	}

	policies := eng.GetPolicies()
	originalCreated := policies[0].CreatedAt
	originalUpdated := policies[0].UpdatedAt

	// Ensure enough time passes so UpdatedAt differs
	time.Sleep(time.Millisecond)

	err = eng.SetPolicy("alerts", 60, "archive")
	if err != nil {
		t.Fatalf("update SetPolicy failed: %v", err)
	}

	policies = eng.GetPolicies()
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy after update, got %d", len(policies))
	}
	if policies[0].RetentionDays != 60 {
		t.Errorf("expected retention_days 60, got %d", policies[0].RetentionDays)
	}
	if policies[0].Action != "archive" {
		t.Errorf("expected action archive, got %s", policies[0].Action)
	}
	if policies[0].CreatedAt != originalCreated {
		t.Errorf("CreatedAt should not change on update, got %s", policies[0].CreatedAt)
	}
	if policies[0].UpdatedAt == originalUpdated {
		t.Error("UpdatedAt should change on update")
	}
}

func TestGetPolicies_Empty(t *testing.T) {
	eng := NewEngine()
	policies := eng.GetPolicies()
	if len(policies) != 0 {
		t.Fatalf("expected 0 policies for new engine, got %d", len(policies))
	}
}

func TestPurgeOldData_DryRun(t *testing.T) {
	eng := NewEngine()

	_ = eng.SetPolicy("sessions", 90, "delete")
	_ = eng.SetPolicy("audit_log", 365, "archive")

	results := eng.PurgeOldData(true)
	if len(results) != 2 {
		t.Fatalf("expected 2 purge results, got %d", len(results))
	}

	for _, r := range results {
		if !r.DryRun {
			t.Errorf("expected dry_run=true for data_type %s", r.DataType)
		}
		if r.RecordsDeleted < 0 {
			t.Errorf("expected non-negative records_deleted for %s, got %d", r.DataType, r.RecordsDeleted)
		}
	}
}

func TestPurgeOldData_MultipleDataTypes(t *testing.T) {
	eng := NewEngine()

	_ = eng.SetPolicy("sessions", 90, "delete")
	_ = eng.SetPolicy("outcomes", 180, "archive")
	_ = eng.SetPolicy("alerts", 30, "delete")

	results := eng.PurgeOldData(false)
	if len(results) != 3 {
		t.Fatalf("expected 3 purge results, got %d", len(results))
	}

	dataTypes := make(map[string]bool)
	for _, r := range results {
		dataTypes[r.DataType] = true
	}

	for _, dt := range []string{"sessions", "outcomes", "alerts"} {
		if !dataTypes[dt] {
			t.Errorf("missing purge result for %s", dt)
		}
	}
}
