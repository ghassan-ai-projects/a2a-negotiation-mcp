package contract

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/calendar"
)

// ─── Regex Patterns ───

// datePatterns matches common date formats.
// Groups: 1=full match, named groups for month/day/year.
var (
	dateLongPattern = regexp.MustCompile(`(?i)(january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\s+(\d{1,2}),?\s+(\d{4})`)

	dateShortPattern = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`)

	dateISOPattern = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)

	// durationPattern matches term lengths like "12-month", "3 year", "24 month"
	durationPattern = regexp.MustCompile(`(\d+)\s*(?:-|\s)?(month|year|yr)\b`)

	// autoRenewPattern matches auto-renewal indicators
	autoRenewPattern = regexp.MustCompile(`(?i)(automatically\s+renew|auto-renews?|auto\s+renewal|renews\s+automatically)`)

	// terminationPattern matches notice periods like "30 days notice", "60-day notice", "cancel within X days"
	terminationPattern = regexp.MustCompile(`(?i)(?:cancel|cancelled|terminat(?:e|ion|ing)?|notice)\s+(?:within\s+)?(\d+)\s*(?:-|\s)?day`)
	terminationPattern2 = regexp.MustCompile(`(?i)(\d+)\s*(?:-|\s)?days?\s+(?:notice|cancel|cancellation)`)

	// priceLockPattern matches price lock / guarantee periods
	priceLockPattern = regexp.MustCompile(`(?i)(?:price\s+(?:guaranteed|locked|lock|frozen|fixed)|rate\s+lock)\s+(?:for\s+)?(?:(?:the\s+)?(?:first\s+)?)?(\d+)\s*(?:-|\s)?(?:months?|years?)`)

	// pricePattern matches dollar amounts like $145.00, $1,234.56, $165
	pricePattern = regexp.MustCompile(`\$([\d,]+(?:\.\d{2})?)`)

	// perSeatPattern identifies per-seat/user pricing units
	perSeatPattern = regexp.MustCompile(`(?i)per\s+(seat|user|license|employee|member)`)

	// annualTermPattern matches annual billing
	annualTermPattern = regexp.MustCompile(`(?i)(annually|annual|per\s+year|/year|year\s+term)`)

	// monthlyTermPattern matches monthly billing
	monthlyTermPattern = regexp.MustCompile(`(?i)(monthly|per\s+month|/month)`)

	// dataPortabilityPattern matches data portability / export clauses
	dataPortabilityPattern = regexp.MustCompile(`(?i)(data\s+(portability|export|access)|portability\s+clause)`)
)

// ─── Engine ───

// Engine provides contract term extraction and calendar population.
type Engine struct {
	calendarEngine *calendar.Engine
	logger         *slog.Logger
}

// NewEngine creates a new contract Engine.
func NewEngine(calendarEngine *calendar.Engine, logger *slog.Logger) *Engine {
	return &Engine{
		calendarEngine: calendarEngine,
		logger:         logger,
	}
}

// ParseContract extracts contract terms from raw text using rule-based patterns.
func (e *Engine) ParseContract(ctx context.Context, rawText, vendor, sku string) (*ContractParseResult, error) {
	if rawText == "" {
		return &ContractParseResult{
			RawText: rawText,
			Terms:   ContractTerms{Confidence: "low"},
			FieldConf: PerFieldConfidence{
				EndDate:           "low",
				AutoRenew:         "low",
				TerminationNotice: "low",
				PriceLockPeriod:   "low",
				AnnualContract:    "low",
			},
			Warnings: []string{"empty text provided — no terms extracted"},
		}, nil
	}

	result := &ContractParseResult{
		RawText: rawText,
		Terms: ContractTerms{
			Vendor: vendor,
			SKU:    sku,
		},
		FieldConf: PerFieldConfidence{},
	}

	var warnings []string

	// ── Dates ──
	startDate, endDate, dateWarns := extractDates(rawText)
	result.Terms.StartDate = startDate
	result.Terms.EndDate = endDate
	warnings = append(warnings, dateWarns...)

	// Set date confidence
	switch {
	case endDate != "" && startDate != "":
		result.FieldConf.EndDate = "high"
	case endDate != "":
		result.FieldConf.EndDate = "medium"
	default:
		result.FieldConf.EndDate = "low"
		warnings = append(warnings, "no clear end date found — calendar population may be incomplete")
	}

	// ── Duration / Term ──
	termMonths, durationWarns := extractDuration(rawText)
	if termMonths >= 12 {
		result.Terms.AnnualContract = true
		result.FieldConf.AnnualContract = "high"
	} else if termMonths > 0 {
		result.FieldConf.AnnualContract = "medium"
	} else {
		// Fall back to term indicators
		if annualTermPattern.MatchString(rawText) {
			result.Terms.AnnualContract = true
			result.FieldConf.AnnualContract = "medium"
		} else if monthlyTermPattern.MatchString(rawText) {
			result.FieldConf.AnnualContract = "medium"
		} else {
			result.FieldConf.AnnualContract = "low"
		}
	}
	warnings = append(warnings, durationWarns...)

	// ── Auto-Renew ──
	if autoRenewPattern.MatchString(rawText) {
		result.Terms.AutoRenew = true
		result.FieldConf.AutoRenew = "high"
	} else {
		result.FieldConf.AutoRenew = resultsFromBool(false)
	}

	// ── Termination Notice ──
	noticeDays, noticeWarns := extractTerminationNotice(rawText)
	result.Terms.TerminationNotice = noticeDays
	if noticeDays > 0 {
		result.FieldConf.TerminationNotice = "high"
	} else {
		result.FieldConf.TerminationNotice = "low"
		warnings = append(warnings, "no termination notice period found — defaulting to 0")
	}
	warnings = append(warnings, noticeWarns...)

	// ── Price Lock ──
	lockPeriod, lockWarns := extractPriceLock(rawText)
	result.Terms.PriceLockPeriod = lockPeriod
	if lockPeriod != "" {
		result.FieldConf.PriceLockPeriod = "high"
	} else {
		result.FieldConf.PriceLockPeriod = "low"
	}
	warnings = append(warnings, lockWarns...)

	// ── Pricing ──
	extractedPrice, pricingUnit := extractPricing(rawText)
	if extractedPrice > 0 {
		result.FieldConf.Pricing = "high"
	} else {
		result.FieldConf.Pricing = "low"
	}

	// ── Renewal Term Days (default 30 if auto-renew found, else 0) ──
	if result.Terms.AutoRenew {
		result.Terms.RenewalTermDays = 30
	} else if endDate != "" {
		result.Terms.RenewalTermDays = 30
	}

	// ── Data Portability ──
	if dataPortabilityPattern.MatchString(rawText) {
		result.Terms.DataPortability = true
	}

	// ── Overall Confidence ──
	result.Terms.Confidence = computeOverallConfidence(&result.FieldConf)

	// ── Assemble Warnings ──
	if len(warnings) > 0 && warnings[0] == "" {
		warnings = warnings[1:]
	}
	// Deduplicate warnings
	seen := make(map[string]bool)
	unique := make([]string, 0, len(warnings))
	for _, w := range warnings {
		if w != "" && !seen[w] {
			seen[w] = true
			unique = append(unique, w)
		}
	}
	result.Warnings = unique

	// Pricing extract (informational)
	_ = extractedPrice
	_ = pricingUnit

	return result, nil
}

// PopulateCalendar creates a calendar entry from parsed contract terms.
// Calls calendar.Store.CreateContract if end_date is found.
func (e *Engine) PopulateCalendar(ctx context.Context, result *ContractParseResult) error {
	if result.Terms.EndDate == "" {
		return fmt.Errorf("cannot populate calendar: no end date in parsed terms")
	}

	// Parse the end date
	endDate, err := parseDateString(result.Terms.EndDate)
	if err != nil {
		return fmt.Errorf("cannot populate calendar: invalid end date %q: %w", result.Terms.EndDate, err)
	}

	now := time.Now().UTC()

	// Compute start date (default to 1 year before end for annual, 1 month for monthly)
	var startDate time.Time
	if result.Terms.StartDate != "" {
		startDate, err = parseDateString(result.Terms.StartDate)
		if err != nil {
			e.logger.Warn("invalid start date in terms, using computed default", "start_date", result.Terms.StartDate)
			startDate = now
		}
	} else {
		startDate = now
	}

	contract := &calendar.Contract{
		Vendor:       result.Terms.Vendor,
		SKU:          result.Terms.SKU,
		UserID:       "auto",
		Seats:        1,
		CurrentPrice: 0,
		RenewalDate:  endDate,
		Status:       "active",
	}

	if err := e.calendarEngine.Store().CreateContract(ctx, contract); err != nil {
		return fmt.Errorf("create calendar contract: %w", err)
	}

	e.logger.Info("auto-populated renewal calendar",
		"vendor", contract.Vendor,
		"sku", contract.SKU,
		"renewal_date", endDate.Format(time.RFC3339),
		"contract_id", contract.ID,
	)

	result.AutoPopulated = true
	_ = startDate

	return nil
}

// ─── Internal Extraction Helpers ───

// extractDates finds start and end dates in raw text.
func extractDates(rawText string) (startDate, endDate string, warnings []string) {
	var dates []time.Time
	var dateStrs []string

	// Try long-form dates: "January 15, 2026"
	for _, m := range dateLongPattern.FindAllStringSubmatch(rawText, -1) {
		if len(m) >= 4 {
			monthStr := m[1]
			dayStr := m[2]
			yearStr := m[3]
			month := monthNumber(monthStr)
			day, _ := strconv.Atoi(dayStr)
			year, _ := strconv.Atoi(yearStr)
			if month > 0 && day > 0 && year > 0 {
				t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
				dates = append(dates, t)
				dateStrs = append(dateStrs, t.Format("2006-01-02"))
			}
		}
	}

	// Try ISO dates: "2026-01-15"
	if len(dates) < 2 {
		for _, m := range dateISOPattern.FindAllStringSubmatch(rawText, -1) {
			if len(m) >= 4 {
				year, _ := strconv.Atoi(m[1])
				month, _ := strconv.Atoi(m[2])
				day, _ := strconv.Atoi(m[3])
				if year > 0 && month > 0 && day > 0 && month <= 12 && day <= 31 {
					t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
					// Avoid duplicates
					isDup := false
					for _, d := range dates {
						if d.Equal(t) {
							isDup = true
							break
						}
					}
					if !isDup {
						dates = append(dates, t)
						dateStrs = append(dateStrs, t.Format("2006-01-02"))
					}
				}
			}
		}
	}

	// Try short-form dates: "01/15/2026"
	if len(dates) < 2 {
		for _, m := range dateShortPattern.FindAllStringSubmatch(rawText, -1) {
			if len(m) >= 4 {
				month, _ := strconv.Atoi(m[1])
				day, _ := strconv.Atoi(m[2])
				year, _ := strconv.Atoi(m[3])
				if year > 0 && month > 0 && day > 0 && month <= 12 && day <= 31 {
					t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
					isDup := false
					for _, d := range dates {
						if d.Equal(t) {
							isDup = true
							break
						}
					}
					if !isDup {
						dates = append(dates, t)
						dateStrs = append(dateStrs, t.Format("2006-01-02"))
					}
				}
			}
		}
	}

	if len(dates) == 0 {
		return "", "", []string{"no dates found in text"}
	}

	// If only one date found, treat it as the end date
	if len(dates) == 1 {
		return "", dateStrs[0], nil
	}

	// Multiple dates: earliest is start, latest is end
	earliest := dates[0]
	latest := dates[0]
	earliestStr := dateStrs[0]
	latestStr := dateStrs[0]
	for i := 1; i < len(dates); i++ {
		if dates[i].Before(earliest) {
			earliest = dates[i]
			earliestStr = dateStrs[i]
		}
		if dates[i].After(latest) {
			latest = dates[i]
			latestStr = dateStrs[i]
		}
	}

	if earliest.Equal(latest) {
		// Same date, treat as end date
		return "", latestStr, nil
	}

	return earliestStr, latestStr, nil
}

// extractDuration parses term lengths in months.
func extractDuration(rawText string) (int, []string) {
	matches := durationPattern.FindAllStringSubmatch(rawText, -1)
	if len(matches) == 0 {
		return 0, nil
	}

	var totalMonths int
	for _, m := range matches {
		if len(m) >= 3 {
			num, _ := strconv.Atoi(m[1])
			unit := strings.ToLower(m[2])
			switch {
			case strings.HasPrefix(unit, "year") || strings.HasPrefix(unit, "yr"):
				totalMonths += num * 12
			case strings.HasPrefix(unit, "month"):
				totalMonths += num
			}
		}
	}

	if totalMonths == 0 {
		return 0, nil
	}

	return totalMonths, nil
}

// extractTerminationNotice finds the notice period in days.
func extractTerminationNotice(rawText string) (int, []string) {
	for _, m := range terminationPattern.FindAllStringSubmatch(rawText, -1) {
		if len(m) >= 2 {
			n, err := strconv.Atoi(m[1])
			if err == nil && n > 0 {
				return n, nil
			}
		}
	}
	for _, m := range terminationPattern2.FindAllStringSubmatch(rawText, -1) {
		if len(m) >= 2 {
			n, err := strconv.Atoi(m[1])
			if err == nil && n > 0 {
				return n, nil
			}
		}
	}
	return 0, nil
}

// extractPriceLock finds price lock/guarantee periods.
func extractPriceLock(rawText string) (string, []string) {
	m := priceLockPattern.FindStringSubmatch(rawText)
	if len(m) >= 2 {
		num, _ := strconv.Atoi(m[1])
		if num > 0 {
			// Check if the matched group includes "year"
			fullMatch := strings.ToLower(m[0])
			if strings.Contains(fullMatch, "year") {
				return formatLockPeriod(num, "year"), nil
			}
			return formatLockPeriod(num, "month"), nil
		}
	}
	return "", nil
}

// extractPricing finds dollar amounts and unit context.
func extractPricing(rawText string) (float64, string) {
	m := pricePattern.FindStringSubmatch(rawText)
	if len(m) < 2 {
		return 0, ""
	}

	cleaned := strings.ReplaceAll(m[1], ",", "")
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || amount <= 0 {
		return 0, ""
	}

	// Determine unit
	unit := "flat"
	if perSeatPattern.MatchString(rawText) {
		if monthlyTermPattern.MatchString(rawText) {
			unit = "per_seat_month"
		} else if annualTermPattern.MatchString(rawText) {
			unit = "per_seat_year"
		} else {
			unit = "per_seat"
		}
	} else if monthlyTermPattern.MatchString(rawText) {
		unit = "flat_monthly"
	} else if annualTermPattern.MatchString(rawText) {
		unit = "flat_yearly"
	}

	return math.Round(amount*100) / 100, unit
}

// ─── Utility Helpers ───

func monthNumber(name string) int {
	names := map[string]int{
		"january": 1, "february": 2, "march": 3, "april": 4, "may": 5, "june": 6,
		"july": 7, "august": 8, "september": 9, "october": 10, "november": 11, "december": 12,
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "jun": 6, "jul": 7,
		"aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
	return names[strings.ToLower(name)]
}

func formatLockPeriod(num int, unit string) string {
	if num == 1 {
		return fmt.Sprintf("%d %s", num, unit)
	}
	return fmt.Sprintf("%d %ss", num, unit)
}

func resultsFromBool(v bool) string {
	if v {
		return "high"
	}
	return "low"
}

// computeOverallConfidence determines the overall confidence level.
func computeOverallConfidence(fieldConf *PerFieldConfidence) string {
	highCount := 0
	lowCount := 0
	total := 0

	fields := []string{
		fieldConf.EndDate,
		fieldConf.AutoRenew,
		fieldConf.TerminationNotice,
		fieldConf.PriceLockPeriod,
		fieldConf.AnnualContract,
		fieldConf.Pricing,
	}

	for _, f := range fields {
		if f != "" {
			total++
			switch f {
			case "high":
				highCount++
			case "low":
				lowCount++
			}
		}
	}

	if total == 0 {
		return "low"
	}

	ratio := float64(highCount) / float64(total)
	switch {
	case ratio >= 0.6:
		return "high"
	case ratio >= 0.3:
		return "medium"
	default:
		return "low"
	}
}

// parseDateString tries to parse a date string in various formats.
func parseDateString(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"January 2, 2006",
		"January 02, 2006",
		"01/02/2006",
		"1/2/2006",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
