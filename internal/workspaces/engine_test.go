package workspaces

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	_ "modernc.org/sqlite"
)

func setupTest(t *testing.T) (*Engine, *Store) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:workspaces_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pstore, err := pricing.NewStoreFromDB(db)
	if err != nil {
		t.Fatalf("pricing NewStoreFromDB: %v", err)
	}

	store, err := NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("workspaces NewStore: %v", err)
	}

	// Create deal_outcomes table for summary queries
	_, err = pstore.DB().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS deal_outcomes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vendor TEXT,
			sku TEXT,
			list_price REAL,
			final_price REAL,
			discount_pct REAL,
			created_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create deal_outcomes: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, pstore.DB(), logger)
	return eng, store
}

func TestCreateWorkspace(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	ws, err := eng.Create(ctx, "Test Workspace", "A test workspace")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ws.Name != "Test Workspace" {
		t.Errorf("expected name 'Test Workspace', got %q", ws.Name)
	}
	if ws.Description != "A test workspace" {
		t.Errorf("expected description 'A test workspace', got %q", ws.Description)
	}
	if ws.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if ws.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestListWorkspaces(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	_, err := eng.Create(ctx, "Alpha", "First workspace")
	if err != nil {
		t.Fatalf("Create Alpha: %v", err)
	}
	_, err = eng.Create(ctx, "Beta", "Second workspace")
	if err != nil {
		t.Fatalf("Create Beta: %v", err)
	}

	workspaces, err := eng.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}
	// Should be sorted by name
	if workspaces[0].Name != "Alpha" || workspaces[1].Name != "Beta" {
		t.Errorf("expected sorted order Alpha, Beta, got %q, %q", workspaces[0].Name, workspaces[1].Name)
	}
}

func TestWorkspaceSummary_WithDealData(t *testing.T) {
	eng, _ := setupTest(t)
	ctx := context.Background()

	ws, err := eng.Create(ctx, "Data Workspace", "With deals")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Seed deal data into deal_outcomes for summary to read
	_, err = eng.db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`, "VendorA", "SKU1", 100.0, 80.0, 20.0)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}
	_, err = eng.db.ExecContext(ctx, `
		INSERT INTO deal_outcomes (vendor, sku, list_price, final_price, discount_pct, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`, "VendorB", "SKU2", 200.0, 150.0, 25.0)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	summary, err := eng.Summary(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.ID != ws.ID {
		t.Errorf("expected workspace ID %q, got %q", ws.ID, summary.ID)
	}
	if summary.Name != "Data Workspace" {
		t.Errorf("expected name 'Data Workspace', got %q", summary.Name)
	}
	if summary.VendorCount != 2 {
		t.Errorf("expected 2 vendors, got %d", summary.VendorCount)
	}
	if summary.DealCount != 2 {
		t.Errorf("expected 2 deals, got %d", summary.DealCount)
	}
	if summary.TotalSavings <= 0 {
		t.Errorf("expected positive total savings, got %f", summary.TotalSavings)
	}
}

func TestDeleteWorkspace(t *testing.T) {
	eng, store := setupTest(t)
	ctx := context.Background()

	ws, err := eng.Create(ctx, "Delete Me", "To be deleted")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete(ctx, ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.Get(ctx, ws.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after deletion")
	}
}
