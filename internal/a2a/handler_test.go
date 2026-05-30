package a2a

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	_ "modernc.org/sqlite"
)

func setupA2ATest(t *testing.T) (*A2AHandler, *MandateStore, *pricing.Store, *history.Store) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	pricingStore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pricingStore.Close() })

	historyStore, err := history.NewStore(pricingStore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	mandateStore, err := NewMandateStore(pricingStore.DB())
	if err != nil {
		t.Fatalf("NewMandateStore: %v", err)
	}

	handler := NewA2AHandler(pricingStore, historyStore, mandateStore, logger, "http://localhost:8080")
	return handler, mandateStore, pricingStore, historyStore
}

func seedVendor(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"INSERT INTO vendors (name, category) VALUES ('Slack', 'messaging')")
	if err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
		VALUES (1, 'Pro', 'Slack Pro plan', 8.75, 6.50, 8.75, 15, 'per_seat_month')`)
	if err != nil {
		t.Fatalf("seed pricing: %v", err)
	}
}

func TestHandleTask_QueryPrice(t *testing.T) {
	handler, _, ps, _ := setupA2ATest(t)
	seedVendor(t, ps.DB())

	body, _ := json.Marshal(TaskRequest{
		Task:   "query_price",
		Params: map[string]any{"vendor": "Slack", "sku": "Pro"},
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/task", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleTask(w, req)

	var resp TaskResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
	if resp.Result == nil {
		t.Fatal("Result is nil")
	}
	vendor, _ := resp.Result["vendor"].(string)
	if vendor != "Slack" {
		t.Errorf("vendor = %q, want %q", vendor, "Slack")
	}
}

func TestHandleTask_QueryPrice_MissingVendor(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	body, _ := json.Marshal(TaskRequest{
		Task:   "query_price",
		Params: map[string]any{},
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/task", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleTask(w, req)

	var resp TaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "failed" {
		t.Errorf("Status = %q, want %q", resp.Status, "failed")
	}
}

func TestHandleTask_UnknownTask(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	body, _ := json.Marshal(TaskRequest{Task: "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/a2a/task", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleTask(w, req)

	var resp TaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "failed" {
		t.Errorf("Status = %q, want %q", resp.Status, "failed")
	}
}

func TestHandleGetTask(t *testing.T) {
	handler, _, ps, hs := setupA2ATest(t)
	seedVendor(t, ps.DB())

	// Create a session first
	sess := &history.SessionRecord{
		ID: "test-session-1", Vendor: "Slack", SKU: "Pro",
		Strategy: "balanced", Status: "active",
		CurrentOffer: 7.00, ListPrice: 8.75,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := hs.SaveSession(context.Background(), sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/a2a/task/test-session-1", nil)
	req.SetPathValue("id", "test-session-1")
	w := httptest.NewRecorder()

	handler.HandleGetTask(w, req)

	var resp TaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "active" {
		t.Errorf("Status = %q, want %q", resp.Status, "active")
	}
}

func TestHandleNegotiate(t *testing.T) {
	handler, _, ps, _ := setupA2ATest(t)
	seedVendor(t, ps.DB())

	body, _ := json.Marshal(NegotiateRequest{
		Vendor:   "Slack",
		SKU:      "Pro",
		Strategy: "balanced",
		Budget:   7.00,
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/negotiate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleNegotiate(w, req)

	var resp NegotiateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status == "" {
		t.Fatal("empty status")
	}
	if resp.MandateID == "" {
		t.Error("MandateID should not be empty")
	}
	if resp.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
	if resp.Mandate == nil {
		t.Error("Mandate should not be nil")
	}
	if resp.ListPrice != 8.75 {
		t.Errorf("ListPrice = %f, want 8.75", resp.ListPrice)
	}
}

func TestHandleNegotiate_MissingVendor(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	body, _ := json.Marshal(NegotiateRequest{
		Vendor:   "",
		Strategy: "balanced",
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/negotiate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleNegotiate(w, req)

	var resp NegotiateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "error" {
		t.Errorf("Status = %q, want %q", resp.Status, "error")
	}
}

func TestHandleNegotiate_InvalidVendor(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	body, _ := json.Marshal(NegotiateRequest{
		Vendor:   "NonExistentVendor",
		SKU:      "Pro",
		Strategy: "balanced",
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/negotiate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.HandleNegotiate(w, req)

	var resp NegotiateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	// Should create a mandate but fail on session creation -> still get a response
	if resp.MandateID == "" {
		t.Error("MandateID should not be empty even on session failure")
	}
}

func TestHandleAgentCard(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	w := httptest.NewRecorder()

	handler.HandleAgentCard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	var card AgentCard
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatalf("decode agent card: %v", err)
	}
	if card.Name == "" {
		t.Error("Name should not be empty")
	}
}

func TestHandleTask_InvalidMethod(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	req := httptest.NewRequest(http.MethodGet, "/a2a/task", nil)
	w := httptest.NewRecorder()

	handler.HandleTask(w, req)

	var resp TaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == "" {
		t.Error("expected error for invalid method")
	}
}

func TestHandleTask_InvalidBody(t *testing.T) {
	handler, _, _, _ := setupA2ATest(t)

	req := httptest.NewRequest(http.MethodPost, "/a2a/task", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handler.HandleTask(w, req)

	var resp TaskResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == "" {
		t.Error("expected error for invalid JSON")
	}
}
