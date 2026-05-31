package translation

import (
	"context"
	"fmt"
	"strings"
)

// Engine provides stateless translation and glossary-building operations.
type Engine struct{}

// NewEngine creates a new translation engine.
func NewEngine() *Engine {
	return &Engine{}
}

var supportedLanguages = map[string]bool{
	"en": true,
	"es": true,
	"fr": true,
	"de": true,
	"zh": true,
	"ja": true,
	"ar": true,
}

// Translate simulates translating text into the target language by prepending
// a language tag. The targetLang must be one of [en, es, fr, de, zh, ja, ar].
func (e *Engine) Translate(ctx context.Context, text, targetLang string) (*TranslationResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text must not be empty")
	}
	if !supportedLanguages[targetLang] {
		return nil, fmt.Errorf("unsupported target language: %s", targetLang)
	}
	return &TranslationResult{
		OriginalText:   text,
		TranslatedText: fmt.Sprintf("[%s] %s", targetLang, text),
		TargetLanguage: targetLang,
	}, nil
}

// BuildGlossary creates a glossary from a list of source terms for the given
// language pair. Each term is mapped to a simulated translation.
func (e *Engine) BuildGlossary(ctx context.Context, terms []string, fromLang, toLang string) (*Glossary, error) {
	entries := make([]GlossaryEntry, len(terms))
	for i, term := range terms {
		entries[i] = GlossaryEntry{
			SourceTerm: term,
			TargetTerm: fmt.Sprintf("[%s] %s", toLang, term),
		}
	}
	return &Glossary{
		FromLanguage: fromLang,
		ToLanguage:   toLang,
		Entries:      entries,
	}, nil
}
