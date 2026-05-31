package coverage

import (
	"bufio"
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Engine parses go test -cover output to produce coverage reports.
type Engine struct{}

// NewEngine creates a new coverage Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// coverageLineRx matches lines like:
// ok  	pkg/path	0.5s	coverage: 85.2% of statements
var coverageLineRx = regexp.MustCompile(
	`^ok\s+(\S+)\s+\S+\s+coverage:\s+([0-9.]+)%`,
)

// Run executes go test -cover ./... and returns a CoverageReport.
func (e *Engine) Run() (*CoverageReport, error) {
	cmd := exec.Command("go", "test", "-cover", "./...")
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return e.ParseOutput(stdout)
}

// ParseOutput parses raw go test -cover output into a CoverageReport.
func (e *Engine) ParseOutput(data []byte) (*CoverageReport, error) {
	var pkgs []PackageCoverage
	var untested []string
	var totalPct float64
	var pkgCount int

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		matches := coverageLineRx.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		pkgName := matches[1]
		pct, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			continue
		}

		pc := PackageCoverage{
			Name:        pkgName,
			CoveragePct: pct,
		}
		pkgs = append(pkgs, pc)
		totalPct += pct
		pkgCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var overallPct float64
	if pkgCount > 0 {
		overallPct = totalPct / float64(pkgCount)
	}

	// Identify untested packages: scan again for lines with "?\s+\S+\s+\[no test files\]"
	// We already have the covered packages; for simplicity we use a second pass
	// looking for the "no test files" pattern.
	untested = e.findUntestedPackages(data)

	recommendation := e.generateRecommendation(overallPct, untested)

	return &CoverageReport{
		OverallPct:       overallPct,
		Packages:         pkgs,
		TotalTests:       len(pkgs),
		UntestedPackages: untested,
		Recommendation:   recommendation,
	}, nil
}

var noTestRx = regexp.MustCompile(`^\?\s+(\S+)\s+\[no test files\]`)

func (e *Engine) findUntestedPackages(data []byte) []string {
	var untested []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		matches := noTestRx.FindStringSubmatch(line)
		if matches != nil {
			untested = append(untested, matches[1])
		}
	}
	return untested
}

func (e *Engine) generateRecommendation(overallPct float64, untested []string) string {
	var b strings.Builder
	if overallPct < 50 {
		b.WriteString("Critical: overall coverage is below 50%. ")
	} else if overallPct < 80 {
		b.WriteString("Moderate: overall coverage is between 50% and 80%. ")
	} else {
		b.WriteString("Good: overall coverage is above 80%. ")
	}

	if len(untested) > 0 {
		b.WriteString("Untested packages found: ")
		b.WriteString(strings.Join(untested, ", "))
		b.WriteString(". Consider adding tests for these packages to improve coverage.")
	} else {
		b.WriteString("All packages have test files.")
	}
	return b.String()
}
