package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/group"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/health"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/negotiation"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sell"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/sla"
	"github.com/mark3labs/mcp-go/mcp"
)

// ─── Test setup helpers ───

func setupTest(t *testing.T) *NegotiationServer {
	t.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		t.Fatalf("NewInMemoryStore: %v", err)
	}
	t.Cleanup(func() { pstore.Close() })

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("history.NewStore: %v", err)
	}

	seedPricingData(t, pstore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gstore, err := group.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("group.NewStore: %v", err)
	}
	geng := group.NewEngine(gstore, pstore, logger)

	sstore, err := sell.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("sell.NewStore: %v", err)
	}
	seng := sell.NewEngine(sstore, logger)

	negEng := negotiation.NewEngine(pstore)
	cstore, err := calendar.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("calendar.NewStore: %v", err)
	}
	ceng := calendar.NewEngine(cstore, negEng, hstore, pstore, logger)

	hstore2, err := health.NewStoreFromDB(pstore.DB())
	if err != nil {
		t.Fatalf("health.NewStoreFromDB: %v", err)
	}
	heng := health.NewEngine(hstore2, logger)

	slastore, err := sla.NewStore(pstore.DB())
	if err != nil {
		t.Fatalf("sla.NewStore: %v", err)
	}
	slaEng := sla.NewEngine(slastore, logger)

	return NewNegotiationServer(pstore, hstore, geng, seng, ceng, heng, nil, slaEng, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
}

