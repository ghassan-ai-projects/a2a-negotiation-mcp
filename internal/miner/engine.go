package miner

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// Engine discovers negotiation opportunities from business data.
type Engine struct {
	pricingStore *pricing.Store
	logger       *slog.Logger
}

// NewEngine creates a mining engine.
func NewEngine(pricingStore *pricing.Store, logger *slog.Logger) *Engine {
	return &Engine{
		pricingStore: pricingStore,
		logger:       logger,
	}
}

// DiscoverOpportunities analyzes a business profile and returns ranked opportunities.
func (e *Engine) DiscoverOpportunities(ctx context.Context, profile BusinessProfile) ([]NegotiationOpportunity, error) {
	e.logger.Debug("discovering opportunities",
		"name", profile.Name,
		"industry", profile.Industry,
		"vendors", profile.Vendors,
	)

	candidates := e.generateCandidates(profile)

	// Cross-reference known vendors against the pricing DB.
	if len(profile.Vendors) > 0 {
		for _, v := range profile.Vendors {
			opp, err := e.crossReferenceVendor(ctx, v)
			if err == nil {
				candidates = append(candidates, opp)
			} else {
				e.logger.Debug("vendor cross-reference skipped",
					"vendor", v, "error", err.Error(),
				)
			}
		}
	}

	// Score all candidates.
	scored := e.scoreOpportunities(candidates)

	// Sort descending by score.
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top 10.
	n := 10
	if len(scored) < n {
		n = len(scored)
	}
	result := make([]NegotiationOpportunity, n)
	for i := 0; i < n; i++ {
		result[i] = scored[i].NegotiationOpportunity
		result[i].ID = fmt.Sprintf("opp-%s-%d", result[i].Category, i+1)
	}
	return result, nil
}

// ─── internal types ───

type scoredOpportunity struct {
	NegotiationOpportunity
	score float64
}

// ─── candidate generation ───

