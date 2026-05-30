package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/google/uuid"
)

// A2AHandler holds dependencies for A2A HTTP handlers.
type A2AHandler struct {
	pricingStore   *pricing.Store
	historyStore   *history.Store
	mandateStore   *MandateStore
	negotiationEng *negotiation.Engine
	logger         *slog.Logger
	baseURL        string
}

// NewA2AHandler creates a new A2AHandler.
func NewA2AHandler(pricingStore *pricing.Store, historyStore *history.Store, mandateStore *MandateStore, logger *slog.Logger, baseURL string) *A2AHandler {
	return &A2AHandler{
		pricingStore:   pricingStore,
		historyStore:   historyStore,
		mandateStore:   mandateStore,
		negotiationEng: negotiation.NewEngine(pricingStore),
		logger:         logger,
		baseURL:        baseURL,
	}
}

// HandleTask handles POST /a2a/task — dispatches to sub-handlers by task type.
func (h *A2AHandler) HandleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, TaskResponse{Error: "method not allowed"})
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, TaskResponse{Error: fmt.Sprintf("invalid request body: %s", err.Error())})
		return
	}

	taskID := uuid.New().String()

	switch req.Task {
	case "query_price":
		h.handleQueryPriceTask(w, r.Context(), taskID, req.Params)
	case "mandate_create":
		h.handleMandateCreateTask(w, r.Context(), taskID, req.Params)
	case "mandate_settle":
		h.handleMandateSettleTask(w, r.Context(), taskID, req.Params)
	case "mandate_cancel":
		h.handleMandateCancelTask(w, r.Context(), taskID, req.Params)
	default:
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID,
			Status: "failed",
			Error:  fmt.Sprintf("unknown task: %s", req.Task),
		})
	}
}