func seedPricingData(t *testing.T, store *pricing.Store) {
	t.Helper()
	ctx := context.Background()

	vendors := []struct {
		name, category, sku, desc string
		listPrice, minObs, maxObs float64
		typicalPct                float64
		unit                      string
	}{
		{"Slack", "Communication", "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month"},
		{"GitHub", "Developer", "Team", "Team plan", 4.00, 3.00, 3.80, 15, "per_seat_month"},
		{"Salesforce", "CRM", "Enterprise", "Enterprise per seat", 165.00, 110.00, 155.00, 28, "per_seat_month"},
		{"OpenAI", "ai", "gpt-4o", "OpenAI GPT-4o", 10.00, 2.50, 10.00, 15, "/1M_input_tokens"},
		{"OpenAI", "ai", "gpt-4o-mini", "OpenAI GPT-4o-mini", 0.60, 0.15, 0.60, 15, "/1M_input_tokens"},
		{"OpenAI", "ai", "o1", "OpenAI o1", 60.00, 15.00, 60.00, 15, "/1M_input_tokens"},
		{"Anthropic", "ai", "claude-3.5-sonnet", "Anthropic Claude 3.5 Sonnet", 15.00, 3.00, 15.00, 15, "/1M_input_tokens"},
		{"Google", "ai", "gemini-2.5-flash", "Google Gemini 2.5 Flash", 0.60, 0.15, 0.60, 15, "/1M_input_tokens"},
		{"DeepSeek", "ai", "deepseek-chat", "DeepSeek Chat", 1.10, 0.27, 1.10, 15, "/1M_input_tokens"},
		{"Mistral", "ai", "mistral-small", "Mistral Small", 0.60, 0.20, 0.60, 15, "/1M_input_tokens"},
		{"OpenAI", "ai", "dall-e-3", "OpenAI DALL-E 3", 0.120, 0.040, 0.120, 15, "/image"},


	}

	for _, v := range vendors {
		_, err := store.DB().ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
			v.name, v.category)
		if err != nil {
			t.Fatalf("insert vendor %s: %v", v.name, err)
		}
		var vid int64
		err = store.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", v.name).Scan(&vid)
		if err != nil {
			t.Fatalf("get vendor id %s: %v", v.name, err)
		}
		_, err = store.DB().ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET list_price=excluded.list_price
		`, vid, v.sku, v.desc, v.listPrice, v.minObs, v.maxObs, v.typicalPct, v.unit)
		if err != nil {
			t.Fatalf("insert pricing %s/%s: %v", v.name, v.sku, err)
		}
	}
}

func toolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func resourceRequest(uri string, args map[string]any) mcp.ReadResourceRequest {
	return mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI:       uri,
			Arguments: args,
		},
	}
}

// extractText returns the text content of a tool result (empty if not a text result).
func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("content is not TextContent")
	}
	return tc.Text
}

// extractResourceText returns the text content of a resource result.
func extractResourceText(t *testing.T, contents []mcp.ResourceContents) string {
	t.Helper()
	if len(contents) == 0 {
		t.Fatal("resource contents is empty")
	}
	trc, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatal("resource contents[0] is not TextResourceContents")
	}
	return trc.Text
}

// parseJSON unmarshals a JSON string into the given target.
func parseJSON(t *testing.T, data string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), target); err != nil {
		t.Fatalf("json.Unmarshal: %v\nraw: %s", err, data)
	}
}

// ─── Tool: negotiate_query_price ───

func TestHandleQueryPrice_KnownVendor(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_query_price", map[string]any{
		"vendor": "Slack",
		"sku":    "Pro",
	})
	result, err := ns.handleQueryPrice(ctx, req)
	if err != nil {
		t.Fatalf("handleQueryPrice: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if resp["vendor"] != "Slack" {
		t.Errorf("expected vendor Slack, got %v", resp["vendor"])
	}
	if v, ok := resp["list_price"].(float64); !ok || v != 8.75 {
		t.Errorf("expected list_price 8.75, got %v", resp["list_price"])
	}
	if v, ok := resp["confidence"].(string); !ok || v == "" {
		t.Errorf("expected non-empty confidence, got %v", resp["confidence"])
	}
	// Verify suggested range
	suggested, ok := resp["suggested_counter_offer"].(map[string]any)
	if !ok {
		t.Fatal("expected suggested_counter_offer map")
	}
	sMin, _ := suggested["min"].(float64)
	sMax, _ := suggested["max"].(float64)
	if sMin <= 0 || sMax <= 0 {
		t.Errorf("expected positive suggested range, got min=%f max=%f", sMin, sMax)
	}
	if sMin >= sMax {
		t.Errorf("expected suggested min < max, got min=%f max=%f", sMin, sMax)
	}
}

func TestHandleQueryPrice_UnknownVendor(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_query_price", map[string]any{
		"vendor": "NonExistentCorp",
	})
	result, err := ns.handleQueryPrice(ctx, req)
	if err != nil {
		t.Fatalf("handleQueryPrice: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for unknown vendor")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Pricing lookup failed") && !strings.Contains(text, "not found") {
		t.Errorf("expected pricing lookup error message, got %q", text)
	}
}

// ─── Tool: negotiate_calculate_savings ───

func TestHandleCalculateSavings_KnownVendor(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_calculate_savings", map[string]any{
		"vendor":        "Slack",
		"current_spend": float64(10000),
	})
	result, err := ns.handleCalculateSavings(ctx, req)
	if err != nil {
		t.Fatalf("handleCalculateSavings: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if resp["vendor"] != "Slack" {
		t.Errorf("expected vendor Slack, got %v", resp["vendor"])
	}
	if v, ok := resp["current_spend"].(float64); !ok || v != 10000 {
		t.Errorf("expected current_spend 10000, got %v", resp["current_spend"])
	}
	if v, ok := resp["savings_percentage"].(float64); !ok || v <= 0 {
		t.Errorf("expected positive savings_percentage, got %v", resp["savings_percentage"])
	}
	if v, ok := resp["confidence"].(string); !ok || v == "" {
		t.Errorf("expected non-empty confidence, got %v", resp["confidence"])
	}
}

func TestHandleCalculateSavings_UnknownVendor(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_calculate_savings", map[string]any{
		"vendor":        "NonExistentCorp",
		"current_spend": float64(10000),
	})
	result, err := ns.handleCalculateSavings(ctx, req)
	if err != nil {
		t.Fatalf("handleCalculateSavings: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for unknown vendor")
	}
}

// ─── Tool: negotiate_create_session ───

func TestHandleCreateSession_ValidStrategy(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_create_session", map[string]any{
		"vendor":   "Slack",
		"sku":      "Pro",
		"strategy": "balanced",
	})
	result, err := ns.handleCreateSession(ctx, req)
	if err != nil {
		t.Fatalf("handleCreateSession: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if resp["session_id"] == "" {
		t.Error("expected non-empty session_id")
	}
	if v, ok := resp["initial_offer"].(float64); !ok || v <= 0 {
		t.Errorf("expected positive initial_offer, got %v", resp["initial_offer"])
	}
	if v, ok := resp["list_price"].(float64); !ok || v != 8.75 {
		t.Errorf("expected list_price 8.75, got %v", resp["list_price"])
	}
	if v, ok := resp["strategy"].(string); !ok || v != "balanced" {
		t.Errorf("expected strategy balanced, got %v", resp["strategy"])
	}
}

func TestHandleCreateSession_InvalidStrategy(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_create_session", map[string]any{
		"vendor":   "Slack",
		"sku":      "Pro",
		"strategy": "crazy",
	})
	result, err := ns.handleCreateSession(ctx, req)
	if err != nil {
		t.Fatalf("handleCreateSession: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid strategy")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Session creation failed") {
		t.Errorf("expected session creation failure message, got %q", text)
	}
}

// ─── Tool: negotiate_run ───

func TestHandleRunNegotiation_SessionCreatedThenRun(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	// Step 1: Create a session with aggressive strategy so it auto-accepts with threshold
	createReq := toolRequest("negotiate_create_session", map[string]any{
		"vendor":   "Slack",
		"sku":      "Pro",
		"strategy": "aggressive",
	})
	createResult, err := ns.handleCreateSession(ctx, createReq)
	if err != nil {
		t.Fatalf("handleCreateSession: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create session error: %s", extractText(t, createResult))
	}

	var createResp map[string]any
	parseJSON(t, extractText(t, createResult), &createResp)
	sessionID, _ := createResp["session_id"].(string)
	if sessionID == "" {
		t.Fatal("expected non-empty session_id from create")
	}

	// Step 2: Run the negotiation with a generous auto-approve threshold
	runReq := toolRequest("negotiate_run", map[string]any{
		"session_id":             sessionID,
		"auto_approve_threshold": float64(7.00),
	})
	runResult, err := ns.handleRunNegotiation(ctx, runReq)
	if err != nil {
		t.Fatalf("handleRunNegotiation: %v", err)
	}
	if runResult.IsError {
		t.Fatalf("run negotiation error: %s", extractText(t, runResult))
	}

	var runResp map[string]any
	parseJSON(t, extractText(t, runResult), &runResp)

	if v, ok := runResp["status"].(string); !ok || v != "completed" {
		t.Errorf("expected status completed, got %v", runResp["status"])
	}
	if v, ok := runResp["outcome"].(string); !ok || v != "accepted" {
		t.Errorf("expected outcome accepted, got %v", runResp["outcome"])
	}
	if v, ok := runResp["rounds_completed"].(float64); !ok || v <= 0 {
		t.Errorf("expected positive rounds_completed, got %v", runResp["rounds_completed"])
	}
	if v, ok := runResp["current_offer"].(float64); !ok || v <= 0 {
		t.Errorf("expected positive current_offer, got %v", runResp["current_offer"])
	}
	// Verify we got a history array
	if _, ok := runResp["history"]; !ok {
		t.Error("expected history in response")
	}
}

func TestHandleRunNegotiation_InvalidSession(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_run", map[string]any{
		"session_id": "nonexistent-session-id",
	})
	result, err := ns.handleRunNegotiation(ctx, req)
	if err != nil {
		t.Fatalf("handleRunNegotiation: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for nonexistent session")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Session not found") {
		t.Errorf("expected session not found message, got %q", text)
	}
}

// ─── Tool: negotiate_history ───

func TestHandleHistory_Empty(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_history", map[string]any{})
	result, err := ns.handleHistory(ctx, req)
	if err != nil {
		t.Fatalf("handleHistory: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if v, ok := resp["total_deals"].(float64); !ok || v != 0 {
		t.Errorf("expected 0 total_deals, got %v", resp["total_deals"])
	}
}

func TestHandleHistory_AfterDeal(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	// Create and run a session to generate a deal outcome
	createReq := toolRequest("negotiate_create_session", map[string]any{
		"vendor":   "Slack",
		"sku":      "Pro",
		"strategy": "aggressive",
	})
	createResult, err := ns.handleCreateSession(ctx, createReq)
	if err != nil {
		t.Fatalf("handleCreateSession: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create session error: %s", extractText(t, createResult))
	}

	var createResp map[string]any
	parseJSON(t, extractText(t, createResult), &createResp)
	sessionID, _ := createResp["session_id"].(string)

	// Run to accept
	runReq := toolRequest("negotiate_run", map[string]any{
		"session_id":             sessionID,
		"auto_approve_threshold": float64(7.00),
	})
	_, err = ns.handleRunNegotiation(ctx, runReq)
	if err != nil {
		t.Fatalf("handleRunNegotiation: %v", err)
	}

	// Now check history
	histReq := toolRequest("negotiate_history", map[string]any{
		"period": "all",
	})
	histResult, err := ns.handleHistory(ctx, histReq)
	if err != nil {
		t.Fatalf("handleHistory: %v", err)
	}
	if histResult.IsError {
		t.Fatalf("history error: %s", extractText(t, histResult))
	}

	var histResp map[string]any
	parseJSON(t, extractText(t, histResult), &histResp)

	if v, ok := histResp["total_deals"].(float64); !ok || v <= 0 {
		t.Errorf("expected at least 1 deal, got %v", histResp["total_deals"])
	}
	if v, ok := histResp["total_savings"].(float64); !ok || v <= 0 {
		t.Errorf("expected positive total_savings, got %v", histResp["total_savings"])
	}
}

// ─── Tool: negotiate_strategies ───

func TestHandleStrategies_ReturnsAllThree(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_strategies", map[string]any{})
	result, err := ns.handleStrategies(ctx, req)
	if err != nil {
		t.Fatalf("handleStrategies: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	strats, ok := resp["strategies"].([]any)
	if !ok {
		t.Fatal("expected strategies array")
	}
	if len(strats) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(strats))
	}

	// Verify strategy names
	names := make(map[string]bool)
	for _, s := range strats {
		strat, ok := s.(map[string]any)
		if !ok {
			t.Fatal("strategy is not a map")
		}
		name, _ := strat["name"].(string)
		names[name] = true
	}
	for _, want := range []string{"aggressive", "balanced", "conservative"} {
		if !names[want] {
			t.Errorf("missing strategy: %s", want)
		}
	}
}

// ─── Tool: negotiate_discover_opportunities ───

func TestHandleDiscoverOpportunities_IndustryBased(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_discover_opportunities", map[string]any{
		"business_name": "Acme Corp",
		"description":   "A tech company",
		"industry":      "tech",
		"employees":     float64(500),
	})
	result, err := ns.handleDiscoverOpportunities(ctx, req)
	if err != nil {
		t.Fatalf("handleDiscoverOpportunities: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if v, ok := resp["business_name"].(string); !ok || v != "Acme Corp" {
		t.Errorf("expected business_name Acme Corp, got %v", resp["business_name"])
	}
	opps, ok := resp["opportunities"].([]any)
	if !ok {
		t.Fatal("expected opportunities array")
	}
	if len(opps) == 0 {
		t.Error("expected at least 1 opportunity for tech industry")
	}
	count, ok := resp["opportunity_count"].(float64)
	if !ok || int(count) != len(opps) {
		t.Errorf("opportunity_count %v doesn't match opportunities len %d", count, len(opps))
	}

	// Verify first opportunity has required fields
	first, ok := opps[0].(map[string]any)
	if !ok {
		t.Fatal("first opportunity is not a map")
	}
	if first["vendor"] == "" {
		t.Error("expected non-empty vendor in first opportunity")
	}
	if first["rationale"] == "" {
		t.Error("expected non-empty rationale in first opportunity")
	}
}

// ─── Resource: negotiate://pricing/{vendor}/{sku} ───

func TestPricingResource_KnownVendor(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := resourceRequest("negotiate://pricing/Slack/Pro", map[string]any{
		"vendor": "Slack",
		"sku":    "Pro",
	})
	contents, err := ns.handlePricingResource(ctx, req)
	if err != nil {
		t.Fatalf("handlePricingResource: %v", err)
	}

	text := extractResourceText(t, contents)
	var resp map[string]any
	parseJSON(t, text, &resp)

	if resp["vendor"] != "Slack" {
		t.Errorf("expected vendor Slack, got %v", resp["vendor"])
	}
	if v, ok := resp["list_price"].(float64); !ok || v != 8.75 {
		t.Errorf("expected list_price 8.75, got %v", resp["list_price"])
	}
	if v, ok := resp["confidence"].(string); !ok || v == "" {
		t.Errorf("expected non-empty confidence, got %v", resp["confidence"])
	}
}

// ─── Resource: negotiate://session/{session_id} ───

func TestSessionResource_KnownSession(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	// Create a session via the tool to be stored in history
	createReq := toolRequest("negotiate_create_session", map[string]any{
		"vendor":   "Slack",
		"sku":      "Pro",
		"strategy": "balanced",
	})
	createResult, err := ns.handleCreateSession(ctx, createReq)
	if err != nil {
		t.Fatalf("handleCreateSession: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create session error: %s", extractText(t, createResult))
	}
	var createResp map[string]any
	parseJSON(t, extractText(t, createResult), &createResp)
	sessionID, _ := createResp["session_id"].(string)

	// Read the session resource
	req := resourceRequest("negotiate://session/"+sessionID, map[string]any{
		"session_id": sessionID,
	})
	contents, err := ns.handleSessionResource(ctx, req)
	if err != nil {
		t.Fatalf("handleSessionResource: %v", err)
	}

	text := extractResourceText(t, contents)
	var resp map[string]any
	parseJSON(t, text, &resp)

	sess, ok := resp["session"].(map[string]any)
	if !ok {
		t.Fatal("expected session map in response")
	}
	if sess["vendor"] != "Slack" {
		t.Errorf("expected vendor Slack in session, got %v", sess["vendor"])
	}
	if sess["strategy"] != "balanced" {
		t.Errorf("expected strategy balanced, got %v", sess["strategy"])
	}
}

// ─── Resource: negotiate://history/{period} ───

func TestHistoryResource_KnownPeriod(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := resourceRequest("negotiate://history/all", map[string]any{
		"period": "all",
	})
	contents, err := ns.handleHistoryResource(ctx, req)
	if err != nil {
		t.Fatalf("handleHistoryResource: %v", err)
	}

	text := extractResourceText(t, contents)
	var resp map[string]any
	parseJSON(t, text, &resp)

	if v, ok := resp["total_deals"].(float64); !ok {
		t.Errorf("expected total_deals field, got %v", resp["total_deals"])
		_ = v
	}
}

// ─── Resource: negotiate://strategies ───

func TestStrategiesResource_ReturnsAll(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := resourceRequest("negotiate://strategies", map[string]any{})
	contents, err := ns.handleStrategiesResource(ctx, req)
	if err != nil {
		t.Fatalf("handleStrategiesResource: %v", err)
	}

	text := extractResourceText(t, contents)
	var resp map[string]any
	parseJSON(t, text, &resp)

	strats, ok := resp["strategies"].([]any)
	if !ok {
		t.Fatal("expected strategies array")
	}
	if len(strats) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(strats))
	}
}

// ─── Resource: negotiate://opportunities/{industry} ───

func TestOpportunitiesResource_KnownIndustry(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := resourceRequest("negotiate://opportunities/tech", map[string]any{
		"industry": "tech",
	})
	contents, err := ns.handleOpportunitiesResource(ctx, req)
	if err != nil {
		t.Fatalf("handleOpportunitiesResource: %v", err)
	}

	text := extractResourceText(t, contents)
	var resp map[string]any
	parseJSON(t, text, &resp)

	opps, ok := resp["opportunities"].([]any)
	if !ok {
		t.Fatal("expected opportunities array")
	}
	if len(opps) == 0 {
		t.Error("expected at least 1 opportunity for tech industry")
	}
	if v, ok := resp["industry"].(string); !ok || v != "tech" {
		t.Errorf("expected industry tech, got %v", resp["industry"])
	}
	_ = opps
}

func TestHandleFindCheapestModel_Chat(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_find_cheapest_model", map[string]any{
		"task_type": "chat",
	})
	result, err := ns.handleFindCheapestModel(ctx, req)
	if err != nil {
		t.Fatalf("handleFindCheapestModel: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	if resp["task_type"] != "chat" {
		t.Errorf("expected task_type chat, got %v", resp["task_type"])
	}
	models, ok := resp["models"].([]any)
	if !ok || len(models) == 0 {
		t.Fatalf("expected at least one model, got %v", resp["models"])
	}

	// Verify ordering: cheapest first
	var prevPrice float64
	for i, m := range models {
		model := m.(map[string]any)
		price, ok := model["price_per_unit"].(float64)
		if !ok {
			t.Fatalf("model[%d] missing price_per_unit", i)
		}
		if i > 0 && price < prevPrice {
			t.Errorf("models not sorted by price: model[%d]=%f < model[%d]=%f", i, price, i-1, prevPrice)
		}
		prevPrice = price
	}
}

func TestHandleFindCheapestModel_WithBudget(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_find_cheapest_model", map[string]any{
		"task_type": "chat",
		"budget":    1.0,
	})
	result, err := ns.handleFindCheapestModel(ctx, req)
	if err != nil {
		t.Fatalf("handleFindCheapestModel: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	models, ok := resp["models"].([]any)
	if !ok {
		t.Fatalf("expected models array, got %v", resp["models"])
	}

	for i, m := range models {
		model := m.(map[string]any)
		price, ok := model["price_per_unit"].(float64)
		if !ok {
			t.Fatalf("model[%d] missing price_per_unit", i)
		}
		if price > 1.0 {
			t.Errorf("model[%d] price %f exceeds budget 1.0", i, price)
		}
	}
}

func TestHandleFindCheapestModel_InvalidTaskType(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_find_cheapest_model", map[string]any{
		"task_type": "invalid_task",
	})
	result, err := ns.handleFindCheapestModel(ctx, req)
	if err != nil {
		t.Fatalf("handleFindCheapestModel: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error for invalid task_type, got success")
	}
}

func TestHandleFindCheapestModel_ImageGeneration(t *testing.T) {
	ns := setupTest(t)
	ctx := context.Background()

	req := toolRequest("negotiate_find_cheapest_model", map[string]any{
		"task_type": "image_generation",
	})
	result, err := ns.handleFindCheapestModel(ctx, req)
	if err != nil {
		t.Fatalf("handleFindCheapestModel: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	var resp map[string]any
	parseJSON(t, extractText(t, result), &resp)

	models, ok := resp["models"].([]any)
	if !ok {
		t.Fatalf("expected models array, got %v", resp["models"])
	}
	if len(models) == 0 {
		t.Fatal("expected at least one model for image_generation")
	}
}