func (e *Engine) generateCandidates(profile BusinessProfile) []NegotiationOpportunity {
	// Industry-specific opportunity templates.
	type template struct {
		category    string
		vendor      string
		spend       float64
		discountPct float64
		confidence  string
		rationale   string
	}

	// Build templates based on industry.
	var templates []template
	switch profile.Industry {
	case "tech", "technology", "software", "saas":
		templates = []template{
			{"software", "Microsoft", 250000, 20, "high", "Software licensing renewals offer 15-25% savings with multi-year commitments"},
			{"hosting", "AWS", 500000, 25, "high", "Cloud hosting spend is highly negotiable — reserved instances save 25-40%"},
			{"saas", "Salesforce", 120000, 18, "high", "SaaS subscriptions are typically 10-20% below list price at renewal"},
		}
	case "logistics", "transportation", "shipping":
		templates = []template{
			{"carrier", "FedEx", 200000, 22, "high", "Carrier contracts are routinely 15-25% below published rates for volume shippers"},
			{"software", "Oracle", 150000, 20, "medium", "Logistics software licensing has 15-25% negotiation headroom"},
			{"saas", "Salesforce", 80000, 18, "medium", "CRM subscriptions are typically 10-20% below list at renewal"},
		}
	case "healthcare", "health":
		templates = []template{
			{"software", "Epic", 350000, 20, "high", "Healthcare software licensing is negotiable, especially at renewal"},
			{"hosting", "AWS", 400000, 25, "high", "Cloud hosting for healthcare data benefits from reserved instance pricing"},
			{"saas", "Zoom", 90000, 18, "medium", "SaaS tools used across healthcare orgs can be consolidated for savings"},
		}
	case "retail", "ecommerce":
		templates = []template{
			{"saas", "Shopify", 100000, 18, "high", "E-commerce platform fees are negotiable at scale"},
			{"hosting", "Cloudflare", 150000, 25, "high", "CDN and hosting costs drop significantly with committed usage"},
			{"carrier", "UPS", 180000, 22, "medium", "Shipping contracts have 15-25% negotiation room for volume shippers"},
		}
	case "manufacturing", "industrial":
		templates = []template{
			{"software", "SAP", 300000, 20, "high", "ERP licensing is one of the most negotiable software categories"},
			{"hosting", "Azure", 250000, 25, "high", "Cloud migration and hosting can save 20-30% with right commitment levels"},
			{"carrier", "DHL", 120000, 22, "medium", "Freight and logistics contracts benefit from competitive bidding"},
		}
	default:
		// Industry-agnostic fallback.
		templates = []template{
			{"software", "Microsoft", 150000, 20, "medium", "Software licensing — annual renewals are an opportunity to renegotiate terms"},
			{"hosting", "AWS", 200000, 25, "medium", "Cloud hosting costs can be reduced 15-30% through commitment discounts"},
			{"saas", "Salesforce", 80000, 18, "medium", "SaaS subscriptions — consolidation and multi-year terms unlock savings"},
			{"carrier", "FedEx", 100000, 22, "medium", "Carrier and shipping contracts are negotiable for any business with regular shipments"},
			{"freelance", "", 50000, 10, "low", "Contractor and freelance rates — benchmark against market for fair pricing"},
		}
	}

	// Adjust spend based on employee count.
	empMultiplier := 1.0
	if profile.Employees > 0 {
		switch {
		case profile.Employees > 1000:
			empMultiplier = 2.5
		case profile.Employees > 500:
			empMultiplier = 1.8
		case profile.Employees > 100:
			empMultiplier = 1.3
		case profile.Employees > 10:
			empMultiplier = 1.0
		default:
			empMultiplier = 0.6
		}
	}

	candidates := make([]NegotiationOpportunity, 0, len(templates))
	for _, t := range templates {
		e.logger.Debug("  candidate", "category", t.category, "vendor", t.vendor,
			"spend", t.spend, "emp_mult", empMultiplier,
		)
		candidates = append(candidates, NegotiationOpportunity{
			Category:        t.category,
			Vendor:          t.vendor,
			EstimatedSpend:  math.Round(t.spend*empMultiplier*100) / 100,
			Confidence:      t.confidence,
			Rationale:       t.rationale,
			TypicalDiscount: t.discountPct,
		})
	}

	return candidates
}

// ─── vendor cross-reference ───

func (e *Engine) crossReferenceVendor(ctx context.Context, vendor string) (NegotiationOpportunity, error) {
	result, err := e.pricingStore.GetPricingByVendorSKU(ctx, vendor, "")
	if err != nil {
		return NegotiationOpportunity{}, err
	}

	confidence := "low"
	if result.TypicalPct > 15 {
		confidence = "high"
	} else if result.TypicalPct > 10 {
		confidence = "medium"
	}

	return NegotiationOpportunity{
		Category:        result.Vendor,
		Vendor:          result.Vendor,
		EstimatedSpend:  0, // unknown — caller can estimate
		Confidence:      confidence,
		Rationale:       fmt.Sprintf("Pricing data shows %.0f%% typical discount for %s — stronger negotiating position", result.TypicalPct, result.Vendor),
		TypicalDiscount: result.TypicalPct,
	}, nil
}

// ─── scoring ───

func (e *Engine) scoreOpportunities(opps []NegotiationOpportunity) []scoredOpportunity {
	confidenceValue := map[string]float64{
		"high":   3,
		"medium": 2,
		"low":    1,
	}

	result := make([]scoredOpportunity, 0, len(opps))
	for _, opp := range opps {
		estimatedSavings := opp.EstimatedSpend * opp.TypicalDiscount / 100.0
		cv := confidenceValue[opp.Confidence]
		if cv == 0 {
			cv = 1
		}
		score := estimatedSavings * cv
		result = append(result, scoredOpportunity{
			NegotiationOpportunity: opp,
			score:                  score,
		})
	}
	return result
}
