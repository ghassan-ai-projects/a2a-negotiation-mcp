package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
)

// Engine generates data exports from the history store.
type Engine struct {
	store        *Store
	historyStore *history.Store
	logger       *slog.Logger
}

// NewEngine creates an export engine.
func NewEngine(store *Store, historyStore *history.Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, historyStore: historyStore, logger: logger}
}

// Export generates export data in the requested format and type.
func (e *Engine) Export(ctx context.Context, req ExportRequest) (*ExportResult, error) {
	e.logger.Debug("export called", "format", req.Format, "type", req.Type, "vendor", req.Vendor)

	if req.Format == "" {
		req.Format = "csv"
	}
	if req.Type == "" {
		req.Type = "deals"
	}
	if req.Format != "csv" && req.Format != "json" {
		return nil, fmt.Errorf("invalid format %q: use csv or json", req.Format)
	}

	var data string
	var recordCount int
	var filename string
	now := time.Now().UTC()
	timestamp := now.Format("20060102_150405")

	switch req.Type {
	case "deals":
		deals, err := e.queryDeals(ctx, req.Vendor, req.DateFrom, req.DateTo)
		if err != nil {
			return nil, fmt.Errorf("query deals: %w", err)
		}
		recordCount = len(deals)
		if recordCount == 0 {
			data = emptyOutput(req.Format, "deals")
		} else if req.Format == "csv" {
			data = dealsToCSV(deals)
		} else {
			data = dealsToJSON(deals)
		}
		filename = fmt.Sprintf("a2a-export-deals-%s.%s", timestamp, req.Format)

	case "sessions":
		sessions, err := e.querySessions(ctx, req.Vendor, req.DateFrom, req.DateTo)
		if err != nil {
			return nil, fmt.Errorf("query sessions: %w", err)
		}
		recordCount = len(sessions)
		if recordCount == 0 {
			data = emptyOutput(req.Format, "sessions")
		} else if req.Format == "csv" {
			data = sessionsToCSV(sessions)
		} else {
			data = sessionsToJSON(sessions)
		}
		filename = fmt.Sprintf("a2a-export-sessions-%s.%s", timestamp, req.Format)

	case "analytics":
		summary, err := e.historyStore.GetHistory(ctx, req.Vendor, "")
		if err != nil {
			return nil, fmt.Errorf("query analytics: %w", err)
		}
		analytics := map[string]any{
			"total_deals":             summary.TotalDeals,
			"win_rate":                summary.WinRate,
			"avg_discount_percentage": summary.AvgDiscountPct,
			"total_savings":           summary.TotalSavings,
		}
		if req.Format == "csv" {
			data = analyticsToCSV(analytics)
		} else {
			b, _ := json.MarshalIndent(analytics, "", "  ")
			data = string(b)
		}
		recordCount = 1
		filename = fmt.Sprintf("a2a-export-analytics-%s.%s", timestamp, req.Format)

	case "all":
		deals, _ := e.queryDeals(ctx, req.Vendor, req.DateFrom, req.DateTo)
		sessions, _ := e.querySessions(ctx, req.Vendor, req.DateFrom, req.DateTo)

		allData := map[string]any{
			"deals":     deals,
			"sessions":  sessions,
			"exported_at": now.Format(time.RFC3339),
		}
		if req.Format == "csv" {
			// For "all" in CSV, concatenate deals CSV + sessions CSV with a separator
			var parts []string
			parts = append(parts, "# DEALS")
			parts = append(parts, dealsToCSV(deals))
			parts = append(parts, "# SESSIONS")
			parts = append(parts, sessionsToCSV(sessions))
			data = strings.Join(parts, "\n")
		} else {
			b, _ := json.MarshalIndent(allData, "", "  ")
			data = string(b)
		}
		recordCount = len(deals) + len(sessions)
		filename = fmt.Sprintf("a2a-export-all-%s.%s", timestamp, req.Format)

	default:
		return nil, fmt.Errorf("invalid export type %q: use deals, sessions, analytics, or all", req.Type)
	}

	exportID, err := e.store.SaveExport(ctx, "", req.Format, req.Type, recordCount)
	if err != nil {
		e.logger.Warn("failed to save export metadata", "error", err.Error())
	}

	return &ExportResult{
		ExportID:    exportID,
		Format:      req.Format,
		ExportType:  req.Type,
		RecordCount: recordCount,
		Data:        data,
		Filename:    filename,
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

// queryDeals retrieves deal outcomes from the history store.
func (e *Engine) queryDeals(ctx context.Context, vendor, dateFrom, dateTo string) ([]history.DealOutcome, error) {
	return e.queryDealOutcomes(ctx, vendor, dateFrom, dateTo)
}

func (e *Engine) queryDealOutcomes(ctx context.Context, vendor, dateFrom, dateTo string) ([]history.DealOutcome, error) {
	where := ""
	args := []any{}

	if vendor != "" {
		where += " AND vendor = ?"
		args = append(args, vendor)
	}
	if dateFrom != "" {
		where += " AND created_at >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		where += " AND created_at <= ?"
		args = append(args, dateTo)
	}

	query := `SELECT vendor, sku, list_price, final_price, discount_pct, seats, term_months, strategy, session_id, created_at
		FROM deal_outcomes WHERE 1=1` + where + ` ORDER BY created_at DESC`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query deal outcomes: %w", err)
	}
	defer rows.Close()

	var deals []history.DealOutcome
	for rows.Next() {
		var d history.DealOutcome
		var createdAt string
		if err := rows.Scan(&d.Vendor, &d.SKU, &d.ListPrice, &d.FinalPrice, &d.DiscountPct,
			&d.Seats, &d.TermMonths, &d.Strategy, &d.SessionID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan deal: %w", err)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deals = append(deals, d)
	}
	return deals, rows.Err()
}

// querySessions retrieves negotiation sessions from the history store.
func (e *Engine) querySessions(ctx context.Context, vendor, dateFrom, dateTo string) ([]history.SessionRecord, error) {
	where := ""
	args := []any{}

	if vendor != "" {
		where += " AND vendor = ?"
		args = append(args, vendor)
	}
	if dateFrom != "" {
		where += " AND created_at >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		where += " AND created_at <= ?"
		args = append(args, dateTo)
	}

	query := `SELECT id, vendor, sku, strategy, budget, status, current_offer, list_price,
		rounds_complete, outcome, created_at, updated_at
		FROM negotiation_sessions WHERE 1=1` + where + ` ORDER BY created_at DESC`
	rows, err := e.historyStore.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []history.SessionRecord
	for rows.Next() {
		var s history.SessionRecord
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.Vendor, &s.SKU, &s.Strategy, &s.Budget, &s.Status,
			&s.CurrentOffer, &s.ListPrice, &s.RoundsComplete, &s.Outcome, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// ─── CSV formatters ───

func dealsToCSV(deals []history.DealOutcome) string {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"vendor", "sku", "list_price", "final_price", "discount_pct", "seats", "term_months", "strategy", "session_id", "created_at"})
	for _, d := range deals {
		w.Write([]string{
			d.Vendor, d.SKU,
			fmt.Sprintf("%.2f", d.ListPrice),
			fmt.Sprintf("%.2f", d.FinalPrice),
			fmt.Sprintf("%.2f", d.DiscountPct),
			fmt.Sprintf("%d", d.Seats),
			fmt.Sprintf("%d", d.TermMonths),
			d.Strategy, d.SessionID,
			d.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	return buf.String()
}

func sessionsToCSV(sessions []history.SessionRecord) string {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"id", "vendor", "sku", "strategy", "budget", "status", "current_offer", "list_price", "rounds_complete", "outcome", "created_at", "updated_at"})
	for _, s := range sessions {
		w.Write([]string{
			s.ID, s.Vendor, s.SKU, s.Strategy,
			fmt.Sprintf("%.2f", s.Budget),
			s.Status,
			fmt.Sprintf("%.2f", s.CurrentOffer),
			fmt.Sprintf("%.2f", s.ListPrice),
			fmt.Sprintf("%d", s.RoundsComplete),
			s.Outcome,
			s.CreatedAt.Format(time.RFC3339),
			s.UpdatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	return buf.String()
}

func analyticsToCSV(analytics map[string]any) string {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Write([]string{"metric", "value"})
	for k, v := range analytics {
		w.Write([]string{k, fmt.Sprintf("%v", v)})
	}
	w.Flush()
	return buf.String()
}

func emptyOutput(format string, exportType string) string {
	if format == "csv" {
		return fmt.Sprintf("no_%s_found", exportType)
	}
	return fmt.Sprintf(`{"type": %q, "count": 0}`, exportType)
}

// ─── JSON formatters ───

func dealsToJSON(deals []history.DealOutcome) string {
	type dealView struct {
		Vendor      string  `json:"vendor"`
		SKU         string  `json:"sku"`
		ListPrice   float64 `json:"list_price"`
		FinalPrice  float64 `json:"final_price"`
		DiscountPct float64 `json:"discount_percentage"`
		Seats       int     `json:"seats"`
		TermMonths  int     `json:"term_months"`
		Strategy    string  `json:"strategy"`
		CreatedAt   string  `json:"created_at"`
	}
	views := make([]dealView, len(deals))
	for i, d := range deals {
		views[i] = dealView{
			Vendor: d.Vendor, SKU: d.SKU,
			ListPrice: d.ListPrice, FinalPrice: d.FinalPrice,
			DiscountPct: d.DiscountPct, Seats: d.Seats,
			TermMonths: d.TermMonths, Strategy: d.Strategy,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		}
	}
	b, _ := json.MarshalIndent(views, "", "  ")
	return string(b)
}

func sessionsToJSON(sessions []history.SessionRecord) string {
	type sessionView struct {
		ID             string  `json:"session_id"`
		Vendor         string  `json:"vendor"`
		SKU            string  `json:"sku"`
		Strategy       string  `json:"strategy"`
		Budget         float64 `json:"budget"`
		Status         string  `json:"status"`
		CurrentOffer   float64 `json:"current_offer"`
		ListPrice      float64 `json:"list_price"`
		RoundsComplete int     `json:"rounds_completed"`
		Outcome        string  `json:"outcome"`
		CreatedAt      string  `json:"created_at"`
		UpdatedAt      string  `json:"updated_at"`
	}
	views := make([]sessionView, len(sessions))
	for i, s := range sessions {
		views[i] = sessionView{
			ID: s.ID, Vendor: s.Vendor, SKU: s.SKU,
			Strategy: s.Strategy, Budget: s.Budget,
			Status: s.Status, CurrentOffer: s.CurrentOffer,
			ListPrice: s.ListPrice, RoundsComplete: s.RoundsComplete,
			Outcome: s.Outcome,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		}
	}
	b, _ := json.MarshalIndent(views, "", "  ")
	return string(b)
}
