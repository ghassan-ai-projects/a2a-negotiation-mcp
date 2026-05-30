package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	serverAddr := flag.String("server", "localhost:8080", "MCP server address (host:port)")
	rawJSON := flag.Bool("json", false, "Output raw JSON instead of formatted text")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		return
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "query":
		runQuery(*serverAddr, *rawJSON, cmdArgs)
	case "negotiate":
		runNegotiate(*serverAddr, *rawJSON, cmdArgs)
	case "discover":
		runDiscover(*serverAddr, *rawJSON, cmdArgs)
	case "health":
		runHealth(*serverAddr, *rawJSON, cmdArgs)
	case "sla":
		runSLA(*serverAddr, *rawJSON, cmdArgs)
	case "group":
		runGroup(*serverAddr, *rawJSON, cmdArgs)
	case "marketplace":
		runMarketplace(*serverAddr, *rawJSON, cmdArgs)
	case "contracts":
		runContracts(*serverAddr, *rawJSON, cmdArgs)
	case "quote":
		runQuote(*serverAddr, *rawJSON, cmdArgs)
	case "strategies":
		runStrategies(*serverAddr, *rawJSON, cmdArgs)
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'a2a-cli help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: a2a-cli [global flags] <command> [args]

Global flags:
  -server host:port    MCP server address (default: localhost:8080)
  -json                Output raw JSON instead of formatted text

Commands:
  query       Query fair market price for a vendor/product
  negotiate   Start and run a new negotiation session
  discover    Discover negotiation opportunities
  health      Get vendor health and leverage assessment
  sla         Manage SLA contracts and breaches
  group       Manage buying groups
  marketplace Browse the unused-seats marketplace
  contracts   List and manage vendor contracts
  quote       Analyze a vendor quote email
  strategies  List available negotiation strategies
  help        Show detailed help

Run 'a2a-cli help <command>' for command-specific help.
`)
}

func printHelp() {
	args := flag.Args()[1:]
	if len(args) > 0 {
		printCommandHelp(args[0])
		return
	}
	printUsage()
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "query":
		fmt.Print(`Usage: a2a-cli query <vendor> [sku] [--seats N] [--term N]

Query fair market price range for a vendor's product.

Arguments:
  vendor    Vendor name (e.g. Slack, GitHub, Salesforce)
  sku       Product SKU (optional)
  --seats   Number of seats/units
  --term    Contract term in months

Examples:
  a2a-cli query Slack
  a2a-cli query Slack Pro --seats 50 --term 12
`)
	case "negotiate":
		fmt.Print(`Usage: a2a-cli negotiate <vendor> <sku> [--seats N] [--strategy S] [--budget N]

Start and run a new negotiation session.

Arguments:
  vendor      Vendor name (e.g. Slack, GitHub, Salesforce)
  sku         Product SKU
  --seats     Number of seats/units
  --strategy  Strategy: aggressive, balanced, or conservative (default: balanced)
  --budget    Maximum budget per unit

Examples:
  a2a-cli negotiate Slack Pro
  a2a-cli negotiate Slack Pro --seats 50 --strategy balanced --budget 10
`)
	case "discover":
		fmt.Print(`Usage: a2a-cli discover [--industry I]

Discover negotiation opportunities.

Flags:
  --industry  Industry to filter by (e.g. fintech, healthcare)

Examples:
  a2a-cli discover
  a2a-cli discover --industry fintech
`)
	case "health":
		fmt.Print(`Usage: a2a-cli health <vendor>

Get vendor health and leverage assessment.

Arguments:
  vendor  Vendor name (e.g. Salesforce, Slack)

Examples:
  a2a-cli health Salesforce
`)
	case "sla":
		fmt.Print(`Usage: a2a-cli sla <subcommand> [args]

Manage SLA contracts and breaches.

Subcommands:
  add     Add an SLA contract
  breach  Record a service breach
  report  Get SLA report for a month

'a2a-cli help sla add' for more details.
`)
	case "group":
		fmt.Print(`Usage: a2a-cli group <subcommand> [args]

Manage buying groups.

Subcommands:
  create  Create a buying group
  join    Join an existing group

'a2a-cli help group create' for more details.
`)
	case "marketplace":
		fmt.Print(`Usage: a2a-cli marketplace <subcommand> [args]

Browse the unused-seats marketplace.

Subcommands:
  list    Marketplace overview
  search  Search for unused seats

'a2a-cli help marketplace list' for more details.
`)
	case "contracts":
		fmt.Print(`Usage: a2a-cli contracts list [--vendor V] [--status S] [--expiring N]

List and filter vendor contracts.

Flags:
  --vendor    Filter by vendor name
  --status    Filter by status: active, negotiating, renewed, cancelled
  --expiring  Contracts expiring within N days

