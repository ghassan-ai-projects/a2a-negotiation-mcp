package gamification

import "time"

type Streak struct {
	UserID            string    `json:"user_id"`
	CurrentStreak     int       `json:"current_streak"`
	LongestStreak     int       `json:"longest_streak"`
	LastNegotiationAt time.Time `json:"last_negotiation_at"`
	TotalSavings      float64   `json:"total_savings"`
	TotalDeals        int       `json:"total_deals"`
}

type LeaderboardEntry struct {
	UserID       string  `json:"user_id"`
	TotalSavings float64 `json:"total_savings"`
	TotalDeals   int     `json:"total_deals"`
	AvgSavings   float64 `json:"avg_savings"`
	Streak       int     `json:"streak"`
}

type Badge struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Icon     string     `json:"icon"`
	Earned   bool       `json:"earned"`
	EarnedAt *time.Time `json:"earned_at,omitempty"`
}
