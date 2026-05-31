package qrshare

import (
	"context"
	"testing"
)

func TestGenerateQR_ValidSession(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.GenerateQR(ctx, "session-123")
	if err != nil {
		t.Fatalf("GenerateQR failed: %v", err)
	}

	if result.SessionID != "session-123" {
		t.Errorf("expected session ID 'session-123', got %q", result.SessionID)
	}
	if result.QRData == "" {
		t.Error("expected non-empty QR data")
	}
	if result.Format != "png" {
		t.Errorf("expected format 'png', got %q", result.Format)
	}
	if result.Description != "QR code for session session-123" {
		t.Errorf("unexpected description: %q", result.Description)
	}
}

func TestGenerateQR_EmptySession(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	_, err := eng.GenerateQR(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty session ID, got nil")
	}
}
