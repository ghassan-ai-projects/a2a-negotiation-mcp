package sla

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupTestStore creates an in-memory SQLite store for SLA testing.
func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", "file:sla_test_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

// setupTestEngine creates an in-memory Store and Engine for SLA testing.
func setupTestEngine(t *testing.T) (*Engine, *Store) {
	t.Helper()

	store := setupTestStore(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := NewEngine(store, logger)
	return eng, store
}

// seedTestContract adds a known SLA contract for testing.
func seedTestContract(t *testing.T, store *Store) *SLAContract {
	t.Helper()
	ctx := context.Background()

	c := &SLAContract{
		Vendor:       "TestVendor",
		Service:      "TestService",
		UptimePct:    99.9,
		CreditPct:    10,
		MaxCreditPct: 25,
		MonthlySpend: 10000,
		Status:       "active",
	}
	if err := store.AddContract(ctx, c); err != nil {
		t.Fatalf("AddContract: %v", err)
	}
	return c
}

// --- Store Tests ---

func TestAddAndGetContract(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	c := seedTestContract(t, store)

	got, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Vendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", got.Vendor)
	}
	if got.Service != "TestService" {
		t.Errorf("expected service TestService, got %s", got.Service)
	}
	if got.UptimePct != 99.9 {
		t.Errorf("expected uptime_pct 99.9, got %f", got.UptimePct)
	}
	if got.CreditPct != 10 {
		t.Errorf("expected credit_pct 10, got %f", got.CreditPct)
	}
	if got.MaxCreditPct != 25 {
		t.Errorf("expected max_credit_pct 25, got %f", got.MaxCreditPct)
	}
	if got.MonthlySpend != 10000 {
		t.Errorf("expected monthly_spend 10000, got %f", got.MonthlySpend)
	}
	if got.Status != "active" {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestGetContract_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.GetContract(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent contract")
	}
}

func TestListContracts(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	c1 := seedTestContract(t, store)

	c2 := &SLAContract{
		Vendor:       "Vendor2",
		Service:      "Service2",
		UptimePct:    99.0,
		CreditPct:    5,
		MaxCreditPct: 15,
		MonthlySpend: 5000,
		Status:       "paused",
	}
	if err := store.AddContract(ctx, c2); err != nil {
		t.Fatalf("AddContract c2: %v", err)
	}

	// List all contracts
	contracts, err := store.ListContracts(ctx, "")
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(contracts))
	}

	ids := map[string]bool{c1.ID: true, c2.ID: true}
	for _, c := range contracts {
		if !ids[c.ID] {
			t.Errorf("unexpected contract id: %s", c.ID)
		}
		delete(ids, c.ID)
	}
}

func TestListContracts_FilteredByStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = seedTestContract(t, store)

	// Only active
	contracts, err := store.ListContracts(ctx, "active")
	if err != nil {
		t.Fatalf("ListContracts: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("expected 1 active contract, got %d", len(contracts))
	}
	if contracts[0].Status != "active" {
		t.Errorf("expected status active, got %s", contracts[0].Status)
	}
}

func TestAddAndGetBreach(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	b := &SLABreach{
		Vendor:       "TestVendor",
		Service:      "TestService",
		Date:         now,
		DurationMins: 60,
		CreditDue:    100.00,
	}
	if err := store.AddBreach(ctx, b); err != nil {
		t.Fatalf("AddBreach: %v", err)
	}

	got, err := store.GetBreach(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetBreach: %v", err)
	}
	if got.Vendor != "TestVendor" {
		t.Errorf("expected vendor TestVendor, got %s", got.Vendor)
	}
	if got.DurationMins != 60 {
		t.Errorf("expected duration_mins 60, got %d", got.DurationMins)
	}
	if got.CreditDue != 100.00 {
		t.Errorf("expected credit_due 100, got %f", got.CreditDue)
	}
	if got.Filed {
		t.Errorf("expected filed false, got true")
	}
}

func TestFileBreach(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	b := &SLABreach{
		Vendor:       "TestVendor",
		Service:      "TestService",
		Date:         now,
		DurationMins: 60,
		CreditDue:    100.00,
	}
	if err := store.AddBreach(ctx, b); err != nil {
		t.Fatalf("AddBreach: %v", err)
	}

	if err := store.FileBreach(ctx, b.ID, 100.00); err != nil {
		t.Fatalf("FileBreach: %v", err)
	}

	got, err := store.GetBreach(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetBreach after file: %v", err)
	}
	if !got.Filed {
		t.Errorf("expected filed true")
	}
	if got.Payout != 100.00 {
		t.Errorf("expected payout 100, got %f", got.Payout)
	}
	if got.FiledAt.IsZero() {
		t.Error("expected non-zero filed_at")
	}
}

func TestFileBreach_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.FileBreach(ctx, "nonexistent-id", 100.00)
	if err == nil {
		t.Fatal("expected error for nonexistent breach")
	}
}

// --- Engine Tests ---

func TestEngineAddContract(t *testing.T) {
	eng, store := setupTestEngine(t)
	ctx := context.Background()

	c, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}
	if c.ID == "" {
		t.Error("expected non-empty ID")
	}
	if c.Status != "active" {
		t.Errorf("expected status active, got %s", c.Status)
	}

	got, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Vendor != "TestVendor" {
		t.Errorf("expected TestVendor, got %s", got.Vendor)
	}
}

func TestRecordBreach_CreditCalculation(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	// Credit = (10000 * 10/100) * (60 / 43200) = 1000 * 0.0013888 = 1.3888... -> $1.39
	breach, err := eng.RecordBreach(ctx, "TestVendor", "TestService", time.Now().UTC(), 60)
	if err != nil {
		t.Fatalf("RecordBreach: %v", err)
	}

	expectedCredit := 1.39
	if math.Abs(breach.CreditDue-expectedCredit) > 0.01 {
		t.Errorf("expected credit_due %.2f, got %.2f", expectedCredit, breach.CreditDue)
	}
	if breach.DurationMins != 60 {
		t.Errorf("expected duration_mins 60, got %d", breach.DurationMins)
	}
}

