package gamification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/learning"
)

// Engine manages gamification logic: streaks, leaderboard, and badges.
type Engine struct {
	store       *Store
	logger      *slog.Logger
	learningEng *learning.Engine // optional, used for learning-dependent badge checks
}

// New creates a gamification engine.
func New(store *Store, logger *slog.Logger) *Engine {
	return &Engine{store: store, logger: logger}
}

// SetLearningEngine attaches an optional learning engine for advanced badge checks.
func (e *Engine) SetLearningEngine(eng *learning.Engine) {
	e.learningEng = eng
}

// GetStreak returns current streak info for a user.
func (e *Engine) GetStreak(ctx context.Context, userID string) (*Streak, error) {
	streak, err := e.store.GetStreak(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get streak: %w", err)
	}
	if streak == nil {
		streak = &Streak{
			UserID:        userID,
			CurrentStreak: 0,
			LongestStreak: 0,
			TotalSavings:  0,
			TotalDeals:    0,
		}
	}
	return streak, nil
}

// RecordNegotiation updates streak and totals after a deal.
// If last negotiation was >48h ago, streak resets to 1.
// Otherwise increments streak.
func (e *Engine) RecordNegotiation(ctx context.Context, userID string, savings float64) error {
	now := time.Now().UTC()

	existing, err := e.store.GetStreak(ctx, userID)
	if err != nil {
		return fmt.Errorf("record negotiation get streak: %w", err)
	}

	var (
		currentStreak int
		longestStreak int
		totalSavings  float64
		totalDeals    int
	)

	if existing != nil {
		totalSavings = existing.TotalSavings
		totalDeals = existing.TotalDeals
		longestStreak = existing.LongestStreak

		// Check if gap > 48h from last negotiation
		if !existing.LastNegotiationAt.IsZero() && now.Sub(existing.LastNegotiationAt) > 48*time.Hour {
			currentStreak = 1
		} else if !existing.LastNegotiationAt.IsZero() {
			currentStreak = existing.CurrentStreak + 1
		} else {
			currentStreak = 1
		}
	} else {
		currentStreak = 1
	}

	totalDeals++
	totalSavings += savings

	if currentStreak > longestStreak {
		longestStreak = currentStreak
	}

	streak := &Streak{
		UserID:            userID,
		CurrentStreak:     currentStreak,
		LongestStreak:     longestStreak,
		LastNegotiationAt: now,
		TotalSavings:      totalSavings,
		TotalDeals:        totalDeals,
	}

	if err := e.store.UpsertStreak(ctx, streak); err != nil {
		return fmt.Errorf("record negotiation upsert: %w", err)
	}

	e.logger.Debug("recorded negotiation",
		"user_id", userID,
		"savings", savings,
		"current_streak", currentStreak,
		"total_deals", totalDeals,
	)

	return nil
}

// GetLeaderboard returns top savers.
func (e *Engine) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	return e.store.GetLeaderboard(ctx, limit)
}

// GetBadges returns all badges with earned status for a user.
func (e *Engine) GetBadges(ctx context.Context, userID string) ([]Badge, error) {
	allBadges := allBadgeDefs()

	earned, err := e.store.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get badges: %w", err)
	}

	now := time.Now()
	result := make([]Badge, len(allBadges))
	for i, def := range allBadges {
		result[i] = Badge{
			ID:   def.ID,
			Name: def.Name,
			Icon: def.Icon,
		}
		if earnedAt, ok := earned[def.ID]; ok {
			result[i].Earned = true
			t := earnedAt
			result[i].EarnedAt = &t
		} else {
			result[i].Earned = false
		}
		_ = now
	}

	return result, nil
}

