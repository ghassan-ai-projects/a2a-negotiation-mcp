package quote

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// KnownVendors is a list of common SaaS vendor names for pattern matching.
var KnownVendors = []string{
	"Salesforce", "Slack", "GitHub", "GitLab", "Atlassian", "Jira", "Confluence",
	"Microsoft", "Office 365", "Azure", "AWS", "Amazon Web Services", "Google Cloud",
	"GCP", "Google Workspace", "Zoom", "Datadog", "New Relic", "Splunk",
	"Elastic", "MongoDB", "Snowflake", "Databricks", "Palantir", "Twilio",
	"SendGrid", "Stripe", "Shopify", "HubSpot", "Marketo", "Adobe", "Workday",
	"ServiceNow", "Zendesk", "Freshworks", "Asana", "Monday", "Notion",
	"Figma", "Canva", "Okta", "Auth0", "CrowdStrike", "Palo Alto Networks",
	"Cloudflare", "Fastly", "Akamai", "DigitalOcean", "Linode", "Vercel",
	"Netlify", "Heroku", "Oracle", "SAP", "IBM", "Cisco", "VMware",
	"Dell", "HP", "Tableau", "Looker", "Segment", "Amplitude", "Mixpanel",
	"Intercom", "Drift", "Calendly", "Loom", "Miro", "Jfrog", "Docker",
	"Kubernetes", "Red Hat", "Hashicorp", "Terraform", "PagerDuty", "Sentry",
}

// pricePattern matches dollar amounts like $145.00, $1,234.56, $8.75, $165
var pricePattern = regexp.MustCompile(`\$([\d,]+(?:\.\d{2})?)`)

// quantityPattern matches quantity indicators like "50 seats", "100 users", "25 licenses"
var quantityPattern = regexp.MustCompile(`(\d+)\s*(seats|users|licenses|copies|employees|members|people)\b`)

// annualTermPattern matches annual/yearly term indicators
var annualTermPattern = regexp.MustCompile(`(?i)(annually|annual|per year|/year|year)`)

// monthlyTermPattern matches monthly term indicators
var monthlyTermPattern = regexp.MustCompile(`(?i)(monthly|per month|/month|month)`)

// vendorPattern matches common vendor names in text
var vendorPattern = buildVendorPattern()

func buildVendorPattern() *regexp.Regexp {
	// Escape vendor names and join with |
	escaped := make([]string, len(KnownVendors))
	for i, v := range KnownVendors {
		escaped[i] = regexp.QuoteMeta(v)
	}
	return regexp.MustCompile(`(?i)(` + strings.Join(escaped, "|") + `)`)
}

// Engine provides quote parsing and analysis.
type Engine struct {
	pricingStore *pricing.Store
	logger       *slog.Logger
}

// NewEngine creates a new quote Engine.
func NewEngine(pricingStore *pricing.Store, logger *slog.Logger) *Engine {
	return &Engine{
		pricingStore: pricingStore,
		logger:       logger,
	}
}

