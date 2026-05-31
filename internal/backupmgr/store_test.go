package backupmgr

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

func TestCreateBackup(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	backup, err := store.CreateBackup(ctx, "users,orders")
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if backup.ID <= 0 {
		t.Errorf("expected positive ID, got %d", backup.ID)
	}
	if backup.Tables != "users,orders" {
		t.Errorf("expected tables 'users,orders', got %q", backup.Tables)
	}
	if backup.SizeBytes <= 0 {
		t.Errorf("expected positive size_bytes, got %d", backup.SizeBytes)
	}
	if backup.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", backup.Status)
	}
	if backup.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestCreateBackup_Defaults(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	backup, err := store.CreateBackup(ctx, "all")
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if backup.Tables != "all" {
		t.Errorf("expected tables 'all', got %q", backup.Tables)
	}
	if backup.SizeBytes < 1024 || backup.SizeBytes > 10*1024*1024 {
		t.Errorf("expected size between 1KB and 10MB, got %d", backup.SizeBytes)
	}
	if backup.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", backup.Status)
	}
}

func TestListBackups(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	// Empty list should return empty slice
	backups, err := store.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups (empty): %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}

	// Create two backups
	b1, err := store.CreateBackup(ctx, "users")
	if err != nil {
		t.Fatalf("CreateBackup 1: %v", err)
	}
	b2, err := store.CreateBackup(ctx, "orders,products")
	if err != nil {
		t.Fatalf("CreateBackup 2: %v", err)
	}

	backups, err = store.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 2 {
		t.Errorf("expected 2 backups, got %d", len(backups))
	}
	// Both backups present regardless of ordering
	if backups[0].ID != b1.ID && backups[0].ID != b2.ID {
		t.Errorf("first backup has unexpected ID %d, expected %d or %d", backups[0].ID, b1.ID, b2.ID)
	}
	if backups[1].ID != b1.ID && backups[1].ID != b2.ID {
		t.Errorf("second backup has unexpected ID %d, expected %d or %d", backups[1].ID, b1.ID, b2.ID)
	}
	if backups[0].ID == backups[1].ID {
		t.Errorf("expected two different backups, got same ID %d", backups[0].ID)
	}
}

func TestRestoreBackup(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	backup, err := store.CreateBackup(ctx, "users,orders")
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	restored, err := store.RestoreBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}

	if restored.ID != backup.ID {
		t.Errorf("expected ID %d, got %d", backup.ID, restored.ID)
	}
	if restored.Tables != backup.Tables {
		t.Errorf("expected tables %q, got %q", backup.Tables, restored.Tables)
	}
	if restored.Status != "restored" {
		t.Errorf("expected status 'restored', got %q", restored.Status)
	}
	if restored.SizeBytes != backup.SizeBytes {
		t.Errorf("expected size_bytes %d, got %d", backup.SizeBytes, restored.SizeBytes)
	}
}

func TestRestoreBackup_NotFound(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	_, err := store.RestoreBackup(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent backup, got nil")
	}
}

func TestSetBackupSchedule(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()

	err := store.SetBackupSchedule(ctx, "0 2 * * *", "all")
	if err != nil {
		t.Fatalf("SetBackupSchedule: %v", err)
	}

	// Update existing schedule
	err = store.SetBackupSchedule(ctx, "0 3 * * *", "users,orders")
	if err != nil {
		t.Fatalf("SetBackupSchedule update: %v", err)
	}
}
