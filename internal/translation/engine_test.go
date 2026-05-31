package translation

import (
	"context"
	"testing"
)

func TestTranslate(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.Translate(ctx, "Hello world", "es")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if result.TranslatedText != "[es] Hello world" {
		t.Errorf("expected '[es] Hello world', got %s", result.TranslatedText)
	}
	if result.TargetLanguage != "es" {
		t.Errorf("expected target 'es', got %s", result.TargetLanguage)
	}
	if result.OriginalText != "Hello world" {
		t.Errorf("expected original 'Hello world', got %s", result.OriginalText)
	}
}

func TestTranslate_InvalidLanguage(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.Translate(ctx, "Hello", "xx")
	if err == nil {
		t.Fatal("expected error for invalid language, got nil")
	}
}

func TestBuildGlossary(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	terms := []string{"hello", "world", "contract"}
	glossary, err := eng.BuildGlossary(ctx, terms, "en", "fr")
	if err != nil {
		t.Fatalf("BuildGlossary: %v", err)
	}

	if glossary.FromLanguage != "en" {
		t.Errorf("expected from 'en', got %s", glossary.FromLanguage)
	}
	if glossary.ToLanguage != "fr" {
		t.Errorf("expected to 'fr', got %s", glossary.ToLanguage)
	}
	if len(glossary.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(glossary.Entries))
	}
	if glossary.Entries[0].SourceTerm != "hello" {
		t.Errorf("expected source 'hello', got %s", glossary.Entries[0].SourceTerm)
	}
	if glossary.Entries[0].TargetTerm != "[fr] hello" {
		t.Errorf("expected target '[fr] hello', got %s", glossary.Entries[0].TargetTerm)
	}
}
