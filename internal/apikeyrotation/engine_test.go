package apikeyrotation

import (
	"testing"
)

func TestAddKey_Success(t *testing.T) {
	eng := NewEngine()
	entry, err := eng.AddKey("alice")
	if err != nil {
		t.Fatalf("AddKey: %v", err)
	}
	if entry.KeyID == "" {
		t.Error("expected non-empty KeyID")
	}
	if entry.Owner != "alice" {
		t.Errorf("expected owner alice, got %s", entry.Owner)
	}
	if entry.Status != "active" {
		t.Errorf("expected status active, got %s", entry.Status)
	}
	if entry.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
	if entry.ExpiresAt == "" {
		t.Error("expected non-empty ExpiresAt")
	}
	if entry.LastRotated == "" {
		t.Error("expected non-empty LastRotated")
	}
}

func TestAddKey_EmptyOwner(t *testing.T) {
	eng := NewEngine()
	_, err := eng.AddKey("")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
}

func TestRotateKey_Success(t *testing.T) {
	eng := NewEngine()
	original, err := eng.AddKey("bob")
	if err != nil {
		t.Fatalf("AddKey: %v", err)
	}

	result, err := eng.RotateKey(original.KeyID)
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	if result.OldKeyID != original.KeyID {
		t.Errorf("expected OldKeyID %s, got %s", original.KeyID, result.OldKeyID)
	}
	if result.NewKeyID == "" {
		t.Error("expected non-empty NewKeyID")
	}
	if result.NewKeyID == original.KeyID {
		t.Error("NewKeyID should differ from OldKeyID")
	}
	if result.Status != "rotated" {
		t.Errorf("expected status rotated, got %s", result.Status)
	}
}

func TestRotateKey_NotFound(t *testing.T) {
	eng := NewEngine()
	_, err := eng.RotateKey("nonexistent-key")
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
}

func TestRotateKey_EmptyKeyID(t *testing.T) {
	eng := NewEngine()
	_, err := eng.RotateKey("")
	if err == nil {
		t.Fatal("expected error for empty key_id")
	}
}

func TestRotateKey_RevokedKey(t *testing.T) {
	eng := NewEngine()
	entry, err := eng.AddKey("charlie")
	if err != nil {
		t.Fatalf("AddKey: %v", err)
	}

	// First rotation revokes the original
	_, err = eng.RotateKey(entry.KeyID)
	if err != nil {
		t.Fatalf("first RotateKey: %v", err)
	}

	// Second rotation on the same (now revoked) key should fail
	_, err = eng.RotateKey(entry.KeyID)
	if err == nil {
		t.Fatal("expected error when rotating a revoked key")
	}
}

func TestKeyHealth_ReturnsAllKeys(t *testing.T) {
	eng := NewEngine()
	_, _ = eng.AddKey("alice")
	_, _ = eng.AddKey("bob")

	keys, err := eng.KeyHealth()
	if err != nil {
		t.Fatalf("KeyHealth: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestKeyHealth_AfterRotation(t *testing.T) {
	eng := NewEngine()
	entry, _ := eng.AddKey("dave")
	_, _ = eng.RotateKey(entry.KeyID)

	keys, err := eng.KeyHealth()
	if err != nil {
		t.Fatalf("KeyHealth: %v", err)
	}

	// Should have 2 keys: original (revoked) + new (active)
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	// Verify the original is revoked
	var found bool
	for _, k := range keys {
		if k.KeyID == entry.KeyID {
			found = true
			if k.Status != "revoked" {
				t.Errorf("expected original key status revoked, got %s", k.Status)
			}
			break
		}
	}
	if !found {
		t.Error("original key not found in key health list")
	}
}
