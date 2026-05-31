package pushnotif

import (
	"context"
	"testing"
)

func TestSendPush(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.SendPush(ctx, 1, "Test Title", "Test body content")
	if err != nil {
		t.Fatalf("SendPush: %v", err)
	}

	if result.DeviceID != 1 {
		t.Errorf("expected device ID 1, got %d", result.DeviceID)
	}
	if result.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %s", result.Title)
	}
	if result.Body != "Test body content" {
		t.Errorf("expected body 'Test body content', got %s", result.Body)
	}
	if result.Status != "sent" {
		t.Errorf("expected status 'sent', got %s", result.Status)
	}
}

func TestSendPush_DifferentDevice(t *testing.T) {
	eng := NewEngine()
	ctx := context.Background()

	result, err := eng.SendPush(ctx, 42, "Alert", "Something happened")
	if err != nil {
		t.Fatalf("SendPush: %v", err)
	}

	if result.DeviceID != 42 {
		t.Errorf("expected device ID 42, got %d", result.DeviceID)
	}
	if result.Status != "sent" {
		t.Errorf("expected status 'sent', got %s", result.Status)
	}
}