// AnalyzeQuote extracts pricing information from raw text, cross-references
// against the pricing database, and returns a full analysis with
// counter-offer recommendations.
func (e *Engine) AnalyzeQuote(ctx context.Context, input QuoteInput) (*QuoteAnalysis, error) {
	quote := Quote{}

	// Determine vendor: use explicit input or extract from text
	vendor := input.Vendor
	if vendor == "" && input.RawText != "" {
		vendor = e.extractVendor(input.RawText)
	}
	quote.Vendor = vendor

	// Determine SKU: use explicit input or leave empty for any match
	quote.SKU = input.SKU

	// Extract quantity
	seats := e.extractQuantity(input.RawText)
	if seats > 0 {
		quote.Seats = seats
	} else {
		quote.Seats = 1 // default
	}

	// Extract price per unit
	price := e.extractPrice(input.RawText)
	quote.PricePerUnit = price

	// Extract term
	term := e.extractTerm(input.RawText)
	quote.TermMonths = term

	// Calculate total price
	quote.TotalPrice = quote.PricePerUnit * float64(quote.Seats) * float64(quote.TermMonths)

	// Cross-reference with pricing database
	if vendor == "" {
		return nil, fmt.Errorf("unable to determine vendor from input")
	}

	pricingResult, err := e.pricingStore.GetPricingByVendorSKU(ctx, vendor, input.SKU)
	if err != nil {
		// Pricing data not found — return partial analysis with what we have
		e.logger.Warn("pricing cross-reference failed", "vendor", vendor, "error", err)
		quote.ListPrice = 0
		quote.DiscountOffered = 0

		analysis := &QuoteAnalysis{
			Quote:            quote,
			MarketRange:      []float64{0, 0},
			CounterOfferMin:  0,
			CounterOfferMax:  0,
			PotentialSavings: 0,
			Confidence:       "low",
		}
		return analysis, nil
	}

	quote.ListPrice = pricingResult.ListPrice
	quote.Description = pricingResult.Description

	// Calculate discount offered
	if pricingResult.ListPrice > 0 {
		if pricingResult.ListPrice > 0 {
			quote.DiscountOffered = math.Round((1-quote.PricePerUnit/pricingResult.ListPrice)*100*100) / 100
		}
	}

	// Build analysis
	marketRange := []float64{pricingResult.MarketMin, pricingResult.MarketMax}

	// Counter-offer range from pricing DB suggestions
	counterMin := pricingResult.SuggestedMin
	counterMax := pricingResult.SuggestedMax

	// Calculate potential savings
	// Current total vs. what we'd pay at counter-offer max per unit
	totalAtCounterMax := counterMax * float64(quote.Seats) * float64(quote.TermMonths)
	potentialSavings := quote.TotalPrice - totalAtCounterMax
	if potentialSavings < 0 {
		potentialSavings = 0
	}

	return &QuoteAnalysis{
		Quote:            quote,
		MarketRange:      marketRange,
		CounterOfferMin:  math.Round(counterMin*100) / 100,
		CounterOfferMax:  math.Round(counterMax*100) / 100,
		PotentialSavings: math.Round(potentialSavings*100) / 100,
		Confidence:       pricingResult.Confidence,
	}, nil
}

