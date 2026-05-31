package translation

// LanguagePreference represents a vendor's preferred language for negotiation.
type LanguagePreference struct {
	ID       int    `json:"id"`
	Vendor   string `json:"vendor"`
	Language string `json:"language"`
}

// TranslationResult holds the result of a text translation.
type TranslationResult struct {
	OriginalText     string `json:"original_text"`
	TranslatedText   string `json:"translated_text"`
	TargetLanguage   string `json:"target_language"`
	DetectedLanguage string `json:"detected_language,omitempty"`
}

// GlossaryEntry maps a single source term to its target-language equivalent.
type GlossaryEntry struct {
	SourceTerm string `json:"source_term"`
	TargetTerm string `json:"target_term"`
}

// Glossary groups a set of glossary entries for a specific language pair.
type Glossary struct {
	FromLanguage string          `json:"from_language"`
	ToLanguage   string          `json:"to_language"`
	Entries      []GlossaryEntry `json:"entries"`
}