// CheckAndAwardBadges checks which badges a user qualifies for and awards them.
func (e *Engine) CheckAndAwardBadges(ctx context.Context, userID string, streak *Streak) ([]Badge, error) {
	now := time.Now().UTC()

	allDefs := allBadgeDefs()
	earned, err := e.store.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check badges get earned: %w", err)
	}

	var awarded []Badge

	for _, def := range allDefs {
		// Skip already earned badges
		if _, ok := earned[def.ID]; ok {
			continue
		}

		qualified, err := e.checkBadgeCondition(ctx, def, userID, streak)
		if err != nil {
			e.logger.Warn("badge check error", "badge_id", def.ID, "error", err.Error())
			continue
		}

		if qualified {
			if err := e.store.AwardBadge(ctx, userID, def.ID, now); err != nil {
				e.logger.Warn("badge award error", "badge_id", def.ID, "error", err.Error())
				continue
			}
			awarded = append(awarded, Badge{
				ID:       def.ID,
				Name:     def.Name,
				Icon:     def.Icon,
				Earned:   true,
				EarnedAt: &now,
			})
			e.logger.Info("badge awarded", "user_id", userID, "badge_id", def.ID, "name", def.Name)
		}
	}

	// If nothing new was awarded, return them all with their current status
	if len(awarded) == 0 {
		return e.GetBadges(ctx, userID)
	}

	return awarded, nil
}

// ─── Badge Definitions ───

type badgeDef struct {
	ID   string
	Name string
	Icon string
	Check func(ctx context.Context, store *Store, userID string, streak *Streak, eng *learning.Engine) (bool, error)
}

func allBadgeDefs() []badgeDef {
	return []badgeDef{
		{
			ID: "first_deal", Name: "First Deal", Icon: "🏆",
			Check: func(_ context.Context, _ *Store, _ string, streak *Streak, _ *learning.Engine) (bool, error) {
				return streak.TotalDeals >= 1, nil
			},
		},
		{
			ID: "streak_5", Name: "On Fire", Icon: "🔥",
			Check: func(_ context.Context, _ *Store, _ string, streak *Streak, _ *learning.Engine) (bool, error) {
				return streak.CurrentStreak >= 5, nil
			},
		},
		{
			ID: "thousand_club", Name: "Thousand Club", Icon: "💰",
			Check: func(_ context.Context, _ *Store, _ string, streak *Streak, _ *learning.Engine) (bool, error) {
				return streak.TotalSavings >= 1000, nil
			},
		},
		{
			ID: "perfectionist", Name: "Perfectionist", Icon: "🎯",
			Check: func(ctx context.Context, _ *Store, _ string, streak *Streak, eng *learning.Engine) (bool, error) {
				if eng == nil {
					return false, nil
				}
				insights, err := eng.GetGlobalInsights(ctx)
				if err != nil {
					return false, fmt.Errorf("get global insights: %w", err)
				}
				totalOutcomes, _ := insights["total_outcomes"].(int)
				overallWinRate, _ := insights["overall_win_rate"].(float64)
				// >90% win rate over 10+ deals
				return totalOutcomes >= 10 && overallWinRate > 90.0, nil
			},
		},
		{
			ID: "power_negotiator", Name: "Power Negotiator", Icon: "🚀",
			Check: func(_ context.Context, _ *Store, _ string, streak *Streak, _ *learning.Engine) (bool, error) {
				return streak.TotalDeals >= 50, nil
			},
		},
		{
			ID: "saas_master", Name: "SaaS Master", Icon: "💎",
			Check: func(ctx context.Context, store *Store, userID string, streak *Streak, eng *learning.Engine) (bool, error) {
				if streak.TotalDeals == 0 {
					return false, nil
				}
				// Count distinct vendors from learning outcomes where eng != nil
				if eng == nil {
					return false, nil
				}
				// Query learning outcomes for distinct vendor count
				// Use the learning engine's underlying store via its public methods
				insights, err := eng.GetGlobalInsights(ctx)
				if err != nil {
					return false, fmt.Errorf("get global insights: %w", err)
				}
				// Get top vendors count from the learning data
				topVendors, _ := insights["top_vendors"].([]map[string]interface{})
				_ = store
				_ = userID
				return len(topVendors) >= 10, nil
			},
		},
	}
}

func (e *Engine) checkBadgeCondition(ctx context.Context, def badgeDef, userID string, streak *Streak) (bool, error) {
	return def.Check(ctx, e.store, userID, streak, e.learningEng)
}
