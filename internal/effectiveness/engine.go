package effectiveness

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Engine computes negotiation effectiveness scores.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates an effectiveness Engine.
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Score computes the effectiveness score for a user over the given period.
// period: "90d", "30d", "1y", etc. Empty defaults to "90d".
func (e *Engine) Score(ctx context.Context, userID string, period string) (EffectivenessScore, error) {
	if period == "" {
		period = "90d"
	}
	since := parsePeriodEffectiveness(period)

	// --- Component 1: WinRate (40%) ---
	winRate, err := e.winRate(ctx, since, userID)
	if err != nil {
		e.logger.Warn("effectiveness: win rate query failed", "error", err)
		winRate = 0
	}

	// --- Component 2: DiscountDepth (30%) ---
	avgDiscount, err := e.avgDiscountPct(ctx, since, userID)
	if err != nil {
		e.logger.Warn("effectiveness: avg discount query failed", "error", err)
		avgDiscount = 0
	}

	// --- Component 3: SavingsVolume (20%) ---
	totalSavings, err := e.totalSavings(ctx, since, userID)
	if err != nil {
		e.logger.Warn("effectiveness: total savings query failed", "error", err)
		totalSavings = 0
	}

	// --- Component 4: StreakConsistency (10%) ---
	streak, err := e.currentStreak(ctx, userID)
	if err != nil {
		e.logger.Warn("effectiveness: streak query failed", "error", err)
		streak = 0
	}

	// Compute scores
	winRateScore := winRate // already 0-100

	discountDepthScore := (avgDiscount / 50.0) * 100
	if discountDepthScore > 100 {
		discountDepthScore = 100
	}

	var savingsVolumeScore float64
	if totalSavings > 0 {
		savingsVolumeScore = (math.Log10(totalSavings) / math.Log10(100000)) * 100
	} else {
		savingsVolumeScore = 0
	}
	if savingsVolumeScore > 100 {
		savingsVolumeScore = 100
	}

	streakConsistencyScore := (float64(streak) / 30.0) * 100
	if streakConsistencyScore > 100 {
		streakConsistencyScore = 100
	}

	// Composite
	overall := winRateScore*0.40 + discountDepthScore*0.30 + savingsVolumeScore*0.20 + streakConsistencyScore*0.10
	overall = math.Round(overall*100) / 100

	components := []ScoreComponent{
		{Name: "Win Rate", Score: math.Round(winRateScore*100) / 100, Weight: 0.40},
		{Name: "Discount Depth", Score: math.Round(discountDepthScore*100) / 100, Weight: 0.30},
		{Name: "Savings Volume", Score: math.Round(savingsVolumeScore*100) / 100, Weight: 0.20},
		{Name: "Streak Consistency", Score: math.Round(streakConsistencyScore*100) / 100, Weight: 0.10},
	}

	// Trend
	trend, err := e.scoreTrend(ctx, since, userID)
	if err != nil {
		e.logger.Warn("effectiveness: trend query failed", "error", err)
		trend = []ScoreTrendPoint{}
	}

	// Tips
	tips := e.generateTips(winRate, avgDiscount, totalSavings, streak)

	// Vs average
	avgScore, err := e.averageScore(ctx, since)
	if err != nil {
		e.logger.Warn("effectiveness: average score query failed", "error", err)
		avgScore = 0
	}
	vsAverage := math.Round((overall-avgScore)*100) / 100

	return EffectivenessScore{
		UserID:       userID,
		Period:       period,
		OverallScore: overall,
		Components:   components,
		Trend:        trend,
		VsAverage:    vsAverage,
		Tips:         tips,
	}, nil
}

func (e *Engine) winRate(ctx context.Context, since time.Time, userID string) (float64, error) {
	var won, total sql.NullFloat64
	// Since userID is not tracked in deal_outcomes directly, we use it for session-based filtering
	if userID != "" {
		err := e.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN outcome = 'won' THEN 1 ELSE 0 END), 0),
			       COALESCE(COUNT(*), 0)
			FROM negotiation_sessions
			WHERE created_at >= ? AND outcome IN ('won','lost')
		`, since.Format(time.RFC3339)).Scan(&won, &total)
		if err != nil {
			return 0, err
		}
	} else {
		err := e.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN outcome = 'won' THEN 1 ELSE 0 END), 0),
			       COALESCE(COUNT(*), 0)
			FROM negotiation_sessions
			WHERE created_at >= ? AND outcome IN ('won','lost')
		`, since.Format(time.RFC3339)).Scan(&won, &total)
		if err != nil {
			return 0, err
		}
	}
	if total.Float64 == 0 {
		return 0, nil
	}
	return math.Round((won.Float64/total.Float64)*10000) / 100, nil
}

func (e *Engine) avgDiscountPct(ctx context.Context, since time.Time, userID string) (float64, error) {
	var avg sql.NullFloat64
	err := e.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(discount_pct), 0)
		FROM deal_outcomes
		WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&avg)
	if err != nil {
		return 0, err
	}
	return avg.Float64, nil
}

