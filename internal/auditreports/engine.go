package auditreports

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// GenerateReport generates a simulated audit report for the given period and format.
// from/to must be YYYY-MM-DD. format must be "json" or "csv".
func (e *Engine) GenerateReport(ctx context.Context, from, to, format string) (*AuditReport, error) {
	if _, err := time.Parse(time.DateOnly, from); err != nil {
		return nil, fmt.Errorf("invalid from date %q: must be YYYY-MM-DD", from)
	}
	if _, err := time.Parse(time.DateOnly, to); err != nil {
		return nil, fmt.Errorf("invalid to date %q: must be YYYY-MM-DD", to)
	}
	format = strings.ToLower(format)
	if format != "json" && format != "csv" {
		return nil, fmt.Errorf("invalid format %q: must be json or csv", format)
	}

	rowCount := 12 + rand.Intn(18) // 12-29 rows
	negCount := rowCount
	totalSavings := 12500.0 + rand.Float64()*35000
	avgDiscount := 12.0 + rand.Float64()*18

	var data string
	switch format {
	case "json":
		data = buildJSONData(from, to, rowCount, totalSavings, avgDiscount)
	case "csv":
		data = buildCSVData(from, to, rowCount, totalSavings, avgDiscount)
	}

	return &AuditReport{
		PeriodFrom: from,
		PeriodTo:   to,
		Format:     format,
		Data:       data,
		RowCount:   rowCount,
		Summary: AuditSummary{
			TotalNegotiations: negCount,
			TotalSavings:      math.Round(totalSavings*100) / 100,
			AvgDiscount:       math.Round(avgDiscount*100) / 100,
			PeriodDescription: fmt.Sprintf("%s to %s", from, to),
		},
	}, nil
}

// GetAuditSummary returns a simulated audit summary for the given period.
// period must be one of: 30d, 90d, 1y, all.
func (e *Engine) GetAuditSummary(ctx context.Context, period string) (*AuditSummary, error) {
	period = strings.ToLower(period)
	var negCount int
	var totalSavings float64
	var avgDiscount float64
	var desc string

	switch period {
	case "30d":
		negCount = 8 + rand.Intn(7)
		totalSavings = 3200 + rand.Float64()*4800
		avgDiscount = 10 + rand.Float64()*8
		desc = "Last 30 days"
	case "90d":
		negCount = 25 + rand.Intn(15)
		totalSavings = 9800 + rand.Float64()*10200
		avgDiscount = 12 + rand.Float64()*10
		desc = "Last 90 days"
	case "1y":
		negCount = 120 + rand.Intn(60)
		totalSavings = 45000 + rand.Float64()*35000
		avgDiscount = 14 + rand.Float64()*10
		desc = "Last 12 months"
	case "all":
		negCount = 420 + rand.Intn(180)
		totalSavings = 185000 + rand.Float64()*115000
		avgDiscount = 16 + rand.Float64()*8
		desc = "All time"
	default:
		return nil, fmt.Errorf("invalid period %q: must be 30d, 90d, 1y, or all", period)
	}

	return &AuditSummary{
		TotalNegotiations: negCount,
		TotalSavings:      math.Round(totalSavings*100) / 100,
		AvgDiscount:       math.Round(avgDiscount*100) / 100,
		PeriodDescription: desc,
	}, nil
}

// GetAuditTrail returns simulated audit trail entries for a given entity.
func (e *Engine) GetAuditTrail(ctx context.Context, entityType, entityID string) ([]AuditTrailEntry, error) {
	if entityType == "" {
		return nil, fmt.Errorf("entity_type is required")
	}
	if entityID == "" {
		return nil, fmt.Errorf("entity_id is required")
	}

	actions := []string{"created", "updated", "viewed", "archived", "restored", "deleted", "exported", "shared"}
	entryCount := 3 + rand.Intn(8)
	entries := make([]AuditTrailEntry, entryCount)

	now := time.Now().UTC()
	for i := range entries {
		entries[i] = AuditTrailEntry{
			EntityType: entityType,
			EntityID:   entityID,
			Action:     actions[rand.Intn(len(actions))],
			Timestamp:  now.Add(-time.Duration(entryCount-i) * 24 * time.Hour).Format(time.RFC3339),
		}
	}

	return entries, nil
}

func buildJSONData(from, to string, rowCount int, totalSavings float64, avgDiscount float64) string {
	var b strings.Builder
	b.WriteString("[\n")
	for i := 0; i < rowCount; i++ {
		vendor := vendors[rand.Intn(len(vendors))]
		amount := 500 + rand.Float64()*4500
		discount := avgDiscount * (0.5 + rand.Float64())
		saving := amount * discount / 100
		b.WriteString(fmt.Sprintf(`  {"vendor":%q,"category":%q,"amount":%.2f,"saving":%.2f,"discount_pct":%.1f}`,
			vendor, categories[rand.Intn(len(categories))], amount, saving, discount))
		if i < rowCount-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]")
	return b.String()
}

func buildCSVData(from, to string, rowCount int, totalSavings float64, avgDiscount float64) string {
	var b strings.Builder
	b.WriteString("vendor,category,amount,saving,discount_pct\n")
	for i := 0; i < rowCount; i++ {
		vendor := vendors[rand.Intn(len(vendors))]
		amount := 500 + rand.Float64()*4500
		discount := avgDiscount * (0.5 + rand.Float64())
		saving := amount * discount / 100
		b.WriteString(fmt.Sprintf("%s,%s,%.2f,%.2f,%.1f\n",
			vendor, categories[rand.Intn(len(categories))], amount, saving, discount))
	}
	return b.String()
}

var vendors = []string{
	"Slack", "GitHub", "Salesforce", "OpenAI", "Anthropic",
	"Google", "DeepSeek", "Mistral", "Datadog", "Snowflake",
	"AWS", "Azure", "Atlassian", "Twilio", "HubSpot",
}

var categories = []string{
	"Communication", "Developer", "CRM", "AI", "Observability",
	"Cloud", "Infrastructure", "Productivity", "Messaging", "Marketing",
}
