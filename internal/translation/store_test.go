package translation

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

func TestSetPreference(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	pref, err := s.SetPreference(ctx, "Acme Corp", "es")
	if err != nil {
		t.Fatalf("SetPreference: %v", err)
	}

	if pref.Vendor != "Acme Corp" {
		t.Errorf("expected vendor 'Acme Corp', got %s", pref.Vendor)
	}
	if pref.Language != "es" {
		t.Errorf("expected language 'es', got %s", pref.Language)
	}
	if pref.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
}

func TestGetPreference(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.SetPreference(ctx, "Globex", "fr")
	if err != nil {
		t.Fatalf("SetPreference: %v", err)
	}

	got, err := s.GetPreference(ctx, "Globex")
	if err != nil {
		t.Fatalf("GetPreference: %v", err)
	}

	if got.Vendor != "Globex" {
		t.Errorf("expected vendor 'Globex', got %s", got.Vendor)
	}
	if got.Language != "fr" {
		t.Errorf("expected language 'fr', got %s", got.Language)
	}
}

func TestGetPreference_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetPreference(ctx, "UnknownVendor")
	if err == nil {
		t.Fatal("expected error for non-existent preference, got nil")
	}
}

func TestSetPreference_UpdatesExisting(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.SetPreference(ctx, "Acme Corp", "es")
	if err != nil {
		t.Fatalf("SetPreference: %v", err)
	}

	_, err = s.SetPreference(ctx, "Acme Corp", "de")
	if err != nil {
		t.Fatalf("SetPreference update: %v", err)
	}

	got, err := s.GetPreference(ctx, "Acme Corp")
	if err != nil {
		t.Fatalf("GetPreference: %v", err)
	}
	if got.Language != "de" {
		t.Errorf("expected updated language 'de', got %s", got.Language)
	}
}

func TestSaveAndGetGlossary(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	entries := []GlossaryEntry{
		{SourceTerm: "hello", TargetTerm: "[es] hello"},
		{SourceTerm: "world", TargetTerm: "[es] world"},
	}

	id, err := s.SaveGlossary(ctx, "en", "es", entries)
	if err != nil {
		t.Fatalf("SaveGlossary: %v", err)
	}
	if id == 0 {
		t.Errorf("expected non-zero glossary ID")
	}

	got, err := s.GetGlossary(ctx, "en", "es")
	if err != nil {
		t.Fatalf("GetGlossary: %v", err)
	}

	if got.FromLanguage != "en" {
		t.Errorf("expected from 'en', got %s", got.FromLanguage)
	}
	if got.ToLanguage != "es" {
		t.Errorf("expected to 'es', got %s", got.ToLanguage)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got.Entries))
	}
	if got.Entries[0].SourceTerm != "hello" {
		t.Errorf("expected source term 'hello', got %s", got.Entries[0].SourceTerm)
	}
}

func TestGetGlossary_NotFound(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	_, err := s.GetGlossary(ctx, "en", "de")
	if err == nil {
		t.Fatal("expected error for non-existent glossary, got nil")
	}
}
