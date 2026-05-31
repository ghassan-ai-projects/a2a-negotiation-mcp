package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", "file:wh_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	return store
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupTestEngine(t *testing.T) (*Engine, *Store) {
	t.Helper()
	store := setupTestStore(t)
	logger := testLogger(t)
	eng := NewEngine(store, logger)
	return eng, store
}
func setupTestEngineWithFastBackoff(t *testing.T) (*Engine, *Store) {
	t.Helper()
	store := setupTestStore(t)
	logger := testLogger(t)
	backoffs := []time.Duration{1 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}
	eng := NewEngineWithBackoff(store, logger, backoffs)
	return eng, store
}

// ─── Store Tests ───

func TestCreateSubscription(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub := &Subscription{
		URL:    "https://hooks.example.com/events",
		Events: []string{"negotiation.completed", "renewal.urgent"},
		Secret: "test-secret",
	}

	if err := store.Create(ctx, sub); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sub.Status != "active" {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	// Verify via Get
	got, err := store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.URL != "https://hooks.example.com/events" {
		t.Errorf("expected URL hooks.example.com, got %s", got.URL)
	}
	if len(got.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(got.Events))
	}
	if got.Events[0] != "negotiation.completed" {
		t.Errorf("expected event negotiation.completed, got %s", got.Events[0])
	}
	if got.Secret != "test-secret" {
		t.Errorf("expected secret test-secret, got %s", got.Secret)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
}

func TestGetSubscription_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent subscription")
	}
}

func TestListSubscriptions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub1 := &Subscription{URL: "https://hook1.example.com", Events: []string{"negotiation.completed"}, Secret: "s1"}
	sub2 := &Subscription{URL: "https://hook2.example.com", Events: []string{"renewal.urgent"}, Secret: "s2"}

	if err := store.Create(ctx, sub1); err != nil {
		t.Fatalf("Create sub1: %v", err)
	}
	if err := store.Create(ctx, sub2); err != nil {
		t.Fatalf("Create sub2: %v", err)
	}

	subs, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestListSubscriptions_FilterByStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub := &Subscription{URL: "https://hook1.example.com", Events: []string{"negotiation.completed"}, Secret: "s1"}
	if err := store.Create(ctx, sub); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Disable(ctx, sub.ID); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	// Should not appear in active list
	actives, err := store.List(ctx, "active")
	if err != nil {
		t.Fatalf("List active: %v", err)
	}
	if len(actives) != 0 {
		t.Errorf("expected 0 active, got %d", len(actives))
	}

	// Should appear in disabled list
	disabled, err := store.List(ctx, "disabled")
	if err != nil {
		t.Fatalf("List disabled: %v", err)
	}
	if len(disabled) != 1 {
		t.Errorf("expected 1 disabled, got %d", len(disabled))
	}
}

func TestDisableSubscription(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub := &Subscription{URL: "https://hook.example.com", Events: []string{"negotiation.completed"}, Secret: "s"}
	if err := store.Create(ctx, sub); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Disable(ctx, sub.ID); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	got, err := store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "disabled" {
		t.Errorf("expected status disabled, got %s", got.Status)
	}
}

func TestDisableSubscription_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.Disable(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent subscription")
	}
}

func TestGetByEvent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub1 := &Subscription{URL: "https://hook1.example.com", Events: []string{"negotiation.completed"}, Secret: "s1"}
	sub2 := &Subscription{URL: "https://hook2.example.com", Events: []string{"renewal.urgent"}, Secret: "s2"}
	sub3 := &Subscription{URL: "https://hook3.example.com", Events: []string{"negotiation.completed", "renewal.urgent"}, Secret: "s3"}

	for _, s := range []*Subscription{sub1, sub2, sub3} {
		if err := store.Create(ctx, s); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	// Should find 2 subscriptions for negotiation.completed (sub1, sub3)
	subs, err := store.GetByEvent(ctx, "negotiation.completed")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions for negotiation.completed, got %d", len(subs))
	}

	// Should find 2 subscriptions for renewal.urgent (sub2, sub3)
	subs, err = store.GetByEvent(ctx, "renewal.urgent")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions for renewal.urgent, got %d", len(subs))
	}

	// Should find 0 for deal.closed
	subs, err = store.GetByEvent(ctx, "deal.closed")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions for deal.closed, got %d", len(subs))
	}

	// Disabled subscriptions should not be returned
	if err := store.Disable(ctx, sub1.ID); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	subs, err = store.GetByEvent(ctx, "negotiation.completed")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 subscription after disable, got %d", len(subs))
	}
}

