package datresidency

import (
	"context"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

func setupTest(t *testing.T) *Store {
	t.Helper()
	pStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pStore.Close() })

	store, err := NewStore(pStore.DB())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestSetAndGetRule(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	rule, err := store.SetRule(ctx, "eu", true)
	if err != nil {
		t.Fatalf("SetRule: %v", err)
	}
	if rule.Region != "eu" {
		t.Errorf("expected region eu, got %s", rule.Region)
	}
	if !rule.Allowed {
		t.Errorf("expected allowed true, got false")
	}
	if rule.UpdatedAt == "" {
		t.Error("expected non-empty updated_at")
	}

	got, err := store.GetRule(ctx, "eu")
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if got.Region != "eu" {
		t.Errorf("expected region eu, got %s", got.Region)
	}
	if !got.Allowed {
		t.Errorf("expected allowed true, got false")
	}
}

func TestSetRule_Replace(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.SetRule(ctx, "us", true)
	if err != nil {
		t.Fatalf("SetRule: %v", err)
	}

	rule, err := store.SetRule(ctx, "us", false)
	if err != nil {
		t.Fatalf("SetRule (replace): %v", err)
	}
	if rule.Allowed {
		t.Errorf("expected allowed false after replace, got true")
	}
}

func TestGetRule_NotFound(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.GetRule(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rule")
	}
}

func TestListRules_Empty(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	rules, err := store.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestListRules_Multiple(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.SetRule(ctx, "eu", true)
	if err != nil {
		t.Fatalf("SetRule eu: %v", err)
	}
	_, err = store.SetRule(ctx, "us", true)
	if err != nil {
		t.Fatalf("SetRule us: %v", err)
	}
	_, err = store.SetRule(ctx, "cn", false)
	if err != nil {
		t.Fatalf("SetRule cn: %v", err)
	}

	rules, err := store.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	if rules[0].Region != "cn" {
		t.Errorf("expected first region cn, got %s", rules[0].Region)
	}
	if rules[1].Region != "eu" {
		t.Errorf("expected second region eu, got %s", rules[1].Region)
	}
	if rules[2].Region != "us" {
		t.Errorf("expected third region us, got %s", rules[2].Region)
	}
}

func TestCheckVendor_Compliant(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.SetRule(ctx, "eu", true)
	if err != nil {
		t.Fatalf("SetRule: %v", err)
	}

	check, err := store.CheckVendor(ctx, "Slack", "eu")
	if err != nil {
		t.Fatalf("CheckVendor: %v", err)
	}
	if check.Vendor != "Slack" {
		t.Errorf("expected vendor Slack, got %s", check.Vendor)
	}
	if check.Region != "eu" {
		t.Errorf("expected region eu, got %s", check.Region)
	}
	if !check.Compliant {
		t.Error("expected compliant true")
	}
	if !check.RuleFound {
		t.Error("expected rule_found true")
	}
}

func TestCheckVendor_NotCompliant(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.SetRule(ctx, "eu", false)
	if err != nil {
		t.Fatalf("SetRule: %v", err)
	}

	check, err := store.CheckVendor(ctx, "Slack", "eu")
	if err != nil {
		t.Fatalf("CheckVendor: %v", err)
	}
	if check.Compliant {
		t.Error("expected compliant false")
	}
	if !check.RuleFound {
		t.Error("expected rule_found true")
	}
}

func TestCheckVendor_NoRule(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	check, err := store.CheckVendor(ctx, "Slack", "unknown")
	if err != nil {
		t.Fatalf("CheckVendor: %v", err)
	}
	if check.Compliant {
		t.Error("expected compliant false for unknown region")
	}
	if check.RuleFound {
		t.Error("expected rule_found false for unknown region")
	}
}
