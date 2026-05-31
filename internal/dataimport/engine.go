package dataimport

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
	"github.com/google/uuid"
)

// Engine handles data import and validation.
type Engine struct {
	pricingStore *pricing.Store
	historyStore *history.Store
}

// NewEngine creates a data import engine.
func NewEngine(pricingStore *pricing.Store, historyStore *history.Store) *Engine {
	return &Engine{
		pricingStore: pricingStore,
		historyStore: historyStore,
	}
}

// Validate parses and validates import data without inserting.
func (e *Engine) Validate(ctx context.Context, req ImportRequest) (*ImportResult, error) {
	switch req.Type {
	case ImportTypeDeals:
		return e.validateDeals(ctx, req.Data)
	case ImportTypePricing:
		return e.validatePricing(ctx, req.Data)
	default:
		return &ImportResult{
			Errors:  []string{fmt.Sprintf("unknown import type: %s", req.Type)},
			Summary: "Validation failed: unknown import type",
		}, nil
	}
}

// Import validates and inserts parsed records.
func (e *Engine) Import(ctx context.Context, req ImportRequest) (*ImportResult, error) {
	if req.DryRun {
		return e.Validate(ctx, req)
	}

	switch req.Type {
	case ImportTypeDeals:
		return e.importDeals(ctx, req.Data)
	case ImportTypePricing:
		return e.importPricing(ctx, req.Data)
	default:
		return &ImportResult{
			Errors:  []string{fmt.Sprintf("unknown import type: %s", req.Type)},
			Summary: "Import failed: unknown import type",
		}, nil
	}
}

func (e *Engine) validateDeals(ctx context.Context, data string) (*ImportResult, error) {
	records, errs := e.parseDealsJSON(data)
	validCount := len(records)

	var summary string
	if len(errs) > 0 {
		summary = fmt.Sprintf("Validation: %d valid, %d errors", validCount, len(errs))
	} else {
		summary = fmt.Sprintf("Validation: %d records valid", validCount)
	}

	return &ImportResult{
		ValidCount:    validCount,
		ImportedCount: 0,
		SkippedCount:  0,
		Errors:        errs,
		Summary:       summary,
	}, nil
}

func (e *Engine) validatePricing(ctx context.Context, data string) (*ImportResult, error) {
	records, errs := e.parsePricingJSON(data)
	validCount := len(records)

	var summary string
	if len(errs) > 0 {
		summary = fmt.Sprintf("Validation: %d valid, %d errors", validCount, len(errs))
	} else {
		summary = fmt.Sprintf("Validation: %d records valid", validCount)
	}

	return &ImportResult{
		ValidCount:    validCount,
		ImportedCount: 0,
		SkippedCount:  0,
		Errors:        errs,
		Summary:       summary,
	}, nil
}

func (e *Engine) importDeals(ctx context.Context, data string) (*ImportResult, error) {
	records, parseErrs := e.parseDealsJSON(data)

	var importErrs []string
	importErrs = append(importErrs, parseErrs...)

	importedCount := 0
	skippedCount := 0

	for _, rec := range records {
		outcome := history.DealOutcome{
			Vendor:      rec.Vendor,
			SKU:         rec.SKU,
			ListPrice:   rec.ListPrice,
			FinalPrice:  rec.FinalPrice,
			DiscountPct: rec.DiscountPct,
			Seats:       rec.Seats,
			TermMonths:  rec.TermMonths,
			Strategy:    rec.Strategy,
			SessionID:   uuid.New().String(),
			CreatedAt:   time.Now().UTC(),
		}

		if err := e.historyStore.SaveDealOutcome(ctx, &outcome); err != nil {
			importErrs = append(importErrs, fmt.Sprintf("import deal %s/%s: %s", rec.Vendor, rec.SKU, err))
			skippedCount++
			continue
		}
		importedCount++
	}

	var summary string
	if len(importErrs) > 0 {
		summary = fmt.Sprintf("Imported %d deals, skipped %d, %d errors", importedCount, skippedCount, len(importErrs))
	} else {
		summary = fmt.Sprintf("Imported %d deals successfully", importedCount)
	}

	return &ImportResult{
		ValidCount:    len(records),
		ImportedCount: importedCount,
		SkippedCount:  skippedCount,
		Errors:        importErrs,
		Summary:       summary,
	}, nil
}

