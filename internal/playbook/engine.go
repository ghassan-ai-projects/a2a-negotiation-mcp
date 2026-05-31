package playbook

import (
	"fmt"
	"strings"
)

// Engine generates negotiation playbooks.
type Engine struct{}

// NewEngine creates a new playbook engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Generate produces a complete negotiation playbook with 5 sections.
func (e *Engine) Generate() (*Playbook, error) {
	sections := e.buildSections()
	content := e.renderMarkdown(sections)
	return &Playbook{
		Content:  content,
		Sections: sections,
	}, nil
}

func (e *Engine) buildSections() []PlaybookSection {
	return []PlaybookSection{
		{
			Title: "Available Strategies",
			Items: []string{
				"**Cost-Plus Negotiation** — Request a breakdown of underlying costs and negotiate a fixed margin above those costs. Most effective with infrastructure vendors.",
				"**Value-Based Negotiation** — Anchor the discussion on the measurable value the vendor provides to your business. Use ROI data to justify pricing.",
				"**Competitive Leverage** — Use competing vendor quotes as leverage. Gather 2-3 comparable quotes before engaging.",
				"**Package Consolidation** — Bundle multiple SKUs or services under a single agreement to unlock volume discounts.",
				"**Multi-Year Commit** — Offer a longer contract term in exchange for reduced annual pricing. Typically yields 10-20% additional savings.",
			},
		},
		{
			Title: "Best Practices by Vendor Category",
			Items: []string{
				"**SaaS** — Focus on per-seat pricing tiers. Negotiate unused license credits. Request annual caps on price increases. Target 15-30% discount from list.",
				"**Infrastructure** — Emphasize committed use discounts and reserved instances. Look for free data transfer tiers. Target 20-40% discount from on-demand pricing.",
				"**Professional Services** — Negotiate fixed-price scopes rather than T&M. Request junior resource rates for routine work. Target 5-15% discount from standard rates.",
				"**Marketing / Advertising** — Negotiate performance-based bonuses. Request transparent reporting and attribution. Target 10-25% added value through over-delivery.",
			},
		},
		{
			Title: "Common Tactics",
			Items: []string{
				"**Anchoring** — Start with an ambitious but credible first offer to set the negotiation range in your favor.",
				"**BATNA** — Know your Best Alternative To a Negotiated Agreement. A strong BATNA gives you the confidence to walk away.",
				"**Silence** — After making an offer or counter-offer, stay silent. The next person to speak often concedes ground.",
				"**Trade-Offs** — Never give a concession without asking for something in return. Frame it as \"If I can do X, will you do Y?\".",
				"**Good Cop / Bad Cop** — Use a team split: one person takes a hard line on price while another builds rapport on value.",
			},
		},
		{
			Title: "Price Benchmarks",
			Items: []string{
				"**SaaS (per-seat)** — Typical discount range: 10-30%. Enterprise agreements can reach 40%+ for 500+ seats.",
				"**Cloud Infrastructure** — Committed use discounts: 20-50% vs on-demand. 3-year commits offer deeper savings than 1-year.",
				"**Professional Services** — Standard rates: $150-350/hr. Discount range: 5-15% for blocks of 100+ hours.",
				"**Marketing Platforms** — Ad spend commitments of $50K+/month typically unlock 10-25% in bonus value or reduced fees.",
				"**AI / ML APIs** — Volume tiers typically offer 20-40% discounts at enterprise scale. Negotiate custom tiers for >$100K/month spend.",
			},
		},
		{
			Title: "Vendor-Specific Tips",
			Items: []string{
				"**Lock-In Risk** — Evaluate data portability and integration dependencies before committing. Request guaranteed API access and data export SLAs.",
				"**Flexibility Needs** — Negotiate ramp-up / ramp-down clauses for variable headcount. Avoid fixed minimum seat commitments where possible.",
				"**Support Quality** — Don't accept standard support. Negotiate response-time SLAs, dedicated account management, and escalation paths.",
				"**Auto-Renewal** — Always remove auto-renewal clauses or negotiate 90+ day notice periods. Set calendar reminders 60 days before renewal.",
				"**Audit Rights** — Limit vendor audit frequency to once per year and require 30-day advance notice. Cap retroactive true-ups to 12 months.",
			},
		},
	}
}

func (e *Engine) renderMarkdown(sections []PlaybookSection) string {
	var b strings.Builder

	b.WriteString("# Negotiation Playbook\n\n")
	b.WriteString("Generated playbook with strategies, tactics, benchmarks, and vendor-specific guidance.\n\n---\n\n")

	for _, section := range sections {
		fmt.Fprintf(&b, "## %s\n\n", section.Title)
		for _, item := range section.Items {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		b.WriteString("\n---\n\n")
	}

	b.WriteString("> **Disclaimer:** This playbook provides general guidance. Always consult your procurement and legal teams before finalizing agreements.\n")

	return b.String()
}
