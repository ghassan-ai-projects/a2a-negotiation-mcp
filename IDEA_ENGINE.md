# Idea Generation Engine — a2a-negotiation-mcp

A systematic process for continuously generating high-quality MCP tool ideas for the negotiation server.

## Process

### 1. Domain Scan (Weekly)
**Goal:** Identify uncovered categories by scanning against a feature matrix.

**Matrix dimensions to check:**
- **CRUD lifecycle:** Create → Read → Update → Delete → List → Search → Export → Import
- **Entity types:** Vendors, Negotiations, Contracts, Pricing, Savings, Users, Teams
- **Categories:** Core, AI/ML, Integration, Security, Analytics, Admin, Visualization, Automation, Compliance, Gamification, Data Quality
- **Layers:** Input → Processing → Storage → Analysis → Output → Notification → Automation

**For each empty cell in the matrix → it's a potential new tool.**

### 2. Category Expansion
**Goal:** For each existing feature category, find 2-3 adjacent ideas.

**Questions to ask:**
- What's the next level of detail? (report → detail → breakdown → drill-down)
- What's the related entity? (vendor → contract → SLA → penalty)
- What's the next action? (analyze → optimize → automate → schedule)
- What's the comparison? (compare → rank → benchmark → percentile)

### 3. Pattern Inversion
**Goal:** Invert existing tools to find gaps.

**Existing:** `negotiate_compare_vendors`
**Inversion:** `negotiate_compare_categories`, `negotiate_compare_strategies`, `negotiate_compare_periods`

**Existing:** `negotiate_save_report`
**Inversion:** `negotiate_publish_report`, `negotiate_schedule_report`, `negotiate_star_report`

### 4. Integration Mapping
**Goal:** Map every major SaaS/enterprise tool the project could integrate with.

**External platforms to scan:**
- CRM: Salesforce, HubSpot, Zoho
- Project Mgmt: Jira, Asana, Monday.com, Trello
- Comms: Slack, Teams, Discord, Email
- Docs: Confluence, Google Workspace, Notion
- DevOps: Datadog, PagerDuty, Sentry, New Relic
- Finance: Stripe, QuickBooks, Xero, SAP
- Automation: Zapier, Make, n8n

**For each: mind map 3-5 integration touchpoints.**

### 5. Market Gap Analysis
**Goal:** Identify what competing negotiation tools don't offer.

**Competitors to analyze:** Vendr, Zip, PurchaseControl, Coupa, GEP

**Questions:** What features would make this unique? What's the "AI-native" version of a legacy feature?

### 6. Quality Gate
**Each new idea must pass:**
- [ ] Is it a clear MCP tool (request → response)?
- [ ] Does it have a unique name (not duplicate)?
- [ ] Is it implementable in one package (types.go + engine.go)?
- [ ] Does it add user value (not just noise)?
- [ ] Can it be tested?

## Trigger Schedule

| Trigger | Frequency | Method |
|---------|-----------|--------|
| Domain scan | Weekly (Monday) | Run engine script |
| Integration mapping | Bi-weekly | Review new SaaS products |
| Pattern inversion | Per batch (after each 10 features) | Manual review |
| Market gap | Monthly | Competitive research |
| Quality gate | Every idea | Before GitHub issue creation |

## Template for New Ideas

```markdown
### Category: [Category]
**Tool name:** `negotiate_[name]`
**Description:** [1-sentence value prop]
**Paradigm:** [CRUD | Analysis | Automation | Integration | Admin]
**Implementation:** [Store needed? | Engine only? | Config only?]
**Priority:** [High | Medium | Low]
**Source:** [Domain scan | Inversion | Integration | Market gap]
```

## Output Quality Standard

Each batch of 10 ideas should have:
- ≥7 HIGH/MEDIUM relevance
- ≤1 REDUNDANT
- At least 2 from integration mapping
- At least 2 from pattern inversion
- At least 2 from domain scan
