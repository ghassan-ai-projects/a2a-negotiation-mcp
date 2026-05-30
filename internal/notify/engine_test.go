package notify

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func setupNotifyTest(t *testing.T) (*Engine, *Store) {
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
	engine := NewEngine(store, logger)

	return engine, store
}

// ─── Tests ───

func TestSetPreferences(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	prefs, err := engine.SetPreferences(ctx, "webhook", []string{"deal_closed", "alert"}, "daily", "https://hooks.example.com/notify")
	if err != nil {
		t.Fatalf("SetPreferences: %v", err)
	}

	if prefs.Channel != "webhook" {
		t.Errorf("expected channel webhook, got %s", prefs.Channel)
	}
	if len(prefs.EnabledTypes) != 2 {
		t.Errorf("expected 2 enabled types, got %d", len(prefs.EnabledTypes))
	}
	if prefs.DigestFreq != "daily" {
		t.Errorf("expected digest daily, got %s", prefs.DigestFreq)
	}
	if prefs.WebhookURL != "https://hooks.example.com/notify" {
		t.Errorf("expected webhook URL, got %s", prefs.WebhookURL)
	}
}

func TestGetPreferences_ReturnsSetValues(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	_, err := engine.SetPreferences(ctx, "webhook", []string{"deal_closed"}, "weekly", "https://hooks.example.com/notify")
	if err != nil {
		t.Fatalf("SetPreferences: %v", err)
	}

	prefs, err := engine.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}

	if prefs.DigestFreq != "weekly" {
		t.Errorf("expected digest weekly, got %s", prefs.DigestFreq)
	}
	if len(prefs.EnabledTypes) != 1 || prefs.EnabledTypes[0] != "deal_closed" {
		t.Errorf("expected [deal_closed], got %v", prefs.EnabledTypes)
	}
}

func TestGetPreferences_ReturnsDefaults(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	prefs, err := engine.GetPreferences(ctx)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}

	if prefs.Channel != "webhook" {
		t.Errorf("expected default channel webhook, got %s", prefs.Channel)
	}
	if prefs.DigestFreq != "never" {
		t.Errorf("expected default digest never, got %s", prefs.DigestFreq)
	}
}

func TestSendNotification_LogsNotification(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	// Set up slack preferences
	_, err := engine.SetPreferences(ctx, "slack", []string{"deal_closed"}, "never", "")
	if err != nil {
		t.Fatalf("SetPreferences: %v", err)
	}

	n, err := engine.SendNotification(ctx, "deal_closed", "A deal was closed", "high")
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	if n.ID == 0 {
		t.Error("expected non-zero notification ID")
	}
	if n.Type != "deal_closed" {
		t.Errorf("expected type deal_closed, got %s", n.Type)
	}
	if n.Priority != "high" {
		t.Errorf("expected priority high, got %s", n.Priority)
	}
}

func TestSendNotification_DefaultPriority(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	_, err := engine.SetPreferences(ctx, "slack", []string{"alert"}, "never", "")
	if err != nil {
		t.Fatalf("SetPreferences: %v", err)
	}

	n, err := engine.SendNotification(ctx, "alert", "Test alert", "")
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	if n.Priority != "normal" {
		t.Errorf("expected default priority normal, got %s", n.Priority)
	}
}

func TestSetPreferences_InvalidChannel(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	_, err := engine.SetPreferences(ctx, "sms", []string{}, "never", "")
	if err == nil {
		t.Error("expected error for invalid channel")
	}
}

func TestSetPreferences_InvalidDigestFreq(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	_, err := engine.SetPreferences(ctx, "webhook", []string{}, "yearly", "https://hooks.example.com")
	if err == nil {
		t.Error("expected error for invalid digest frequency")
	}
}

func TestSetPreferences_WebhookRequiresURL(t *testing.T) {
	engine, _ := setupNotifyTest(t)
	ctx := context.Background()

	_, err := engine.SetPreferences(ctx, "webhook", []string{}, "never", "")
	if err == nil {
		t.Error("expected error when webhook URL is empty")
	}
}
