package serverconfig

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

func TestGetConfig_Empty(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	entries, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if entries == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestSetAndGetConfig(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	err := s.SetConfig(ctx, "server.port", "8080")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	err = s.SetConfig(ctx, "server.host", "0.0.0.0")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	entries, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	configMap := make(map[string]string)
	for _, e := range entries {
		configMap[e.Key] = e.Value
	}
	if configMap["server.port"] != "8080" {
		t.Errorf("expected server.port=8080, got %s", configMap["server.port"])
	}
	if configMap["server.host"] != "0.0.0.0" {
		t.Errorf("expected server.host=0.0.0.0, got %s", configMap["server.host"])
	}
}

func TestSetConfig_Replace(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	err := s.SetConfig(ctx, "db.url", "old-value")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	err = s.SetConfig(ctx, "db.url", "new-value")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	entries, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after replace, got %d", len(entries))
	}
	if entries[0].Value != "new-value" {
		t.Errorf("expected value 'new-value', got '%s'", entries[0].Value)
	}
	if entries[0].UpdatedAt == "" {
		t.Errorf("expected non-empty updated_at")
	}
}

func TestExportConfig(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	err := s.SetConfig(ctx, "log.level", "debug")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	err = s.SetConfig(ctx, "log.format", "json")
	if err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	jsonStr, err := s.ExportConfig(ctx)
	if err != nil {
		t.Fatalf("ExportConfig: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON string")
	}
}

func TestImportConfig(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	jsonData := `[
		{"key": "theme", "value": "dark"},
		{"key": "lang", "value": "en"}
	]`

	count, err := s.ImportConfig(ctx, jsonData)
	if err != nil {
		t.Fatalf("ImportConfig: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 imported entries, got %d", count)
	}

	entries, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	configMap := make(map[string]string)
	for _, e := range entries {
		configMap[e.Key] = e.Value
	}
	if configMap["theme"] != "dark" {
		t.Errorf("expected theme=dark, got %s", configMap["theme"])
	}
	if configMap["lang"] != "en" {
		t.Errorf("expected lang=en, got %s", configMap["lang"])
	}
}

func TestImportConfig_MalformedJSON(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.ImportConfig(ctx, "not valid json")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
