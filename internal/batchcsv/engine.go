package batchcsv

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Engine is a stateless CSV processor for batch negotiation uploads.
type Engine struct{}

// NewEngine creates a new Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// ProcessCSV parses and validates a raw CSV string, returning a
// BatchUploadResult with the number of rows successfully created and any
// per-row validation errors.
func (e *Engine) ProcessCSV(ctx context.Context, csvContent string) (*BatchUploadResult, error) {
	if strings.TrimSpace(csvContent) == "" {
		return &BatchUploadResult{
			Errors: []string{"csv content is empty"},
		}, nil
	}

	lines := strings.Split(csvContent, "\n")
	if len(lines) < 2 {
		return &BatchUploadResult{
			Errors: []string{"csv must contain a header row and at least one data row"},
		}, nil
	}

	// Validate header
	header := strings.TrimSpace(lines[0])
	expectedHeader := "vendor,strategy,budget,target_price,notes"
	if header != expectedHeader {
		return &BatchUploadResult{
			Errors: []string{fmt.Sprintf("invalid header: got %q, expected %q", header, expectedHeader)},
		}, nil
	}

	var errors []string
	createdCount := 0
	dataLines := 0

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue // skip blank rows
		}
		dataLines++

		cols := strings.Split(line, ",")

		rowNum := i + 1 // 1-indexed for error messages
		row, err := parseRow(cols)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: %s", rowNum, err))
			continue
		}

		// Validate vendor
		if strings.TrimSpace(row.Vendor) == "" {
			errors = append(errors, fmt.Sprintf("row %d: vendor is required", rowNum))
			continue
		}

		// Validate strategy
		validStrategies := map[string]bool{
			"competitive":   true,
			"collaborative": true,
			"aggressive":    true,
		}
		strategy := strings.TrimSpace(row.Strategy)
		if !validStrategies[strategy] {
			errors = append(errors, fmt.Sprintf("row %d: invalid strategy %q: must be one of [competitive, collaborative, aggressive]", rowNum, strategy))
			continue
		}

		// Validate budget positive
		if row.Budget <= 0 {
			errors = append(errors, fmt.Sprintf("row %d: budget must be a positive number, got %.2f", rowNum, row.Budget))
			continue
		}

		createdCount++
	}

	result := &BatchUploadResult{
		CreatedCount: createdCount,
		RowCount:     dataLines,
		Errors:       errors,
	}
	return result, nil
}

// parseRow converts a slice of CSV columns into a CSVRow.
func parseRow(cols []string) (CSVRow, error) {
	if len(cols) < 4 {
		return CSVRow{}, fmt.Errorf("expected at least 4 columns (vendor,strategy,budget,target_price), got %d", len(cols))
	}

	budget, err := strconv.ParseFloat(strings.TrimSpace(cols[2]), 64)
	if err != nil {
		return CSVRow{}, fmt.Errorf("invalid budget %q: %v", cols[2], err)
	}

	targetPrice, err := strconv.ParseFloat(strings.TrimSpace(cols[3]), 64)
	if err != nil {
		return CSVRow{}, fmt.Errorf("invalid target_price %q: %v", cols[3], err)
	}

	notes := ""
	if len(cols) >= 5 {
		notes = strings.TrimSpace(cols[4])
	}

	return CSVRow{
		Vendor:      strings.TrimSpace(cols[0]),
		Strategy:    strings.TrimSpace(cols[1]),
		Budget:      budget,
		TargetPrice: targetPrice,
		Notes:       notes,
	}, nil
}