func TestGetByEvent_Wildcard(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sub := &Subscription{URL: "https://hook.example.com", Events: []string{"*"}, Secret: "s"}
	if err := store.Create(ctx, sub); err != nil {
		t.Fatalf("Create: %v", err)
	}

	subs, err := store.GetByEvent(ctx, "negotiation.completed")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 wildcard subscription, got %d", len(subs))
	}

	subs, err = store.GetByEvent(ctx, "deal.closed")
	if err != nil {
		t.Fatalf("GetByEvent: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 wildcard subscription for any event, got %d", len(subs))
	}
}

// ─── Engine Tests ───

func TestEngine_Register(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	sub, err := eng.Register(ctx, "https://hooks.example.com/events", []string{"negotiation.completed"}, "mysecret")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sub.URL != "https://hooks.example.com/events" {
		t.Errorf("expected URL hooks.example.com, got %s", sub.URL)
	}
	if len(sub.Events) != 1 || sub.Events[0] != "negotiation.completed" {
		t.Errorf("expected events [negotiation.completed], got %v", sub.Events)
	}
	if sub.Secret != "mysecret" {
		t.Errorf("expected secret mysecret, got %s", sub.Secret)
	}
	if sub.Status != "active" {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

func TestEngine_Register_NoURL(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.Register(ctx, "", []string{"negotiation.completed"}, "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestEngine_Register_NoEvents(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.Register(ctx, "https://hook.example.com", []string{}, "")
	if err == nil {
		t.Fatal("expected error for empty events")
	}
}

func TestEngine_Unregister(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	sub, err := eng.Register(ctx, "https://hooks.example.com", []string{"negotiation.completed"}, "s")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Unregister(ctx, sub.ID); err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	got, err := store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "disabled" {
		t.Errorf("expected status disabled after unregister, got %s", got.Status)
	}
}

func TestEngine_List(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Should return empty list initially
	subs, err := eng.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(subs))
	}

	_, err = eng.Register(ctx, "https://hook1.example.com", []string{"negotiation.completed"}, "s1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err = eng.Register(ctx, "https://hook2.example.com", []string{"renewal.urgent"}, "s2")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	subs, err = eng.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestEngine_List_ActiveOnly(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	sub, err := eng.Register(ctx, "https://hook.example.com", []string{"negotiation.completed"}, "s")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Unregister (disable)
	if err := eng.Unregister(ctx, sub.ID); err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	subs, err := eng.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 active subscriptions after unregister, got %d", len(subs))
	}
}

func TestEngine_Dispatch_FiresToCorrectURL(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	var receivedBody []byte
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{"vendor": "Slack", "final_price": 100.0}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		// Verify the body is a valid Event
		var event Event
		if err := json.Unmarshal(receivedBody, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if event.Type != "negotiation.completed" {
			t.Errorf("expected event type negotiation.completed, got %s", event.Type)
		}
		if event.ID == "" {
			t.Error("expected non-empty event ID")
		}
		if event.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestEngine_Dispatch_SignsPayloadWithHMAC(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	var receivedSignature string
	var receivedBody []byte
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	secret := "my-hmac-secret"
	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, secret)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{"vendor": "Slack"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		if receivedSignature == "" {
			t.Fatal("expected X-Webhook-Signature header")
		}
		// Verify the signature
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(receivedBody)
		expectedSig := hex.EncodeToString(mac.Sum(nil))
		if receivedSignature != expectedSig {
			t.Errorf("expected signature %s, got %s", expectedSig, receivedSignature)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestEngine_Dispatch_NoSignatureWhenNoSecret(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	var receivedSignature string
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		if receivedSignature != "" {
			t.Errorf("expected empty signature, got %s", receivedSignature)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestEngine_Dispatch_RetriesOn500(t *testing.T) {
	eng, _ := setupTestEngineWithFastBackoff(t)
	ctx := context.Background()

	attempts := make(chan int, 10)
	attemptCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		attempts <- attemptCount
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Dispatch should fail (all 4 attempts: initial + 3 retries)
	err = eng.Dispatch(ctx, "negotiation.completed", map[string]any{})
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}

	// Should have made 4 attempts (initial + 3 retries)
	close(attempts)
	var count int
	for range attempts {
		count++
	}
	if count != 4 {
		t.Errorf("expected 4 attempts (initial + 3 retries), got %d", count)
	}
}

func TestEngine_Dispatch_NoSubscribers(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Dispatch without any subscribers should not error
	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{"vendor": "Slack"}); err != nil {
		t.Errorf("expected nil error with no subscribers, got %v", err)
	}
}

func TestEngine_Dispatch_MultipleSubscribers(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	receivedCount := make(chan int, 2)
	var count int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		receivedCount <- count
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		receivedCount <- count
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err = eng.Register(ctx, srv2.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{"vendor": "Slack"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	close(receivedCount)
	var total int
	for range receivedCount {
		total++
	}
	if total != 2 {
		t.Errorf("expected 2 deliveries, got %d", total)
	}
}

func TestEngine_Dispatch_RespectsEventFilter(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Only register for renewal.urgent
	_, err := eng.Register(ctx, srv.URL, []string{"renewal.urgent"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Dispatch negotiation.completed - should NOT fire
	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		t.Fatal("should not have received dispatch for unsubscribed event")
	case <-time.After(500 * time.Millisecond):
		// Expected - no delivery
	}
}

func TestEngine_JSONResult(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	sub, err := eng.Register(ctx, "https://example.com/hook", []string{"deal.closed"}, "secret123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := eng.store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Verify JSON round-trip
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var restored Subscription
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if restored.ID != sub.ID {
		t.Errorf("expected ID %s, got %s", sub.ID, restored.ID)
	}
	if restored.URL != "https://example.com/hook" {
		t.Errorf("expected URL example.com/hook, got %s", restored.URL)
	}
	if restored.Secret != "secret123" {
		t.Errorf("expected secret secret123, got %s", restored.Secret)
	}
}

// Helper to test Dispatch directly without HTTP - verifying event body structure
func TestEngine_Dispatch_EventBodyStructure(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	var receivedBody []byte
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"deal.closed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	testData := map[string]any{
		"vendor": "Slack",
		"sku":    "pro-seat",
		"seats":  10,
		"total":  100.0,
		"buyer":  "acme-corp",
	}
	if err := eng.Dispatch(ctx, "deal.closed", testData); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		var event Event
		if err := json.Unmarshal(receivedBody, &event); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if event.Type != "deal.closed" {
			t.Errorf("expected type deal.closed, got %s", event.Type)
		}
		if event.ID == "" {
			t.Error("expected non-empty event id")
		}
		// Verify data payload
		dataMap, ok := event.Data.(map[string]any)
		if !ok {
			t.Fatal("expected Data to be a map")
		}
		if dataMap["vendor"] != "Slack" {
			t.Errorf("expected data.vendor Slack, got %v", dataMap["vendor"])
		}
		if dataMap["total"] != 100.0 {
			t.Errorf("expected data.total 100.0, got %v", dataMap["total"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestEngine_Dispatch_ContentType(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	var contentType string
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := eng.Register(ctx, srv.URL, []string{"negotiation.completed"}, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := eng.Dispatch(ctx, "negotiation.completed", map[string]any{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	select {
	case <-received:
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestEngine_MultipleEvents(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Register with multiple events
	events := []string{"negotiation.completed", "renewal.urgent", "deal.closed"}
	sub, err := eng.Register(ctx, "https://hook.example.com", events, "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if len(sub.Events) != 3 {
		t.Errorf("expected 3 events, got %d", len(sub.Events))
	}
}

func TestEngine_StoreAndEngineIndependence(t *testing.T) {
	// Verify that store and engine created separately work
	store := setupTestStore(t)
	ctx := context.Background()
	logger := testLogger(t)
	eng := NewEngine(store, logger)

	sub, err := eng.Register(ctx, "https://example.com/hook", []string{"negotiation.completed"}, "s")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Use store directly to retrieve
	got, err := store.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Store.Get: %v", err)
	}
	if got.URL != "https://example.com/hook" {
		t.Errorf("expected URL from store, got %s", got.URL)
	}
}