// GenerateCounterOffer produces a formatted counter-offer text from an analysis.
func (e *Engine) GenerateCounterOffer(ctx context.Context, analysis *QuoteAnalysis) (string, error) {
	q := analysis.Quote

	var b strings.Builder
	b.WriteString("╔══════════════════════════════════════════════╗\n")
	b.WriteString("║          COUNTER-OFFER ANALYSIS             ║\n")
	b.WriteString("╚══════════════════════════════════════════════╝\n\n")

	b.WriteString(fmt.Sprintf("Vendor:            %s\n", q.Vendor))
	if q.SKU != "" {
		b.WriteString(fmt.Sprintf("SKU:               %s\n", q.SKU))
	}
	if q.Description != "" {
		b.WriteString(fmt.Sprintf("Product:           %s\n", q.Description))
	}
	b.WriteString(fmt.Sprintf("Seats:             %d\n", q.Seats))
	b.WriteString(fmt.Sprintf("Term:              %d months\n", q.TermMonths))
	b.WriteString(fmt.Sprintf("Quoted Price:      $%.2f/unit\n", q.PricePerUnit))
	b.WriteString(fmt.Sprintf("Total Quoted:      $%.2f\n", q.TotalPrice))

	if q.ListPrice > 0 {
		b.WriteString(fmt.Sprintf("List Price:        $%.2f/unit\n", q.ListPrice))
		b.WriteString(fmt.Sprintf("Discount Offered:  %.1f%%\n", q.DiscountOffered))
	}

	b.WriteString("\n── Market Analysis ──────────────────────────────\n")
	if len(analysis.MarketRange) == 2 && analysis.MarketRange[0] > 0 {
		b.WriteString(fmt.Sprintf("Market Range:      $%.2f – $%.2f/unit\n", analysis.MarketRange[0], analysis.MarketRange[1]))
	} else {
		b.WriteString("Market Range:      No market data available\n")
	}
	b.WriteString(fmt.Sprintf("Confidence:        %s\n", analysis.Confidence))

	b.WriteString("\n── Counter-Offer Recommendation ─────────────────\n")
	if analysis.CounterOfferMax > 0 {
		b.WriteString(fmt.Sprintf("Suggested Range:   $%.2f – $%.2f/unit\n", analysis.CounterOfferMin, analysis.CounterOfferMax))
		savingsText := fmt.Sprintf("$%.2f", analysis.PotentialSavings)
		b.WriteString(fmt.Sprintf("Potential Savings: %s\n", savingsText))
	} else {
		b.WriteString("Suggested Range:   Insufficient market data\n")
	}

	b.WriteString("\n── Negotiation Points ───────────────────────────\n")
	b.WriteString(fmt.Sprintf("• The quoted price of $%.2f/unit", q.PricePerUnit))
	if analysis.CounterOfferMax > 0 && q.PricePerUnit > analysis.CounterOfferMax {
		b.WriteString(fmt.Sprintf(" is above market range.\n"))
		b.WriteString(fmt.Sprintf("  Target $%.2f/unit or lower for a competitive deal.\n", analysis.CounterOfferMin))
	} else if analysis.CounterOfferMax > 0 {
		b.WriteString(" is within or below market range.\n")
		b.WriteString("  Consider locking in multi-year terms for additional discounts.\n")
	}
	if analysis.Confidence == "high" {
		b.WriteString("• High confidence — extensive market data supports this range.\n")
	} else if analysis.Confidence == "medium" {
		b.WriteString("• Medium confidence — consider gathering more competitive quotes.\n")
	} else {
		b.WriteString("• Low confidence — limited market data available.\n")
		b.WriteString("  Request competitive quotes from alternative vendors.\n")
	}
	b.WriteString(fmt.Sprintf("• Volume discount opportunity: %d seats may qualify for tiered pricing.\n", q.Seats))
	if q.TermMonths >= 12 {
		b.WriteString("• Multi-year commitment leverage: use term length as a negotiation chip.\n")
	}
	b.WriteString("\n")

	return b.String(), nil
}

// extractVendor finds a known vendor name in the raw text.
func (e *Engine) extractVendor(rawText string) string {
	if rawText == "" {
		return ""
	}
	match := vendorPattern.FindString(rawText)
	if match == "" {
		return ""
	}
	// Capitalize properly
	for _, v := range KnownVendors {
		if strings.EqualFold(match, v) {
			return v
		}
	}
	return match
}

// extractQuantity finds the number of seats/users/license count.
func (e *Engine) extractQuantity(rawText string) int {
	if rawText == "" {
		return 0
	}
	match := quantityPattern.FindStringSubmatch(rawText)
	if len(match) >= 2 {
		n, err := strconv.Atoi(match[1])
		if err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// extractPrice finds the first dollar amount in the text.
func (e *Engine) extractPrice(rawText string) float64 {
	if rawText == "" {
		return 0
	}
	match := pricePattern.FindStringSubmatch(rawText)
	if len(match) >= 2 {
		cleaned := strings.ReplaceAll(match[1], ",", "")
		p, err := strconv.ParseFloat(cleaned, 64)
		if err == nil && p > 0 {
			return p
		}
	}
	return 0
}

// extractTerm determines the contract term in months from text.
// Prefers annual/yearly matches over monthly matches when both are present.
func (e *Engine) extractTerm(rawText string) int {
	if rawText == "" {
		return 1
	}

	// Check for annual indicators first (preferred)
	if annualTermPattern.MatchString(rawText) {
		return 12
	}

	// Check for monthly indicators
	if monthlyTermPattern.MatchString(rawText) {
		return 1
	}

	// Default to monthly
	return 1
}
