package health

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// VendorReputation tracks a vendor's negotiation behavior over time.
type VendorReputation struct {
	Vendor         string  `json:"vendor"`
	DealCount      int     `json:"deal_count"`
	AvgDiscountPct float64 `json:"avg_discount_pct"`
	MaxDiscountPct float64 `json:"max_discount_pct"`
	Negotiability  string  `json:"negotiability"` // very_flexible, flexible, neutral, rigid, very_rigid
	WinRate        float64 `json:"win_rate"`
}

// reputationRow mirrors the DB columns for internal use.
type reputationRow struct {
	Vendor           string
	DealCount        int
	TotalDiscountPct float64
	MaxDiscountPct   float64
	SuccessCount     int
	UpdatedAt        string
}

// GetReputation returns the reputation for a vendor, or a zero-value record if unknown.
func (e *Engine) GetReputation(ctx context.Context, vendor string) (*VendorReputation, error) {
	row, err := e.store.getReputationRow(ctx, vendor)
	if err != nil {
		return nil, fmt.Errorf("get reputation: %w", err)
	}
	if row == nil {
		return &VendorReputation{Vendor: vendor}, nil
	}

	vr := row.toReputation()
	return &vr, nil
}

// UpdateReputation records a negotiation outcome and updates the vendor's reputation.
func (e *Engine) UpdateReputation(ctx context.Context, vendor string, discountPct float64, succeeded bool) error {
	row, err := e.store.getReputationRow(ctx, vendor)
	if err != nil {
		return fmt.Errorf("update reputation: %w", err)
	}
	if row == nil {
		row = &reputationRow{Vendor: vendor}
	}

	row.DealCount++
	row.TotalDiscountPct += discountPct
	if discountPct > row.MaxDiscountPct {
		row.MaxDiscountPct = discountPct
	}
	if succeeded {
		row.SuccessCount++
	}
	row.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := e.store.upsertReputation(ctx, row); err != nil {
		return fmt.Errorf("upsert reputation: %w", err)
	}
	return nil
}

// RankFlexibility returns vendors sorted by avg discount descending (most flexible first).
func (e *Engine) RankFlexibility(ctx context.Context, limit int) ([]VendorReputation, error) {
	rows, err := e.store.listReputations(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("rank flexibility: %w", err)
	}

	reputations := make([]VendorReputation, 0, len(rows))
	for _, row := range rows {
		reputations = append(reputations, row.toReputation())
	}

	// Sort by avg discount descending (most flexible first).
	sort.Slice(reputations, func(i, j int) bool {
		return reputations[i].AvgDiscountPct > reputations[j].AvgDiscountPct
	})

	if limit > 0 && len(reputations) > limit {
		reputations = reputations[:limit]
	}

	return reputations, nil
}

// toReputation computes derived fields from the raw DB row.
func (r *reputationRow) toReputation() VendorReputation {
	vr := VendorReputation{
		Vendor:         r.Vendor,
		DealCount:      r.DealCount,
		MaxDiscountPct: r.MaxDiscountPct,
	}
	if r.DealCount > 0 {
		vr.AvgDiscountPct = r.TotalDiscountPct / float64(r.DealCount)
		vr.WinRate = float64(r.SuccessCount) / float64(r.DealCount)
	}
	vr.Negotiability = negotiabilityLabel(vr.AvgDiscountPct)
	return vr
}

// negotiabilityLabel maps avg discount percentage to a negotiability label.
//
//	0-5%   → very_rigid
//	5-15%  → rigid
//	15-25% → neutral
//	25-40% → flexible
//	40%+   → very_flexible
func negotiabilityLabel(avgDiscountPct float64) string {
	switch {
	case avgDiscountPct < 5:
		return "very_rigid"
	case avgDiscountPct < 15:
		return "rigid"
	case avgDiscountPct < 25:
		return "neutral"
	case avgDiscountPct < 40:
		return "flexible"
	default:
		return "very_flexible"
	}
}
