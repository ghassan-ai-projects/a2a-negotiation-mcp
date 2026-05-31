package spendingcaps

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) (*Engine, *Store, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, db, logger)
	return eng, store, db
}

func TestSetAndGetCap(t *testing.T) {
	eng, store, _ := setupTest(t)
	ctx := context.Background()

	err := eng.SetCap(ctx, "Slack", 1000, 1500, "monthly")
	if err != nil {
		t.Fatalf("SetCap: %v", err)
	}
	cap, err := store.GetCap(ctx, "Slack")
	if err != nil {
		t.Fatalf("GetCap: %v", err)
	}
	if cap.Vendor != "Slack" {
		t.Errorf("expected Slack, got %s", cap.Vendor)
	}
	if cap.SoftCap != 1000 {
		t.Errorf("expected soft_cap 1000, got %f", cap.SoftCap)
	}
}

func TestListCaps(t *testing.T) {
	eng, _, _ := setupTest(t)
	ctx := context.Background()
	eng.SetCap(ctx, "Slack", 1000, 1500, "monthly")
	eng.SetCap(ctx, "GitHub", 2000, 3000, "quarterly")

	caps, err := eng.Store().ListCaps(ctx)
	if err != nil {
		t.Fatalf("ListCaps: %v", err)
	}
	if len(caps) != 2 {
		t.Errorf("expected 2 caps, got %d", len(caps))
	}
}

func TestDeleteCap(t *testing.T) {
	eng, store, _ := setupTest(t)
	ctx := context.Background()
	eng.SetCap(ctx, "Slack", 1000, 1500, "monthly")
	store.DeleteCap(ctx, "Slack")

	cap, _ := store.GetCap(ctx, "Slack")
	if cap != nil {
		t.Error("expected cap to be deleted")
	}
}
