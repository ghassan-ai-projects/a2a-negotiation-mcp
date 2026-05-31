package contracttemplates

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func inMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateTemplate(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	eng := NewEngine(store, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	tmpl, err := eng.CreateTemplate(ctx, "Standard SaaS", "general", "This contract is between {{vendor}} and the customer.")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	if tmpl.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if tmpl.Name != "Standard SaaS" {
		t.Errorf("expected name 'Standard SaaS', got %q", tmpl.Name)
	}
	if tmpl.Category != "general" {
		t.Errorf("expected category 'general', got %q", tmpl.Category)
	}
	if tmpl.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestListByCategory(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	eng := NewEngine(store, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	_, err = eng.CreateTemplate(ctx, "SaaS 1", "saas", "Content 1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.CreateTemplate(ctx, "SaaS 2", "saas", "Content 2")
	if err != nil {
		t.Fatal(err)
	}
	_, err = eng.CreateTemplate(ctx, "Service 1", "service", "Content 3")
	if err != nil {
		t.Fatal(err)
	}

	all, err := eng.ListTemplates(ctx, "")
	if err != nil {
		t.Fatalf("ListTemplates all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 templates, got %d", len(all))
	}

	saas, err := eng.ListTemplates(ctx, "saas")
	if err != nil {
		t.Fatalf("ListTemplates saas: %v", err)
	}
	if len(saas) != 2 {
		t.Errorf("expected 2 saas templates, got %d", len(saas))
	}

	service, err := eng.ListTemplates(ctx, "service")
	if err != nil {
		t.Fatalf("ListTemplates service: %v", err)
	}
	if len(service) != 1 {
		t.Errorf("expected 1 service template, got %d", len(service))
	}
}

func TestGenerateContract(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	eng := NewEngine(store, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	tmpl, err := eng.CreateTemplate(ctx, "Master SaaS", "saas",
		"Agreement between {{vendor}} and Customer. Term: {{term}} months. Seats: {{seats}}.")
	if err != nil {
		t.Fatal(err)
	}

	contract, err := eng.GenerateContract(ctx, tmpl.ID, "Acme Corp", map[string]string{
		"term":  "24",
		"seats": "100",
	})
	if err != nil {
		t.Fatalf("GenerateContract: %v", err)
	}

	if contract.TemplateID != tmpl.ID {
		t.Errorf("expected template_id %q, got %q", tmpl.ID, contract.TemplateID)
	}
	if contract.VendorName != "Acme Corp" {
		t.Errorf("expected vendor 'Acme Corp', got %q", contract.VendorName)
	}
	if !strings.Contains(contract.Content, "Acme Corp") {
		t.Error("expected content to contain vendor name")
	}
	if !strings.Contains(contract.Content, "24 months") {
		t.Error("expected content to contain term substitution")
	}
	if !strings.Contains(contract.Content, "100") {
		t.Error("expected content to contain seats substitution")
	}

	found := false
	for _, v := range contract.VariablesUsed {
		if v == "term" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected variables_used to contain 'term', got %v", contract.VariablesUsed)
	}
}

func TestGenerateContract_MissingTemplate(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	eng := NewEngine(store, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	_, err = eng.GenerateContract(ctx, "nonexistent-id", "Vendor", nil)
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}
	if !strings.Contains(err.Error(), "template not found") {
		t.Errorf("expected 'template not found' error, got %q", err.Error())
	}
}
