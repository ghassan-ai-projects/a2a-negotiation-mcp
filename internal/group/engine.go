package group

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// Engine provides collective buying logic.
type Engine struct {
	groupStore   *Store
	pricingStore *pricing.Store
	logger       *slog.Logger
}

// NewEngine creates a new group engine.
func NewEngine(groupStore *Store, pricingStore *pricing.Store, logger *slog.Logger) *Engine {
	return &Engine{
		groupStore:   groupStore,
		pricingStore: pricingStore,
		logger:       logger,
	}
}

// CreateGroup creates a new buying group.
func (e *Engine) CreateGroup(ctx context.Context, vendor, sku string, minMembers int, expiresInHours int) (*BuyingGroup, error) {
	now := time.Now().UTC()
	g := &BuyingGroup{
		TargetVendor: vendor,
		TargetSKU:    sku,
		MinMembers:   minMembers,
		Status:       "forming",
		CreatedAt:    now,
		ExpiresAt:    now.Add(time.Duration(expiresInHours) * time.Hour),
	}
	if err := e.groupStore.CreateGroup(ctx, g); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	e.logger.Info("buying group created", "group_id", g.ID, "vendor", vendor, "sku", sku)
	return g, nil
}

// JoinGroup adds a member to the group. Returns error if group is not "forming".
func (e *Engine) JoinGroup(ctx context.Context, groupID, userID string, quantity int, maxPrice float64) (*GroupMember, error) {
	m := &GroupMember{
		GroupID:  groupID,
		UserID:   userID,
		Quantity: quantity,
		MaxPrice: maxPrice,
	}
	if err := e.groupStore.JoinGroup(ctx, m); err != nil {
		return nil, err
	}
	e.logger.Info("member joined buying group", "group_id", groupID, "user_id", userID)
	return m, nil
}

// ComputeOffer generates a consolidated offer based on total demand and member count.
//
// Quantity multiplier tiers:
//   - 1-9:     1.0x
//   - 10-49:   1.2x
//   - 50-199:  1.4x
//   - 200+:    1.6x
//
// Member multiplier tiers:
//   - 2-4:     1.1x
//   - 5-9:     1.25x
//   - 10+:     1.4x
//
// Final discount = typical_discount * qty_mult * member_mult, capped at 50%.
func (e *Engine) ComputeOffer(ctx context.Context, groupID string) (*ConsolidatedOffer, error) {
	group, err := e.groupStore.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	members, err := e.groupStore.GetMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("group %s has no members", groupID)
	}

	// Look up pricing data for the vendor/SKU
	pricingResult, err := e.pricingStore.GetPricingByVendorSKU(ctx, group.TargetVendor, group.TargetSKU)
	if err != nil {
		return nil, fmt.Errorf("pricing lookup: %w", err)
	}

	totalDemand := 0
	for _, m := range members {
		totalDemand += m.Quantity
	}

	memberCount := len(members)

	// Compute quantity multiplier
	qtyMult := qtyMultiplier(totalDemand)

	// Compute member multiplier
	memberMult := memberMultiplier(memberCount)

	// Final discount: typical_discount * qty_mult * member_mult, capped at 50%
	typicalDiscount := pricingResult.TypicalPct / 100.0
	discountPct := typicalDiscount * qtyMult * memberMult
	if discountPct > 0.50 {
		discountPct = 0.50
	}

	offerPrice := pricingResult.ListPrice * (1 - discountPct)
	offerPrice = math.Round(offerPrice*100) / 100
	savingsPerUnit := math.Round((pricingResult.ListPrice-offerPrice)*100) / 100

	return &ConsolidatedOffer{
		GroupID:        groupID,
		Vendor:         group.TargetVendor,
		SKU:            group.TargetSKU,
		TotalDemand:    totalDemand,
		MemberCount:    memberCount,
		ListPrice:      pricingResult.ListPrice,
		OfferPrice:     offerPrice,
		SavingsPerUnit: savingsPerUnit,
		DiscountPct:    math.Round(discountPct*10000) / 100, // as percentage
	}, nil
}

// qtyMultiplier returns the quantity multiplier based on total demand.
func qtyMultiplier(totalDemand int) float64 {
	switch {
	case totalDemand >= 200:
		return 1.6
	case totalDemand >= 50:
		return 1.4
	case totalDemand >= 10:
		return 1.2
	default:
		return 1.0
	}
}

// memberMultiplier returns the member multiplier based on member count.
func memberMultiplier(memberCount int) float64 {
	switch {
	case memberCount >= 10:
		return 1.4
	case memberCount >= 5:
		return 1.25
	case memberCount >= 2:
		return 1.1
	default:
		return 1.0
	}
}

// GroupStore returns the underlying group store.
func (e *Engine) GroupStore() *Store {
	return e.groupStore
}