func (e *Engine) totalSavings(ctx context.Context, since time.Time, userID string) (float64, error) {
	var savings sql.NullFloat64
	err := e.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(list_price - final_price), 0)
		FROM deal_outcomes
		WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&savings)
	if err != nil {
		return 0, err
	}
	return savings.Float64, nil
}

func (e *Engine) currentStreak(ctx context.Context, userID string) (int, error) {
	if userID == "" {
		return 0, nil
	}
	var streak sql.NullInt64
	err := e.db.QueryRowContext(ctx, `
		SELECT current_streak FROM user_streaks WHERE user_id = ?
	`, userID).Scan(&streak)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return int(streak.Int64), nil
}

func (e *Engine) scoreTrend(ctx context.Context, since time.Time, userID string) ([]ScoreTrendPoint, error) {
	// Build monthly trend points by computing score for each month in range
	// We'll use deal_outcomes aggregated by month to approximate
	var rows *sql.Rows
	var err error
	if userID != "" {
		rows, err = e.db.QueryContext(ctx, `
			SELECT strftime('%Y-%m', created_at) AS month,
			       COUNT(*) AS total,
			       COALESCE(AVG(discount_pct), 0) AS avg_disc
			FROM deal_outcomes
			WHERE created_at >= ?
			GROUP BY strftime('%Y-%m', created_at)
			ORDER BY month ASC
		`, since.Format(time.RFC3339))
	} else {
		rows, err = e.db.QueryContext(ctx, `
			SELECT strftime('%Y-%m', created_at) AS month,
			       COUNT(*) AS total,
			       COALESCE(AVG(discount_pct), 0) AS avg_disc
			FROM deal_outcomes
			WHERE created_at >= ?
			GROUP BY strftime('%Y-%m', created_at)
			ORDER BY month ASC
		`, since.Format(time.RFC3339))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []ScoreTrendPoint
	for rows.Next() {
		var month string
		var total int
		var avgDisc float64
		if err := rows.Scan(&month, &total, &avgDisc); err != nil {
			return nil, err
		}
		// Approximate score for the month based on discount depth (simplified trend)
		score := (avgDisc / 50.0) * 100
		if score > 100 {
			score = 100
		}
		points = append(points, ScoreTrendPoint{
			Date:  month,
			Score: math.Round(score*100) / 100,
		})
	}
	return points, rows.Err()
}

func (e *Engine) averageScore(ctx context.Context, since time.Time) (float64, error) {
	var avg sql.NullFloat64
	err := e.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(discount_pct), 0)
		FROM deal_outcomes WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&avg)
	if err != nil {
		return 0, err
	}
	// Rough approximation: average score based on average discount depth
	score := (avg.Float64 / 50.0) * 100
	if score > 100 {
		score = 100
	}
	return math.Round(score*100) / 100, nil
}

func (e *Engine) generateTips(winRate, avgDiscount, totalSavings float64, streak int) []string {
	var tips []string

	if winRate < 75 {
		tips = append(tips, fmt.Sprintf("Improve win rate from %.0f%% to 75%%", winRate))
	}
	if winRate >= 75 && winRate < 90 {
		tips = append(tips, fmt.Sprintf("Win rate is strong at %.0f%% — aim for 90%%", winRate))
	}

	if avgDiscount < 20 {
		tips = append(tips, fmt.Sprintf("Increase average discount from %.1f%% to 20%%", avgDiscount))
	}
	if avgDiscount > 40 {
		tips = append(tips, fmt.Sprintf("Discount depth is excellent at %.1f%% — maintain leverage", avgDiscount))
	}

	if totalSavings < 10000 {
		tips = append(tips, "Focus on higher-value deals to boost savings volume")
	}

	if streak < 5 {
		tips = append(tips, "Negotiate more consistently to build streak")
	}
	if streak >= 5 && streak < 20 {
		tips = append(tips, fmt.Sprintf("Good streak of %d — keep the momentum going", streak))
	}

	if len(tips) == 0 {
		tips = []string{"Excellent performance across all metrics — keep it up!"}
		if streak > 20 {
			tips = append(tips, fmt.Sprintf("Impressive %d-deal streak — you're on fire!", streak))
		}
	}

	return tips
}

func parsePeriodEffectiveness(period string) time.Time {
	now := time.Now().UTC()
	switch {
	case len(period) > 1 && period[len(period)-1] == 'y':
		n := 0
		fmt.Sscanf(period, "%dy", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(-n, 0, 0)
	case len(period) > 2 && period[len(period)-1] == 'd':
		n := 0
		fmt.Sscanf(period, "%dd", &n)
		if n <= 0 {
			n = 90
		}
		return now.AddDate(0, 0, -n)
	case len(period) > 2 && period[len(period)-1] == 'm':
		n := 0
		fmt.Sscanf(period, "%dm", &n)
		if n <= 0 {
			n = 3
		}
		return now.AddDate(0, -n, 0)
	default:
		return now.AddDate(0, 0, -90)
	}
}
