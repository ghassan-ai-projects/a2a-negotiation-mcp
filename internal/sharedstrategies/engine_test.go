package sharedstrategies

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *Store {
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

func TestShareStrategy_CreatesEntry(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	st, err := eng.ShareStrategy(ctx, "Aggressive Discount", "Push for 30% off", "aggressive")
	if err != nil {
		t.Fatalf("ShareStrategy: %v", err)
	}

	if st.Name != "Aggressive Discount" {
		t.Errorf("expected name 'Aggressive Discount', got %q", st.Name)
	}
	if st.StrategyType != "aggressive" {
		t.Errorf("expected strategy_type 'aggressive', got %q", st.StrategyType)
	}
	if st.UsageCount != 0 {
		t.Errorf("expected usage_count 0, got %d", st.UsageCount)
	}
	if st.ID == "" {
		t.Error("expected non-empty ID")
	}
	if st.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
}

func TestShareStrategy_DefaultType(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	st, err := eng.ShareStrategy(ctx, "Default", "no type given", "")
	if err != nil {
		t.Fatalf("ShareStrategy: %v", err)
	}
	if st.StrategyType != "balanced" {
		t.Errorf("expected default 'balanced', got %q", st.StrategyType)
	}
}

func TestList_ReturnsAll(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	eng.ShareStrategy(ctx, "A", "notes a", "aggressive")
	eng.ShareStrategy(ctx, "B", "notes b", "balanced")

	list, err := eng.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(list))
	}
}

func TestImportStrategy_IncrementsCount(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	st, err := eng.ShareStrategy(ctx, "Counted", "test", "balanced")
	if err != nil {
		t.Fatalf("ShareStrategy: %v", err)
	}

	imported, err := eng.ImportStrategy(ctx, st.ID)
	if err != nil {
		t.Fatalf("ImportStrategy: %v", err)
	}

	if imported.UsageCount != 1 {
		t.Errorf("expected usage_count 1 after import, got %d", imported.UsageCount)
	}

	// Verify store-level persistence
	stored, err := store.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.UsageCount != 1 {
		t.Errorf("expected stored usage_count 1, got %d", stored.UsageCount)
	}
}

func TestImportStrategy_NotFound(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	_, err := eng.ImportStrategy(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent strategy")
	}
}

func TestDelete_RemovesEntry(t *testing.T) {
	store := setupTestStore(t)
	eng := NewEngine(store)
	ctx := context.Background()

	st, err := eng.ShareStrategy(ctx, "Deletable", "to be deleted", "balanced")
	if err != nil {
		t.Fatalf("ShareStrategy: %v", err)
	}

	if err := store.Delete(ctx, st.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}