Examples:
  a2a-cli contracts list
  a2a-cli contracts list --expiring 90
  a2a-cli contracts list --vendor Slack --status active
`)
	case "quote":
		fmt.Print(`Usage: a2a-cli quote <text> [--vendor V] [--sku S]

Analyze a vendor quote email.

Arguments:
  text  The full text of the vendor quote email

Flags:
  --vendor  Vendor name override
  --sku     Product SKU

Examples:
  a2a-cli quote "Annual subscription for Slack Pro: $10/user/month" --vendor Slack --sku Pro
`)
	case "strategies":
		fmt.Print(`Usage: a2a-cli strategies

List available negotiation strategies with descriptions.
`)
	default:
		fmt.Fprintf(os.Stderr, "Unknown help topic: %s\n", cmd)
	}
}

// ─── HTTP helpers ───

func postJSON(server, endpoint string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("http://%s%s", server, endpoint)
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
	}

	if resp.StatusCode >= 400 {
		errMsg, _ := result["error"].(string)
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return result, fmt.Errorf("server error (%d): %s", resp.StatusCode, errMsg)
	}

	return result, nil
}

func outputResult(rawJSON bool, result map[string]any) {
	if rawJSON {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return
	}

	if errStr, ok := result["error"].(string); ok && errStr != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", errStr)
		return
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

// ─── Query ───

func runQuery(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	seats := fs.Int("seats", 0, "Number of seats/units")
	term := fs.Int("term", 0, "Contract term in months")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli query <vendor> [sku] [--seats N] [--term N]")
		os.Exit(1)
	}

	vendor := posArgs[0]
	sku := ""
	if len(posArgs) > 1 {
		sku = posArgs[1]
	}

	params := map[string]any{
		"vendor": vendor,
	}
	if sku != "" {
		params["sku"] = sku
	}
	if *seats > 0 {
		params["quantity"] = *seats
	}
	if *term > 0 {
		params["term_months"] = *term
	}

	req := map[string]any{
		"task":   "query_price",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if rawJSON {
		outputResult(true, result)
		return
	}

	r, _ := result["result"].(map[string]any)
	if r == nil {
		outputResult(false, result)
		return
	}

	v, _ := r["vendor"].(string)
	s, _ := r["sku"].(string)
	lp, _ := r["list_price"].(float64)
	mm, _ := r["market_min"].(float64)
	mx, _ := r["market_max"].(float64)
	sm, _ := r["suggested_min"].(float64)
	sx, _ := r["suggested_max"].(float64)
	cf, _ := r["confidence"].(string)
	td, _ := r["typical_discount_pct"].(float64)

	fmt.Printf("Vendor:         %s\n", v)
	if s != "" {
		fmt.Printf("SKU:            %s\n", s)
	}
	fmt.Printf("List Price:     $%.2f\n", lp)
	fmt.Printf("Market Range:   $%.2f - $%.2f\n", mm, mx)
	fmt.Printf("Suggested Min:  $%.2f\n", sm)
	fmt.Printf("Suggested Max:  $%.2f\n", sx)
	fmt.Printf("Confidence:     %s\n", cf)
	fmt.Printf("Typical Disc.:  %.1f%%\n", td)
}

// ─── Negotiate ───

func runNegotiate(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("negotiate", flag.ExitOnError)
	strategy := fs.String("strategy", "balanced", "Strategy: aggressive, balanced, conservative")
	budget := fs.Float64("budget", 0, "Maximum budget per unit")
	seats := fs.Int("seats", 0, "Number of seats/units")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli negotiate <vendor> <sku> [--strategy S] [--budget N] [--seats N]")
		os.Exit(1)
	}

	vendor := posArgs[0]
	sku := posArgs[1]

	terms := map[string]any{}
	if *seats > 0 {
		terms["seats"] = *seats
	}

	req := map[string]any{
		"vendor":   vendor,
		"sku":      sku,
		"strategy": *strategy,
		"budget":   *budget,
		"terms":    terms,
	}

	result, err := postJSON(server, "/a2a/negotiate", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if rawJSON {
		outputResult(true, result)
		return
	}

	sessionID, _ := result["session_id"].(string)
	status, _ := result["status"].(string)
	offer, _ := result["offer"].(float64)
	listPrice, _ := result["list_price"].(float64)
	strat, _ := result["strategy"].(string)
	mandateID, _ := result["mandate_id"].(string)
	r2, _ := result["result"].(map[string]any)

	outcome, _ := r2["outcome"].(string)
	rounds, _ := r2["rounds_completed"].(float64)

	fmt.Printf("Session:        %s\n", sessionID)
	fmt.Printf("Mandate:        %s\n", mandateID)
	fmt.Printf("Status:         %s\n", status)
	fmt.Printf("Strategy:       %s\n", strat)
	fmt.Printf("List Price:     $%.2f\n", listPrice)
	fmt.Printf("Current Offer:  $%.2f\n", offer)
	if outcome != "" {
		fmt.Printf("Outcome:        %s\n", outcome)
	}
	if rounds > 0 {
		fmt.Printf("Rounds Done:    %.0f\n", rounds)
	}
}

// ─── Discover ───

func runDiscover(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	industry := fs.String("industry", "", "Industry filter (e.g. fintech)")
	fs.Parse(args)

	params := map[string]any{}
	if *industry != "" {
		params["industry"] = *industry
	}

	req := map[string]any{
		"task":   "negotiate_discover_opportunities",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Health ───

func runHealth(server string, rawJSON bool, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli health <vendor>")
		os.Exit(1)
	}
	vendor := args[0]

	req := map[string]any{
		"task":   "negotiate_vendor_health",
		"params": map[string]any{"vendor": vendor},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if rawJSON {
		outputResult(true, result)
		return
	}

	r, _ := result["result"].(map[string]any)
	if r == nil {
		outputResult(false, result)
		return
	}

	v, _ := r["vendor"].(string)
	lvg, _ := r["leverage"].(string)
	sug, _ := r["suggestion"].(string)
	h, _ := r["health"].(map[string]any)

	fmt.Printf("Vendor:     %s\n", v)
	if h != nil {
		score, _ := h["score"].(float64)
		cat, _ := h["category"].(string)
		fmt.Printf("Score:      %.0f/100\n", score)
		fmt.Printf("Category:   %s\n", cat)
	}
	fmt.Printf("Leverage:   %s\n", lvg)
	fmt.Printf("Suggestion: %s\n", sug)
}

// ─── SLA ───

func runSLA(server string, rawJSON bool, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli sla <add|breach|report> [args]")
		os.Exit(1)
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "add":
		runSLAAdd(server, rawJSON, subArgs)
	case "breach":
		runSLABreach(server, rawJSON, subArgs)
	case "report":
		runSLAReport(server, rawJSON, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown sla subcommand: %s (use add, breach, report)\n", sub)
		os.Exit(1)
	}
}

func runSLAAdd(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("sla add", flag.ExitOnError)
	uptime := fs.Float64("uptime", 0, "Guaranteed uptime %% (e.g. 99.9)")
	credit := fs.Float64("credit", 0, "Service credit %% (e.g. 10)")
	maxCredit := fs.Float64("max-credit", 0, "Max credit %% cap (e.g. 25)")
	spend := fs.Float64("spend", 0, "Monthly spend amount")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli sla add <vendor> <service> --uptime N --credit N --max-credit N --spend N")
		os.Exit(1)
	}

	req := map[string]any{
		"task": "negotiate_add_sla",
		"params": map[string]any{
			"vendor":         posArgs[0],
			"service":        posArgs[1],
			"uptime_pct":     *uptime,
			"credit_pct":     *credit,
			"max_credit_pct": *maxCredit,
			"monthly_spend":  *spend,
		},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

func runSLABreach(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("sla breach", flag.ExitOnError)
	date := fs.String("date", time.Now().UTC().Format(time.RFC3339), "Breach date (RFC3339)")
	duration := fs.Int("duration", 0, "Downtime duration in minutes")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli sla breach <vendor> <service> --duration N [--date D]")
		os.Exit(1)
	}

	req := map[string]any{
		"task": "negotiate_record_breach",
		"params": map[string]any{
			"vendor":        posArgs[0],
			"service":       posArgs[1],
			"date":          *date,
			"duration_mins": *duration,
		},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

func runSLAReport(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("sla report", flag.ExitOnError)
	month := fs.String("month", time.Now().UTC().Format(time.RFC3339), "Month (RFC3339)")
	fs.Parse(args)

	req := map[string]any{
		"task": "negotiate_sla_report",
		"params": map[string]any{
			"month": *month,
		},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Group ───

func runGroup(server string, rawJSON bool, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli group <create|join> [args]")
		os.Exit(1)
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "create":
		runGroupCreate(server, rawJSON, subArgs)
	case "join":
		runGroupJoin(server, rawJSON, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown group subcommand: %s (use create, join)\n", sub)
		os.Exit(1)
	}
}

func runGroupCreate(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("group create", flag.ExitOnError)
	sku := fs.String("sku", "", "Product SKU")
	minMembers := fs.Int("min-members", 2, "Minimum members required")
	expires := fs.Int("expires", 72, "Expiration in hours")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli group create <vendor> [--sku S] [--min-members N]")
		os.Exit(1)
	}

	params := map[string]any{
		"target_vendor":    posArgs[0],
		"target_sku":       *sku,
		"min_members":      *minMembers,
		"expires_in_hours": *expires,
	}

	req := map[string]any{
		"task":   "negotiate_create_group",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

func runGroupJoin(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("group join", flag.ExitOnError)
	userID := fs.String("user", "cli-user", "User identifier")
	qty := fs.Int("qty", 0, "Number of seats/units")
	maxPrice := fs.Float64("max-price", 0, "Maximum price per unit")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli group join <group-id> [--user U] [--qty N]")
		os.Exit(1)
	}

	params := map[string]any{
		"group_id": posArgs[0],
		"user_id":  *userID,
	}
	if *qty > 0 {
		params["quantity"] = *qty
	}
	if *maxPrice > 0 {
		params["max_price"] = *maxPrice
	}

	req := map[string]any{
		"task":   "negotiate_join_group",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Marketplace ───

func runMarketplace(server string, rawJSON bool, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli marketplace <list|search> [args]")
		os.Exit(1)
	}
	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "list":
		runMarketplaceList(server, rawJSON, subArgs)
	case "search":
		runMarketplaceSearch(server, rawJSON, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown marketplace subcommand: %s (use list, search)\n", sub)
		os.Exit(1)
	}
}

func runMarketplaceList(server string, rawJSON bool, args []string) {
	req := map[string]any{
		"task":   "negotiate_marketplace_overview",
		"params": map[string]any{},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if rawJSON {
		outputResult(true, result)
		return
	}

	r, _ := result["result"].(map[string]any)
	if r == nil {
		outputResult(false, result)
		return
	}

	vendors, _ := r["vendors"].([]any)
	count, _ := r["vendor_count"].(float64)

	fmt.Printf("Vendors with listings: %.0f\n", count)
	for _, v := range vendors {
		if m, ok := v.(map[string]any); ok {
			name, _ := m["vendor"].(string)
			listings, _ := m["listing_count"].(float64)
			fmt.Printf("  %-20s %3.0f listings\n", name, listings)
		}
	}
}

func runMarketplaceSearch(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("marketplace search", flag.ExitOnError)
	sku := fs.String("sku", "", "SKU filter")
	maxSeats := fs.Int("max-seats", 0, "Max seats filter")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli marketplace search <vendor> [--sku S] [--max-seats N]")
		os.Exit(1)
	}

	params := map[string]any{
		"vendor": posArgs[0],
	}
	if *sku != "" {
		params["sku"] = *sku
	}
	if *maxSeats > 0 {
		params["max_seats"] = *maxSeats
	}

	req := map[string]any{
		"task":   "negotiate_search_used",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Contracts ───

func runContracts(server string, rawJSON bool, args []string) {
	if len(args) < 1 || args[0] != "list" {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli contracts list [--vendor V] [--status S] [--expiring N]")
		os.Exit(1)
	}
	subArgs := args[1:]

	fs := flag.NewFlagSet("contracts list", flag.ExitOnError)
	vendor := fs.String("vendor", "", "Filter by vendor name")
	status := fs.String("status", "", "Filter by status (active, negotiating, renewed, cancelled)")
	expiring := fs.Int("expiring", 0, "Expiring within N days")
	fs.Parse(subArgs)

	params := map[string]any{}
	if *vendor != "" {
		params["vendor"] = *vendor
	}
	if *status != "" {
		params["status"] = *status
	}
	if *expiring > 0 {
		params["expiring_soon"] = *expiring
	}

	req := map[string]any{
		"task":   "negotiate_list_contracts",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Quote ───

func runQuote(server string, rawJSON bool, args []string) {
	fs := flag.NewFlagSet("quote", flag.ExitOnError)
	vendor := fs.String("vendor", "", "Vendor name override")
	sku := fs.String("sku", "", "Product SKU")
	fs.Parse(args)

	posArgs := fs.Args()
	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: a2a-cli quote <text> [--vendor V] [--sku S]")
		os.Exit(1)
	}

	text := strings.Join(posArgs, " ")

	params := map[string]any{
		"raw_text": text,
	}
	if *vendor != "" {
		params["vendor"] = *vendor
	}
	if *sku != "" {
		params["sku"] = *sku
	}

	req := map[string]any{
		"task":   "negotiate_analyze_quote",
		"params": params,
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}

// ─── Strategies ───

func runStrategies(server string, rawJSON bool, args []string) {
	req := map[string]any{
		"task":   "negotiate_strategies",
		"params": map[string]any{},
	}

	result, err := postJSON(server, "/a2a/task", req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	outputResult(rawJSON, result)
}
