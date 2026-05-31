package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
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

// ─── Benchmark setup ───

func setupBenchServer(b *testing.B) *NegotiationServer {
	b.Helper()

	pstore, err := pricing.NewInMemoryStore()
	if err != nil {
		b.Fatalf("NewInMemoryStore: %v", err)
	}
	b.Cleanup(func() { pstore.Close() })

	hstore, err := history.NewStore(pstore.DB())
	if err != nil {
		b.Fatalf("history.NewStore: %v", err)
	}

	seedBenchPricingData(b, pstore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gstore, err := group.NewStore(pstore.DB())
	if err != nil {
		b.Fatalf("group.NewStore: %v", err)
	}
	geng := group.NewEngine(gstore, pstore, logger)

	sstore, err := sell.NewStore(pstore.DB())
	if err != nil {
		b.Fatalf("sell.NewStore: %v", err)
	}
	seng := sell.NewEngine(sstore, logger)

	negEng := negotiation.NewEngine(pstore)
	cstore, err := calendar.NewStore(pstore.DB())
	if err != nil {
		b.Fatalf("calendar.NewStore: %v", err)
	}
	ceng := calendar.NewEngine(cstore, negEng, hstore, pstore, logger)

	hstore2, err := health.NewStoreFromDB(pstore.DB())
	if err != nil {
		b.Fatalf("health.NewStoreFromDB: %v", err)
	}
	heng := health.NewEngine(hstore2, logger)

	slastore, err := sla.NewStore(pstore.DB())
	if err != nil {
		b.Fatalf("sla.NewStore: %v", err)
	}
	slaEng := sla.NewEngine(slastore, logger)

	return NewNegotiationServer(pstore, hstore, geng, seng, ceng, heng, nil, slaEng, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
}

func seedBenchPricingData(b *testing.B, store *pricing.Store) {
	b.Helper()
	ctx := context.Background()

	type product struct {
		name, category, sku, desc string
		listPrice, minObs, maxObs float64
		typicalPct                float64
		unit                      string
	}

	products := []product{
		{"Slack", "Communication", "Pro", "Pro plan", 8.75, 6.50, 8.00, 18, "per_seat_month"},
		{"Slack", "Communication", "Enterprise", "Enterprise Grid", 15.00, 10.00, 14.00, 20, "per_seat_month"},
		{"GitHub", "Developer", "Team", "Team plan", 4.00, 3.00, 3.80, 15, "per_seat_month"},
		{"GitHub", "Developer", "Enterprise", "Enterprise plan", 21.00, 15.00, 20.00, 25, "per_seat_month"},
		{"Salesforce", "CRM", "Enterprise", "Enterprise per seat", 165.00, 110.00, 155.00, 28, "per_seat_month"},
		{"Salesforce", "CRM", "Professional", "Professional per seat", 85.00, 60.00, 80.00, 22, "per_seat_month"},
		{"Zoom", "Communication", "Business", "Business plan", 19.99, 14.99, 18.99, 20, "per_seat_month"},
		{"Zoom", "Communication", "Enterprise", "Enterprise plan", 29.99, 22.00, 28.00, 22, "per_seat_month"},
		{"Datadog", "Monitoring", "Pro", "Pro monitoring", 15.00, 10.00, 14.00, 18, "per_seat_month"},
		{"Datadog", "Monitoring", "Enterprise", "Enterprise monitoring", 25.00, 18.00, 23.00, 20, "per_seat_month"},
		{"AWS", "Cloud", "Business", "Business Support", 100.00, 75.00, 95.00, 15, "flat_monthly"},
		{"AWS", "Cloud", "Enterprise", "Enterprise Support", 15000.00, 10000.00, 14000.00, 25, "flat_monthly"},
	}

	for _, p := range products {
		_, err := store.DB().ExecContext(ctx,
			"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
			p.name, p.category)
		if err != nil {
			b.Fatalf("insert vendor %s: %v", p.name, err)
		}
		var vid int64
		err = store.DB().QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", p.name).Scan(&vid)
		if err != nil {
			b.Fatalf("get vendor id %s: %v", p.name, err)
		}
		_, err = store.DB().ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET
				list_price=excluded.list_price,
				min_observed=excluded.min_observed,
				max_observed=excluded.max_observed,
				typical_pct=excluded.typical_pct,
				description=excluded.description,
				updated_at=datetime('now')
		`, vid, p.sku, p.desc, p.listPrice, p.minObs, p.maxObs, p.typicalPct, p.unit)
		if err != nil {
			b.Fatalf("insert pricing %s/%s: %v", p.name, p.sku, err)
		}
	}
}

func benchToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func benchExtractText(b *testing.B, result *mcp.CallToolResult) string {
	b.Helper()
	if result == nil {
		b.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		b.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		b.Fatal("content is not TextContent")
	}
	return tc.Text
}

var pricingVendorSKUs = []struct {
	vendor, sku string
}{
	{"Slack", "Pro"},
	{"Slack", "Enterprise"},
	{"GitHub", "Team"},
	{"GitHub", "Enterprise"},
	{"Salesforce", "Enterprise"},
	{"Salesforce", "Professional"},
	{"Zoom", "Business"},
	{"Zoom", "Enterprise"},
	{"Datadog", "Pro"},
	{"Datadog", "Enterprise"},
}

// ─── Benchmark 1: Pricing Query (140 lookups) ───

func BenchmarkPricingQuery(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 140; j++ {
			vs := pricingVendorSKUs[j%len(pricingVendorSKUs)]
			req := benchToolRequest("negotiate_query_price", map[string]any{
				"vendor": vs.vendor,
				"sku":    vs.sku,
			})
			result, err := ns.handleQueryPrice(ctx, req)
			if err != nil {
				b.Fatalf("handleQueryPrice: %v", err)
			}
			if result.IsError {
				b.Fatalf("query price error: %s", benchExtractText(b, result))
			}
		}
	}
}

// ─── Benchmark 2: Create Session ───

func BenchmarkCreateSession(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := benchToolRequest("negotiate_create_session", map[string]any{
			"vendor":   "Slack",
			"sku":      "Pro",
			"strategy": "balanced",
		})
		result, err := ns.handleCreateSession(ctx, req)
		if err != nil {
			b.Fatalf("handleCreateSession: %v", err)
		}
		if result.IsError {
			b.Fatalf("create session error: %s", benchExtractText(b, result))
		}
	}
}

// ─── Benchmark 3: Run Negotiation ───

func BenchmarkRunNegotiation(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		createReq := benchToolRequest("negotiate_create_session", map[string]any{
			"vendor":   "Slack",
			"sku":      "Pro",
			"strategy": "aggressive",
		})
		createResult, err := ns.handleCreateSession(ctx, createReq)
		if err != nil {
			b.Fatalf("handleCreateSession: %v", err)
		}
		if createResult.IsError {
			b.Fatalf("create session error: %s", benchExtractText(b, createResult))
		}

		var createResp map[string]any
		if err := json.Unmarshal([]byte(benchExtractText(b, createResult)), &createResp); err != nil {
			b.Fatalf("json.Unmarshal: %v", err)
		}
		sessionID, _ := createResp["session_id"].(string)
		if sessionID == "" {
			b.Fatal("empty session_id")
		}

		runReq := benchToolRequest("negotiate_run", map[string]any{
			"session_id":             sessionID,
			"auto_approve_threshold": float64(7.00),
		})
		runResult, err := ns.handleRunNegotiation(ctx, runReq)
		if err != nil {
			b.Fatalf("handleRunNegotiation: %v", err)
		}
		if runResult.IsError {
			b.Fatalf("run negotiation error: %s", benchExtractText(b, runResult))
		}
	}
}

// ─── Benchmark 4: Parallel 2 ───

func BenchmarkParallel2(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var sessionIDs []string
		for _, v := range []struct{ vendor, sku string }{{"Slack", "Pro"}, {"GitHub", "Team"}} {
			req := benchToolRequest("negotiate_create_session", map[string]any{
				"vendor":   v.vendor,
				"sku":      v.sku,
				"strategy": "aggressive",
			})
			result, err := ns.handleCreateSession(ctx, req)
			if err != nil {
				b.Fatalf("handleCreateSession: %v", err)
			}
			if result.IsError {
				b.Fatalf("create session error: %s", benchExtractText(b, result))
			}
			var resp map[string]any
			if err := json.Unmarshal([]byte(benchExtractText(b, result)), &resp); err != nil {
				b.Fatalf("json.Unmarshal: %v", err)
			}
			sessionIDs = append(sessionIDs, resp["session_id"].(string))
		}

		var wg sync.WaitGroup
		for _, sid := range sessionIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				runReq := benchToolRequest("negotiate_run", map[string]any{
					"session_id":             id,
					"auto_approve_threshold": float64(7.00),
				})
				result, err := ns.handleRunNegotiation(ctx, runReq)
				if err != nil {
					b.Errorf("handleRunNegotiation: %v", err)
				}
				if result != nil && result.IsError {
					b.Errorf("run negotiation error: %s", benchExtractText(b, result))
				}
			}(sid)
		}
		wg.Wait()
	}
}

// ─── Benchmark 5: Parallel 5 ───

func BenchmarkParallel5(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	parallelVendors := []struct{ vendor, sku string }{
		{"Slack", "Pro"},
		{"GitHub", "Team"},
		{"Salesforce", "Enterprise"},
		{"Zoom", "Business"},
		{"Datadog", "Pro"},
	}

	for i := 0; i < b.N; i++ {
		var sessionIDs []string
		for _, v := range parallelVendors {
			req := benchToolRequest("negotiate_create_session", map[string]any{
				"vendor":   v.vendor,
				"sku":      v.sku,
				"strategy": "aggressive",
			})
			result, err := ns.handleCreateSession(ctx, req)
			if err != nil {
				b.Fatalf("handleCreateSession: %v", err)
			}
			if result.IsError {
				b.Fatalf("create session error: %s", benchExtractText(b, result))
			}
			var resp map[string]any
			if err := json.Unmarshal([]byte(benchExtractText(b, result)), &resp); err != nil {
				b.Fatalf("json.Unmarshal: %v", err)
			}
			sessionIDs = append(sessionIDs, resp["session_id"].(string))
		}

		var wg sync.WaitGroup
		for _, sid := range sessionIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				runReq := benchToolRequest("negotiate_run", map[string]any{
					"session_id":             id,
					"auto_approve_threshold": float64(7.00),
				})
				result, err := ns.handleRunNegotiation(ctx, runReq)
				if err != nil {
					b.Errorf("handleRunNegotiation: %v", err)
				}
				if result != nil && result.IsError {
					b.Errorf("run negotiation error: %s", benchExtractText(b, result))
				}
			}(sid)
		}
		wg.Wait()
	}
}

// ─── Benchmark 6: Compute Offer (10 members) ───

func BenchmarkComputeOffer(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()

	createReq := benchToolRequest("negotiate_create_group", map[string]any{
		"target_vendor":    "Slack",
		"target_sku":       "Pro",
		"min_members":      int64(2),
		"expires_in_hours": int64(72),
	})
	createResult, err := ns.handleCreateGroup(ctx, createReq)
	if err != nil {
		b.Fatalf("handleCreateGroup: %v", err)
	}
	if createResult.IsError {
		b.Fatalf("create group error: %s", benchExtractText(b, createResult))
	}
	var createResp map[string]any
	if err := json.Unmarshal([]byte(benchExtractText(b, createResult)), &createResp); err != nil {
		b.Fatalf("json.Unmarshal: %v", err)
	}
	groupID, _ := createResp["group_id"].(string)
	if groupID == "" {
		b.Fatal("empty group_id")
	}

	for j := 0; j < 10; j++ {
		joinReq := benchToolRequest("negotiate_join_group", map[string]any{
			"group_id": groupID,
			"user_id":  "bench-user",
			"quantity": int64(5 + j),
		})
		joinResult, err := ns.handleJoinGroup(ctx, joinReq)
		if err != nil {
			b.Fatalf("handleJoinGroup: %v", err)
		}
		if joinResult.IsError {
			b.Fatalf("join group error: %s", benchExtractText(b, joinResult))
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := benchToolRequest("negotiate_compute_offer", map[string]any{
			"group_id": groupID,
		})
		result, err := ns.handleComputeOffer(ctx, req)
		if err != nil {
			b.Fatalf("handleComputeOffer: %v", err)
		}
		if result.IsError {
			b.Fatalf("compute offer error: %s", benchExtractText(b, result))
		}
	}
}

// ─── Benchmark 7: Quote Parse ───

var benchQuoteTexts = []string{
	`Subject: Quote for Slack Pro Plan
Dear Customer,

Thank you for your interest in Slack. Please find below our quote:

Product: Slack Pro (50 seats)
Price: $8.75 per seat per month
Billing: Annual

Total: $5,250.00 per year

This offer is valid for 30 days.

Best regards,
Slack Sales Team`,

	`Quote for GitHub Enterprise
GitHub Enterprise: 100 users
$21.00/user/month billed annually
Includes 24/7 support and SAML SSO
Total annual commitment: $25,200.00`,

	`Salesforce Enterprise Quote
SKU: ENTERPRISE-SALES
Per user pricing: $165.00/user/month
100 licenses, annual contract
Implementation fee: $5,000 one-time`,

	`AWS Business Support
Monthly fee: $100.00
Includes 24x7 phone support
1-hour response time for critical cases`,

	`Datadog Pro Plan
Pro Monitoring: $15.00 per host per month
200 hosts included
Annual contract recommended
Contact us for volume discounts`,
}

func BenchmarkQuoteParse(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		text := benchQuoteTexts[i%len(benchQuoteTexts)]
		req := benchToolRequest("negotiate_analyze_quote", map[string]any{
			"raw_text": text,
		})
		result, err := ns.handleAnalyzeQuote(ctx, req)
		if err != nil {
			b.Fatalf("handleAnalyzeQuote: %v", err)
		}
		if result.IsError {
			b.Fatalf("analyze quote error: %s", benchExtractText(b, result))
		}
	}
}

// ─── Benchmark 8: Contract Parse ───

var benchContractTexts = []string{
	`MASTER SERVICES AGREEMENT

This Agreement is entered into on January 15, 2026 by and between Acme Corp ("Customer") and Cloud Services Inc. ("Provider").

1. TERM
The initial term of this Agreement shall be 12 months, commencing on February 1, 2026.

2. AUTO-RENEWAL
This Agreement shall automatically renew for successive 12-month terms unless either party provides written notice of non-renewal at least 60 days prior to the end of the then-current term.

3. PRICING
The monthly fee for services is $15,000.00. Provider agrees that pricing shall remain locked for the first 12 months of this Agreement.

4. TERMINATION
Either party may terminate this Agreement for convenience upon 90 days written notice. For cause termination requires 30 days cure period.

5. DATA PORTABILITY
Upon termination, Provider shall make all Customer data available for export within 30 days.`,

	`SaaS SUBSCRIPTION AGREEMENT

Effective Date: 2026-03-01
Vendor: DataWare Inc.
SKU: DW-PRO-2026

Term: 24 months
This agreement will automatically renew unless cancelled 45 days before renewal date.
Price guarantee: first 24 months.

Monthly subscription: $8,750.00
Annual commitment: $95,000.00 (saves 10%)

Notice period: 60 days
Data portability included.

End Date: 2028-02-29`,

	`SOFTWARE LICENSE AGREEMENT

Between: MegaCorp and SmallSoft Inc.
Date: June 1, 2026

Term: 36 months (3 years)
Auto-renewal: Yes, renews automatically for 1-year terms unless notice given 90 days prior.

Price: $25.00 per user per month
Locked for initial term of 36 months.

Termination: 30 days written notice by either party.
No data portability clause included.

Expiration: May 31, 2029`,
}

func BenchmarkContractParse(b *testing.B) {
	ns := setupBenchServer(b)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		text := benchContractTexts[i%len(benchContractTexts)]
		req := benchToolRequest("negotiate_parse_contract", map[string]any{
			"raw_text": text,
		})
		result, err := ns.handleParseContract(ctx, req)
		if err != nil {
			b.Fatalf("handleParseContract: %v", err)
		}
		if result.IsError {
			b.Fatalf("parse contract error: %s", benchExtractText(b, result))
		}
	}
}