func (e *Engine) importPricing(ctx context.Context, data string) (*ImportResult, error) {
	records, parseErrs := e.parsePricingJSON(data)

	var importErrs []string
	importErrs = append(importErrs, parseErrs...)

	importedCount := 0
	skippedCount := 0
	ctxBG := context.Background()

	for _, rec := range records {
		vendorID, err := e.pricingStore.GetVendorID(ctxBG, rec.Vendor)
		if err != nil {
			// Insert vendor if not exists
			_, err := e.pricingStore.DB().ExecContext(ctxBG,
				"INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)",
				rec.Vendor, rec.Category)
			if err != nil {
				importErrs = append(importErrs, fmt.Sprintf("insert vendor %s: %s", rec.Vendor, err))
				skippedCount++
				continue
			}
			vendorID, err = e.pricingStore.GetVendorID(ctxBG, rec.Vendor)
			if err != nil {
				importErrs = append(importErrs, fmt.Sprintf("get vendor id %s: %s", rec.Vendor, err))
				skippedCount++
				continue
			}
		}

		_, err = e.pricingStore.DB().ExecContext(ctxBG, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET
				list_price=excluded.list_price,
				min_observed=excluded.min_observed,
				max_observed=excluded.max_observed,
				typical_pct=excluded.typical_pct,
				description=excluded.description,
				updated_at=datetime('now')
		`, vendorID, rec.SKU, rec.Description, rec.ListPrice, rec.MinObserved, rec.MaxObserved, rec.TypicalPct, rec.Unit)
		if err != nil {
			importErrs = append(importErrs, fmt.Sprintf("import pricing %s/%s: %s", rec.Vendor, rec.SKU, err))
			skippedCount++
			continue
		}
		importedCount++
	}

	var summary string
	if len(importErrs) > 0 {
		summary = fmt.Sprintf("Imported %d pricing records, skipped %d, %d errors", importedCount, skippedCount, len(importErrs))
	} else {
		summary = fmt.Sprintf("Imported %d pricing records successfully", importedCount)
	}

	return &ImportResult{
		ValidCount:    len(records),
		ImportedCount: importedCount,
		SkippedCount:  skippedCount,
		Errors:        importErrs,
		Summary:       summary,
	}, nil
}

func (e *Engine) parseDealsJSON(data string) ([]dealRecord, []string) {
	var records []dealRecord
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return nil, []string{fmt.Sprintf("invalid JSON: %s", err)}
	}

	var errs []string
	var valid []dealRecord
	for i, rec := range records {
		if rec.Vendor == "" {
			errs = append(errs, fmt.Sprintf("record %d: vendor is required", i))
			continue
		}
		if rec.ListPrice <= 0 {
			errs = append(errs, fmt.Sprintf("record %d: list_price must be positive", i))
			continue
		}
		if rec.FinalPrice <= 0 {
			errs = append(errs, fmt.Sprintf("record %d: final_price must be positive", i))
			continue
		}
		valid = append(valid, rec)
	}
	if valid == nil {
		valid = []dealRecord{}
	}
	return valid, errs
}

func (e *Engine) parsePricingJSON(data string) ([]pricingRecord, []string) {
	var records []pricingRecord
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return nil, []string{fmt.Sprintf("invalid JSON: %s", err)}
	}

	var errs []string
	var valid []pricingRecord
	for i, rec := range records {
		if rec.Vendor == "" {
			errs = append(errs, fmt.Sprintf("record %d: vendor is required", i))
			continue
		}
		if rec.SKU == "" {
			errs = append(errs, fmt.Sprintf("record %d: sku is required", i))
			continue
		}
		if rec.ListPrice <= 0 {
			errs = append(errs, fmt.Sprintf("record %d: list_price must be positive", i))
			continue
		}
		valid = append(valid, rec)
	}
	if valid == nil {
		valid = []pricingRecord{}
	}
	return valid, errs
}
