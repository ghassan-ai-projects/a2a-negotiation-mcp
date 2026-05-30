package a2a

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey_CreatesValidUUID(t *testing.T) {
	store := NewAPIKeyStore()
	key, err := store.GenerateKey("test-owner")
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if key == "" {
		t.Fatal("key should not be empty")
	}
	if len(key) != 36 { // UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		t.Errorf("key length = %d, want 36", len(key))
	}
}

func TestGenerateKey_EmptyOwner(t *testing.T) {
	store := NewAPIKeyStore()
	_, err := store.GenerateKey("")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
}

func TestValidateKey_ValidKey(t *testing.T) {
	store := NewAPIKeyStore()
	key, err := store.GenerateKey("alice")
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	owner, ok := store.ValidateKey(key)
	if !ok {
		t.Fatal("ValidateKey returned false for valid key")
	}
	if owner != "alice" {
		t.Errorf("owner = %q, want %q", owner, "alice")
	}
}

func TestValidateKey_InvalidKey(t *testing.T) {
	store := NewAPIKeyStore()
	_, ok := store.ValidateKey("nonexistent-key")
	if ok {
		t.Fatal("ValidateKey returned true for invalid key")
	}
}

func TestValidateKey_EmptyKey(t *testing.T) {
	store := NewAPIKeyStore()
	_, ok := store.ValidateKey("")
	if ok {
		t.Fatal("ValidateKey returned true for empty key")
	}
}

func TestKeyCount(t *testing.T) {
	store := NewAPIKeyStore()
	if n := store.KeyCount(); n != 0 {
		t.Errorf("KeyCount = %d, want 0", n)
	}
	store.GenerateKey("a")
	store.GenerateKey("b")
	store.GenerateKey("c")
	if n := store.KeyCount(); n != 3 {
		t.Errorf("KeyCount = %d, want 3", n)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	// Create a keys file
	content := `{"key1": "owner1", "key2": "owner2"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewAPIKeyStore()
	if err := store.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	owner, ok := store.ValidateKey("key1")
	if !ok {
		t.Fatal("ValidateKey(key1) returned false")
	}
	if owner != "owner1" {
		t.Errorf("owner = %q, want %q", owner, "owner1")
	}

	owner, ok = store.ValidateKey("key2")
	if !ok {
		t.Fatal("ValidateKey(key2) returned false")
	}
	if owner != "owner2" {
		t.Errorf("owner = %q, want %q", owner, "owner2")
	}

	if n := store.KeyCount(); n != 2 {
		t.Errorf("KeyCount = %d, want 2", n)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	store := NewAPIKeyStore()
	err := store.LoadFromFile("/nonexistent/path/keys.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestSaveToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	// First load to set the file path, then add keys, then save
	content := `{}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewAPIKeyStore()
	if err := store.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	store.GenerateKey("alice")
	store.GenerateKey("bob")

	if err := store.SaveToFile(); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	// Read back and verify
	store2 := NewAPIKeyStore()
	if err := store2.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile on saved file: %v", err)
	}

	if n := store2.KeyCount(); n != 2 {
		t.Errorf("KeyCount after save = %d, want 2", n)
	}
}

func TestSaveToFile_NoPath(t *testing.T) {
	store := NewAPIKeyStore()
	err := store.SaveToFile()
	if err == nil {
		t.Fatal("expected error when no file path configured")
	}
}
