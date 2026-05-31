package pushnotif

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

func TestRegisterDevice(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	dev, err := s.RegisterDevice(ctx, "token-abc-123", "ios")
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	if dev.Token != "token-abc-123" {
		t.Errorf("expected token 'token-abc-123', got %s", dev.Token)
	}
	if dev.Platform != "ios" {
		t.Errorf("expected platform 'ios', got %s", dev.Platform)
	}
	if dev.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if dev.CreatedAt == "" {
		t.Errorf("expected non-empty created_at")
	}
}

func TestRegisterDevice_DuplicateToken(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	first, err := s.RegisterDevice(ctx, "duplicate-token", "android")
	if err != nil {
		t.Fatalf("RegisterDevice (first): %v", err)
	}

	second, err := s.RegisterDevice(ctx, "duplicate-token", "android")
	if err != nil {
		t.Fatalf("RegisterDevice (second): %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same device ID for duplicate token, got first=%d second=%d", first.ID, second.ID)
	}
	if first.Token != second.Token {
		t.Errorf("expected same token, got first=%s second=%s", first.Token, second.Token)
	}
}

func TestListDevices(t *testing.T) {
	s := setupTest(t)
	ctx := context.Background()

	// Initially empty
	devices, err := s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}

	// Register a device
	_, err = s.RegisterDevice(ctx, "token-1", "ios")
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	devices, err = s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}

	// Register another
	_, err = s.RegisterDevice(ctx, "token-2", "android")
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}

	devices, err = s.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}