func TestRecordBreach_CapsAtMaxCreditPct(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// Monthly spend 1000, credit_pct 10, max_credit_pct 5
	// base_credit = 1000 * (10/100) = 100
	// max ratio = 5/100 = 0.05
	// Full month breach: ratio = 43200/43200 = 1.0, capped at 0.05
	// credit = 100 * 0.05 = 5.00
	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 5, 1000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	breach, err := eng.RecordBreach(ctx, "TestVendor", "TestService", time.Now().UTC(), totalMonthlyMins)
	if err != nil {
		t.Fatalf("RecordBreach: %v", err)
	}

	expectedCredit := 5.00
	if math.Abs(breach.CreditDue-expectedCredit) > 0.01 {
		t.Errorf("expected credit_due %.2f (capped at max_credit_pct), got %.2f", expectedCredit, breach.CreditDue)
	}
}

func TestFileClaim_MarksBreachAsFiled(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	breach, err := eng.RecordBreach(ctx, "TestVendor", "TestService", time.Now().UTC(), 60)
	if err != nil {
		t.Fatalf("RecordBreach: %v", err)
	}

	filed, err := eng.FileClaim(ctx, breach.ID)
	if err != nil {
		t.Fatalf("FileClaim: %v", err)
	}

	if !filed.Filed {
		t.Error("expected breach to be marked as filed")
	}
	if filed.FiledAt.IsZero() {
		t.Error("expected non-zero filed_at")
	}
	if filed.Payout != breach.CreditDue {
		t.Errorf("expected payout %.2f, got %.2f", breach.CreditDue, filed.Payout)
	}
}

func TestFileClaim_AlreadyFiled(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	breach, err := eng.RecordBreach(ctx, "TestVendor", "TestService", time.Now().UTC(), 60)
	if err != nil {
		t.Fatalf("RecordBreach: %v", err)
	}

	_, err = eng.FileClaim(ctx, breach.ID)
	if err != nil {
		t.Fatalf("first FileClaim: %v", err)
	}

	_, err = eng.FileClaim(ctx, breach.ID)
	if err == nil {
		t.Fatal("expected error for already-filed breach")
	}
}

func TestGetReport_AggregatesCorrectly(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	now := time.Now().UTC()

	breach1, err := eng.RecordBreach(ctx, "TestVendor", "TestService", now, 60)
	if err != nil {
		t.Fatalf("RecordBreach 1: %v", err)
	}

	breach2, err := eng.RecordBreach(ctx, "TestVendor", "TestService", now, 120)
	if err != nil {
		t.Fatalf("RecordBreach 2: %v", err)
	}

	// File only the first breach
	_, err = eng.FileClaim(ctx, breach1.ID)
	if err != nil {
		t.Fatalf("FileClaim: %v", err)
	}

	// Get report for current month
	report, err := eng.GetReport(ctx, now)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if report.Contract.Vendor != "TestVendor" {
		t.Errorf("expected contract vendor TestVendor, got %s", report.Contract.Vendor)
	}
	if len(report.Breaches) != 2 {
		t.Fatalf("expected 2 breaches, got %d", len(report.Breaches))
	}
	if report.FiledCount != 1 {
		t.Errorf("expected 1 filed breach, got %d", report.FiledCount)
	}
	expectedTotal := math.Round((breach1.CreditDue+breach2.CreditDue)*100) / 100
	if math.Abs(report.TotalCredits-expectedTotal) > 0.01 {
		t.Errorf("expected total_credits %.2f, got %.2f", expectedTotal, report.TotalCredits)
	}
}

func TestRecordBreach_ZeroDuration(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.AddContract(ctx, "TestVendor", "TestService", 99.9, 10, 25, 10000)
	if err != nil {
		t.Fatalf("AddContract: %v", err)
	}

	// Zero duration should result in zero credit
	breach, err := eng.RecordBreach(ctx, "TestVendor", "TestService", time.Now().UTC(), 0)
	if err != nil {
		t.Fatalf("RecordBreach: %v", err)
	}

	if breach.CreditDue != 0 {
		t.Errorf("expected credit_due 0 for zero duration, got %.2f", breach.CreditDue)
	}
}

func TestRecordBreach_NoActiveContract(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	_, err := eng.RecordBreach(ctx, "NonExistent", "Service", time.Now().UTC(), 60)
	if err == nil {
		t.Fatal("expected error for non-existent contract")
	}
}

func TestUpdateContractStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	c := seedTestContract(t, store)

	if err := store.UpdateContractStatus(ctx, c.ID, "paused"); err != nil {
		t.Fatalf("UpdateContractStatus: %v", err)
	}

	got, err := store.GetContract(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetContract: %v", err)
	}
	if got.Status != "paused" {
		t.Errorf("expected status paused, got %s", got.Status)
	}
}

func TestEngineAddAndGetReport(t *testing.T) {
	eng, _ := setupTestEngine(t)
	ctx := context.Background()

	// No contracts yet — should still return a result but with no contract
	now := time.Now().UTC()
	report, err := eng.GetReport(ctx, now)
	if err != nil {
		t.Fatalf("GetReport with no contracts: %v", err)
	}
	if report.Contract.ID != "" {
		t.Errorf("expected empty contract ID, got %s", report.Contract.ID)
	}
	if len(report.Breaches) != 0 {
		t.Errorf("expected 0 breaches, got %d", len(report.Breaches))
	}
}
