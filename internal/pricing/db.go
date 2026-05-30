package pricing

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/ierrors"
	_ "modernc.org/sqlite"
)

// Store provides pricing data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store and ensures the schema exists.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// NewInMemoryStore creates a Store using an in-memory SQLite database (for tests).
func NewInMemoryStore() (*Store, error) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open in-memory db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping in-memory db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate in-memory: %w", err)
	}
	return s, nil
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vendors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		category TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS pricing_snapshot (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_id INTEGER NOT NULL REFERENCES vendors(id),
		sku TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		list_price REAL NOT NULL DEFAULT 0,
		min_observed REAL NOT NULL DEFAULT 0,
		max_observed REAL NOT NULL DEFAULT 0,
		typical_pct REAL NOT NULL DEFAULT 0,
		unit TEXT NOT NULL DEFAULT 'per_seat_month',
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(vendor_id, sku)
	);

	CREATE INDEX IF NOT EXISTS idx_pricing_vendor ON pricing_snapshot(vendor_id);
	CREATE INDEX IF NOT EXISTS idx_pricing_sku ON pricing_snapshot(sku);

	CREATE TABLE IF NOT EXISTS mandates (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		principal TEXT NOT NULL,
		agent_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		expires_at TEXT NOT NULL,
		terms TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_mandates_status ON mandates(status);
	CREATE INDEX IF NOT EXISTS idx_mandates_principal ON mandates(principal);
	CREATE INDEX IF NOT EXISTS idx_mandates_type ON mandates(type);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SeedFromCSV loads pricing data from a CSV file.
// CSV columns: vendor,category,sku,description,list_price,min_observed,max_observed,typical_discount_pct,unit
func (s *Store) SeedFromCSV(ctx context.Context, csvPath string) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	vendorCache := make(map[string]int64)

	for _, row := range records[1:] {
		if len(row) < 9 {
			continue
		}
		vendorName := strings.TrimSpace(row[0])
		category := strings.TrimSpace(row[1])
		sku := strings.TrimSpace(row[2])
		description := strings.TrimSpace(row[3])
		listPrice, _ := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		minObserved, _ := strconv.ParseFloat(strings.TrimSpace(row[5]), 64)
		maxObserved, _ := strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
		typicalPct, _ := strconv.ParseFloat(strings.TrimSpace(row[7]), 64)
		unit := strings.TrimSpace(row[8])

		vendorID, ok := vendorCache[vendorName]
		if !ok {
			res, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO vendors (name, category) VALUES (?, ?)", vendorName, category)
			if err != nil {
				return fmt.Errorf("insert vendor %s: %w", vendorName, err)
			}
			id, _ := res.LastInsertId()
			if id == 0 {
				err = tx.QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", vendorName).Scan(&id)
				if err != nil {
					return fmt.Errorf("get vendor id %s: %w", vendorName, err)
				}
			}
			vendorCache[vendorName] = id
			vendorID = id
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO pricing_snapshot (vendor_id, sku, description, list_price, min_observed, max_observed, typical_pct, unit)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(vendor_id, sku) DO UPDATE SET
				list_price=excluded.list_price,
				min_observed=excluded.min_observed,
				max_observed=excluded.max_observed,
				typical_pct=excluded.typical_pct,
				description=excluded.description,
				updated_at=datetime('now')
		`, vendorID, sku, description, listPrice, minObserved, maxObserved, typicalPct, unit)
		if err != nil {
			return fmt.Errorf("insert pricing %s/%s: %w", vendorName, sku, err)
		}
	}

	return tx.Commit()
}

// GetVendorID returns the vendor ID for a given vendor name.
func (s *Store) GetVendorID(ctx context.Context, vendor string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM vendors WHERE name = ?", vendor).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ierrors.New(ierrors.ErrVendorNotFound, "vendor not found", map[string]any{"vendor": vendor})
	}
	if err != nil {
		return 0, fmt.Errorf("query vendor: %w", err)
	}
	return id, nil
}

// GetPricingByVendorSKU returns pricing for a specific vendor and optional SKU.
func (s *Store) GetPricingByVendorSKU(ctx context.Context, vendor string, sku string) (*PriceQueryResult, error) {
	vendorID, err := s.GetVendorID(ctx, vendor)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT p.sku, p.description, p.list_price, p.min_observed, p.max_observed,
			   p.typical_pct, p.unit, v.name
		FROM pricing_snapshot p
		JOIN vendors v ON v.id = p.vendor_id
		WHERE p.vendor_id = ?
	`
	args := []any{vendorID}
	if sku != "" {
		query += " AND p.sku = ?"
		args = append(args, sku)
	}
	query += " ORDER BY p.sku LIMIT 1"

	var result PriceQueryResult
	var unit string
	err = s.db.QueryRowContext(ctx, query, args...).Scan(
		&result.SKU, &result.Description, &result.ListPrice,
		&result.MarketMin, &result.MarketMax, &result.TypicalPct,
		&unit, &result.Vendor,
	)
	if err == sql.ErrNoRows {
		return nil, ierrors.New(ierrors.ErrPricingNotFound, "no pricing data for vendor/sku",
			map[string]any{"vendor": vendor, "sku": sku})
	}
	if err != nil {
		return nil, fmt.Errorf("query pricing: %w", err)
	}

	var dataPoints int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deal_outcomes WHERE vendor = ?`, vendor).Scan(&dataPoints)
	result.DataPoints = dataPoints

	discount := result.TypicalPct / 100.0
	suggestedMin := result.ListPrice * (1 - discount - 0.05)
	suggestedMax := result.ListPrice * (1 - discount + 0.03)
	if suggestedMin < result.MarketMin {
		suggestedMin = result.MarketMin
	}
	if suggestedMax > result.MarketMax {
		suggestedMax = result.MarketMax
	}
	result.SuggestedMin = suggestedMin
	result.SuggestedMax = suggestedMax

	switch {
	case dataPoints >= 20:
		result.Confidence = "high"
	case dataPoints >= 5:
		result.Confidence = "medium"
	default:
		result.Confidence = "low"
	}

	return &result, nil
}

// GetMarketRange returns the overall market price range for a vendor.
func (s *Store) GetMarketRange(ctx context.Context, vendor string) (*MarketRange, error) {
	vendorID, err := s.GetVendorID(ctx, vendor)
	if err != nil {
		return nil, err
	}

	var mr MarketRange
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MIN(min_observed), 0), COALESCE(MAX(max_observed), 0),
			   COALESCE(AVG((min_observed + max_observed) / 2.0), 0), COUNT(*)
		FROM pricing_snapshot WHERE vendor_id = ?
	`, vendorID).Scan(&mr.Min, &mr.Max, &mr.Average, &mr.Count)
	if err != nil {
		return nil, fmt.Errorf("query market range: %w", err)
	}
	return &mr, nil
}

// GetMarketAverageForVendor computes the average market price for a vendor's SKUs.
func (s *Store) GetMarketAverageForVendor(ctx context.Context, vendor string) (float64, error) {
	vendorID, err := s.GetVendorID(ctx, vendor)
	if err != nil {
		return 0, err
	}

	var avg sql.NullFloat64
	err = s.db.QueryRowContext(ctx, `
		SELECT AVG((min_observed + max_observed) / 2.0)
		FROM pricing_snapshot WHERE vendor_id = ?
	`, vendorID).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("query avg price: %w", err)
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// ListVendorsWithPricing returns distinct vendor names that have pricing data.
func (s *Store) ListVendorsWithPricing(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT v.name FROM vendors v
		JOIN pricing_snapshot p ON p.vendor_id = v.id
		ORDER BY v.name
	`)
	if err != nil {
		return nil, fmt.Errorf("query vendors with pricing: %w", err)
	}
	defer rows.Close()

	var vendors []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan vendor: %w", err)
		}
		vendors = append(vendors, name)
	}
	return vendors, rows.Err()
}

// NewStoreFromDB creates a Store using an existing *sql.DB (for tests sharing a DB handle).

// ListPricingByCategory returns all pricing data points for vendors in a given category.
func (s *Store) ListPricingByCategory(ctx context.Context, category string) ([]PriceQueryResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.sku, p.description, p.list_price, p.min_observed, p.max_observed,
			   p.typical_pct, p.unit, v.name
		FROM pricing_snapshot p
		JOIN vendors v ON v.id = p.vendor_id
		WHERE v.category = ?
		ORDER BY v.name, p.sku
	`, category)
	if err != nil {
		return nil, fmt.Errorf("query pricing by category: %w", err)
	}
	defer rows.Close()

	var results []PriceQueryResult
	for rows.Next() {
		var r PriceQueryResult
		var unit string
		if err := rows.Scan(&r.SKU, &r.Description, &r.ListPrice,
			&r.MarketMin, &r.MarketMax, &r.TypicalPct,
			&unit, &r.Vendor); err != nil {
			return nil, fmt.Errorf("scan pricing row: %w", err)
		}
		r.DataPoints = 0
		r.Confidence = "low"
		r.SuggestedMin = r.MarketMin
		r.SuggestedMax = r.MarketMax
		results = append(results, r)
	}
	return results, rows.Err()
}

func NewStoreFromDB(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate from db: %w", err)
	}
	return s, nil
}