// HandleGetTask handles GET /a2a/task/{id} — returns session status.
func (h *A2AHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, TaskResponse{Error: "missing task id"})
		return
	}

	sess, err := h.historyStore.GetSession(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, TaskResponse{
			TaskID: sessionID,
			Status: "not_found",
			Error:  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, TaskResponse{
		TaskID:    sessionID,
		Status:    sess.Status,
		SessionID: sess.ID,
		Result: map[string]any{
			"vendor":            sess.Vendor,
			"sku":               sess.SKU,
			"strategy":          sess.Strategy,
			"current_offer":     sess.CurrentOffer,
			"list_price":        sess.ListPrice,
			"rounds_completed":  sess.RoundsComplete,
			"outcome":           sess.Outcome,
			"created_at":        sess.CreatedAt.Format(time.RFC3339),
			"updated_at":        sess.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// HandleNegotiate handles POST /a2a/negotiate — creates mandate + session + runs negotiation.
func (h *A2AHandler) HandleNegotiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, NegotiateResponse{Status: "error"})
		return
	}

	var req NegotiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NegotiateResponse{
			Status: "error",
			Result: map[string]any{"error": fmt.Sprintf("invalid request: %s", err.Error())},
		})
		return
	}

	if req.Vendor == "" || req.Strategy == "" {
		writeJSON(w, http.StatusBadRequest, NegotiateResponse{
			Status: "error",
			Result: map[string]any{"error": "vendor and strategy are required"},
		})
		return
	}

	ctx := r.Context()
	taskID := uuid.New().String()
	now := time.Now().UTC()

	// 1. Create mandate
	mandate := &Mandate{
		ID:        uuid.New().String(),
		Type:      "intent",
		Principal: "agent-external",
		AgentID:   "a2a-negotiation-mcp",
		Status:    "active",
		ExpiresAt: now.Add(1 * time.Hour),
		Terms: map[string]any{
			"vendor":   req.Vendor,
			"sku":      req.SKU,
			"strategy": req.Strategy,
			"budget":   req.Budget,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.mandateStore.CreateMandate(ctx, mandate); err != nil {
		h.logger.Error("failed to create mandate", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, NegotiateResponse{
			Status: "error",
			Result: map[string]any{"error": "failed to create mandate"},
		})
		return
	}

	// 2. Create negotiation session
	session, err := h.negotiationEng.CreateSession(ctx, req.Vendor, req.SKU, req.Strategy, req.Budget, req.Terms)
	if err != nil {
		h.logger.Warn("failed to create session", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, NegotiateResponse{
			MandateID: mandate.ID,
			Status:    "error",
			Result:    map[string]any{"error": err.Error()},
		})
		return
	}
	session.ID = uuid.New().String()

	// 3. Save session to history
	histSess := &history.SessionRecord{
		ID: session.ID, Vendor: session.Vendor, SKU: session.SKU,
		Strategy: session.Strategy, Budget: session.Budget, Status: session.Status,
		CurrentOffer: session.CurrentOffer, ListPrice: session.ListPrice,
		RoundsComplete: session.RoundsComplete, Outcome: session.Outcome,
		CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt,
	}
	if err := h.historyStore.SaveSession(ctx, histSess); err != nil {
		h.logger.Error("failed to save session", "error", err.Error())
	}

	// 4. Run 1 round of negotiation to get a counter-offer
	result, rounds, err := h.negotiationEng.RunNegotiation(ctx, session, 1, 0)
	if err != nil {
		h.logger.Warn("negotiation run failed", "error", err.Error())
	}

	// 5. Update session in history
	if result != nil {
		histSess.Status = session.Status
		histSess.CurrentOffer = session.CurrentOffer
		histSess.RoundsComplete = session.RoundsComplete
		histSess.Outcome = session.Outcome
		histSess.UpdatedAt = session.UpdatedAt
		if err := h.historyStore.UpdateSession(ctx, histSess); err != nil {
			h.logger.Error("failed to update session", "error", err.Error())
		}

		var roundRecs []history.RoundRecord
		for _, r := range rounds {
			roundRecs = append(roundRecs, history.RoundRecord{
				SessionID: r.SessionID, RoundNumber: r.RoundNumber, Offer: r.Offer,
				DiscountPct: r.DiscountPct, Counterparty: r.Counterparty, Note: r.Note,
				CreatedAt: r.CreatedAt,
			})
		}
		if err := h.historyStore.SaveRounds(ctx, roundRecs); err != nil {
			h.logger.Error("failed to save rounds", "error", err.Error())
		}
	}

	writeJSON(w, http.StatusOK, NegotiateResponse{
		MandateID: mandate.ID,
		TaskID:    taskID,
		SessionID: session.ID,
		Status:    session.Status,
		Offer:     session.CurrentOffer,
		ListPrice: session.ListPrice,
		Strategy:  session.Strategy,
		Mandate:   mandate,
		Result: map[string]any{
			"outcome":          session.Outcome,
			"rounds_completed": session.RoundsComplete,
		},
	})
}

// HandleAgentCard handles GET /.well-known/agent-card.json.
func (h *A2AHandler) HandleAgentCard(w http.ResponseWriter, r *http.Request) {
	cardJSON, err := AgentCardJSON(h.baseURL)
	if err != nil {
		h.logger.Error("failed to generate agent card", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate agent card"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(cardJSON))
}

// ─── Task sub-handlers ───

func (h *A2AHandler) handleQueryPriceTask(w http.ResponseWriter, ctx context.Context, taskID string, params map[string]any) {
	vendor, _ := params["vendor"].(string)
	sku, _ := params["sku"].(string)

	if vendor == "" {
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: "vendor is required",
		})
		return
	}

	result, err := h.pricingStore.GetPricingByVendorSKU(ctx, vendor, sku)
	if err != nil {
		h.logger.Warn("query_price task failed", "vendor", vendor, "error", err.Error())
		writeJSON(w, http.StatusNotFound, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: err.Error(),
		})
		return
	}

	// Create a lightweight session to track this query
	sess := &history.SessionRecord{
		ID: taskID, Vendor: vendor, SKU: sku,
		Strategy: "query", Status: "completed",
		CurrentOffer: result.SuggestedMin, ListPrice: result.ListPrice,
		Outcome: "queried", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := h.historyStore.SaveSession(ctx, sess); err != nil {
		h.logger.Error("failed to save query session", "error", err.Error())
	}

	writeJSON(w, http.StatusOK, TaskResponse{
		TaskID:    taskID,
		Status:    "completed",
		SessionID: taskID,
		Result: map[string]any{
			"vendor":                  result.Vendor,
			"sku":                     result.SKU,
			"list_price":              result.ListPrice,
			"market_min":              result.MarketMin,
			"market_max":              result.MarketMax,
			"suggested_min":           result.SuggestedMin,
			"suggested_max":           result.SuggestedMax,
			"confidence":              result.Confidence,
			"typical_discount_pct":    result.TypicalPct,
		},
	})
}

func (h *A2AHandler) handleMandateCreateTask(w http.ResponseWriter, ctx context.Context, taskID string, params map[string]any) {
	mandateType, _ := params["type"].(string)
	principal, _ := params["principal"].(string)

	if mandateType == "" || principal == "" {
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: "type and principal are required",
		})
		return
	}

	terms, _ := params["terms"].(map[string]any)
	now := time.Now().UTC()

	mandate := &Mandate{
		ID:        uuid.New().String(),
		Type:      mandateType,
		Principal: principal,
		AgentID:   "a2a-negotiation-mcp",
		Status:    "pending",
		ExpiresAt: now.Add(24 * time.Hour),
		Terms:     terms,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.mandateStore.CreateMandate(ctx, mandate); err != nil {
		h.logger.Error("failed to create mandate", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: "failed to create mandate",
		})
		return
	}

	writeJSON(w, http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "completed",
		Result: map[string]any{
			"mandate_id": mandate.ID,
			"type":       mandate.Type,
			"principal":  mandate.Principal,
			"status":     mandate.Status,
			"expires_at": mandate.ExpiresAt.Format(time.RFC3339),
		},
	})
}

func (h *A2AHandler) handleMandateSettleTask(w http.ResponseWriter, ctx context.Context, taskID string, params map[string]any) {
	mandateID, _ := params["mandate_id"].(string)
	if mandateID == "" {
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: "mandate_id is required",
		})
		return
	}

	if err := h.mandateStore.SettleMandate(ctx, mandateID); err != nil {
		h.logger.Warn("settle mandate failed", "mandate_id", mandateID, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "completed",
		Result: map[string]any{
			"mandate_id": mandateID,
			"status":     "settled",
		},
	})
}

func (h *A2AHandler) handleMandateCancelTask(w http.ResponseWriter, ctx context.Context, taskID string, params map[string]any) {
	mandateID, _ := params["mandate_id"].(string)
	if mandateID == "" {
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: "mandate_id is required",
		})
		return
	}

	if err := h.mandateStore.CancelMandate(ctx, mandateID); err != nil {
		h.logger.Warn("cancel mandate failed", "mandate_id", mandateID, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, TaskResponse{
			TaskID: taskID, Status: "failed",
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, TaskResponse{
		TaskID: taskID,
		Status: "completed",
		Result: map[string]any{
			"mandate_id": mandateID,
			"status":     "cancelled",
		},
	})
}

// ─── Helpers ───

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
