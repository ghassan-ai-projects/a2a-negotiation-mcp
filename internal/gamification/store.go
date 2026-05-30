package gamification

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store handles persistence for gamification data.
type Store struct {
	db *sql.DB
}

// NewStore creates a gamification store backed by the given DB.
// Tables are auto-created if they don't exist.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate gamification: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS user_streaks (
		user_id TEXT PRIMARY KEY,
		current_streak INTEGER DEFAULT 0,
		longest_streak INTEGER DEFAULT 0,
		last_negotiation_at TEXT,
		total_savings REAL DEFAULT 0,
		total_deals INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS user_badges (
		user_id TEXT NOT NULL,
		badge_id TEXT NOT NULL,
		earned_at TEXT NOT NULL,
		PRIMARY KEY (user_id, badge_id)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// ─── Streak Operations ───

// UpsertStreak inserts or updates a user's streak record.
func (s *Store) UpsertStreak(ctx context.Context, streak *Streak) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_streaks (user_id, current_streak, longest_streak, last_negotiation_at, total_savings, total_deals)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			current_streak = excluded.current_streak,
			longest_streak = excluded.longest_streak,
			last_negotiation_at = excluded.last_negotiation_at,
			total_savings = excluded.total_savings,
			total_deals = excluded.total_deals
	`, streak.UserID, streak.CurrentStreak, streak.LongestStreak,
		streak.LastNegotiationAt.Format(time.RFC3339),
		streak.TotalSavings, streak.TotalDeals)
	if err != nil {
		return fmt.Errorf("upsert streak: %w", err)
	}
	return nil
}

// GetStreak returns the streak record for a user.
// Returns nil, nil if not found.
func (s *Store) GetStreak(ctx context.Context, userID string) (*Streak, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT user_id, current_streak, longest_streak, last_negotiation_at, total_savings, total_deals
		FROM user_streaks WHERE user_id = ?
	`, userID)

	var streak Streak
	var lastNegStr string
	err := row.Scan(&streak.UserID, &streak.CurrentStreak, &streak.LongestStreak,
		&lastNegStr, &streak.TotalSavings, &streak.TotalDeals)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get streak: %w", err)
	}
	streak.LastNegotiationAt, err = time.Parse(time.RFC3339, lastNegStr)
	if err != nil {
		return nil, fmt.Errorf("parse last_negotiation_at: %w", err)
	}
	return &streak, nil
}

// GetLeaderboard returns top users ordered by total savings descending.
func (s *Store) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, total_savings, total_deals, current_streak
		FROM user_streaks
		ORDER BY total_savings DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.TotalSavings, &e.TotalDeals, &e.Streak); err != nil {
			return nil, fmt.Errorf("scan leaderboard: %w", err)
		}
		if e.TotalDeals > 0 {
			e.AvgSavings = e.TotalSavings / float64(e.TotalDeals)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ─── Badge Operations ───

// AwardBadge records an earned badge for a user.
func (s *Store) AwardBadge(ctx context.Context, userID, badgeID string, earnedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO user_badges (user_id, badge_id, earned_at)
		VALUES (?, ?, ?)
	`, userID, badgeID, earnedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("award badge: %w", err)
	}
	return nil
}

// GetUserBadges returns all badge IDs a user has earned, keyed by badge_id -> earned_at.
func (s *Store) GetUserBadges(ctx context.Context, userID string) (map[string]time.Time, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT badge_id, earned_at FROM user_badges WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user badges: %w", err)
	}
	defer rows.Close()

	badges := make(map[string]time.Time)
	for rows.Next() {
		var badgeID, earnedStr string
		if err := rows.Scan(&badgeID, &earnedStr); err != nil {
			return nil, fmt.Errorf("scan badge: %w", err)
		}
		earnedAt, err := time.Parse(time.RFC3339, earnedStr)
		if err != nil {
			return nil, fmt.Errorf("parse badge earned_at: %w", err)
		}
		badges[badgeID] = earnedAt
	}
	return badges, rows.Err()
}
