package translation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Store manages translation-related persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a TranslationStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate translation: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vendor_language_prefs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL UNIQUE,
		language TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS glossary (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_language TEXT NOT NULL,
		to_language TEXT NOT NULL,
		entries_json TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SetPreference sets (inserts or replaces) the language preference for a vendor.
func (s *Store) SetPreference(ctx context.Context, vendor, language string) (*LanguagePreference, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO vendor_language_prefs (vendor, language)
		VALUES (?, ?)
		ON CONFLICT(vendor) DO UPDATE SET language = excluded.language
	`, vendor, language)
	if err != nil {
		return nil, fmt.Errorf("set preference: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("set preference last insert id: %w", err)
	}
	return &LanguagePreference{
		ID:       int(id),
		Vendor:   vendor,
		Language: language,
	}, nil
}

// GetPreference retrieves the language preference for a vendor.
func (s *Store) GetPreference(ctx context.Context, vendor string) (*LanguagePreference, error) {
	var p LanguagePreference
	err := s.db.QueryRowContext(ctx, `
		SELECT id, vendor, language FROM vendor_language_prefs WHERE vendor = ?
	`, vendor).Scan(&p.ID, &p.Vendor, &p.Language)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("preference not found: vendor=%s", vendor)
	}
	if err != nil {
		return nil, fmt.Errorf("get preference: %w", err)
	}
	return &p, nil
}

// SaveGlossary stores a glossary as a JSON blob in a single row. Returns the row ID.
func (s *Store) SaveGlossary(ctx context.Context, fromLang, toLang string, entries []GlossaryEntry) (int, error) {
	data, err := json.Marshal(entries)
	if err != nil {
		return 0, fmt.Errorf("save glossary marshal: %w", err)
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO glossary (from_language, to_language, entries_json)
		VALUES (?, ?, ?)
	`, fromLang, toLang, string(data))
	if err != nil {
		return 0, fmt.Errorf("save glossary: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("save glossary last insert id: %w", err)
	}
	return int(id), nil
}

// GetGlossary retrieves a glossary for the given language pair.
func (s *Store) GetGlossary(ctx context.Context, fromLang, toLang string) (*Glossary, error) {
	var (
		id          int
		entriesJSON string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, entries_json FROM glossary
		WHERE from_language = ? AND to_language = ?
		ORDER BY id DESC LIMIT 1
	`, fromLang, toLang).Scan(&id, &entriesJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("glossary not found: %s->%s", fromLang, toLang)
	}
	if err != nil {
		return nil, fmt.Errorf("get glossary: %w", err)
	}

	var entries []GlossaryEntry
	if err := json.Unmarshal([]byte(entriesJSON), &entries); err != nil {
		return nil, fmt.Errorf("get glossary unmarshal: %w", err)
	}

	return &Glossary{
		FromLanguage: fromLang,
		ToLanguage:   toLang,
		Entries:      entries,
	}, nil
}
