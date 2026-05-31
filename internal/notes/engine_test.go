package notes

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupNotesStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestAddNote_CreatesNote(t *testing.T) {
	store := setupNotesStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	note, err := eng.AddNote(ctx, "session-1", "Important note")
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	if note.SessionID != "session-1" {
		t.Errorf("expected session_id 'session-1', got %q", note.SessionID)
	}
	if note.Content != "Important note" {
		t.Errorf("expected content 'Important note', got %q", note.Content)
	}
	if note.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestListNotes_BySession(t *testing.T) {
	store := setupNotesStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	eng.AddNote(ctx, "session-1", "Note 1")
	eng.AddNote(ctx, "session-1", "Note 2")
	eng.AddNote(ctx, "session-2", "Other session note")

	notes, err := eng.ListNotes(ctx, "session-1")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	if len(notes) != 2 {
		t.Fatalf("expected 2 notes for session-1, got %d", len(notes))
	}
}

func TestListNotes_NoNotesReturnsEmpty(t *testing.T) {
	store := setupNotesStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	notes, err := eng.ListNotes(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestDeleteNote_RemovesEntry(t *testing.T) {
	store := setupNotesStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	note, err := eng.AddNote(ctx, "session-1", "Delete me")
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	if err := eng.DeleteNote(ctx, note.ID); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	notes, err := eng.ListNotes(ctx, "session-1")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes after delete, got %d", len(notes))
	}
}
