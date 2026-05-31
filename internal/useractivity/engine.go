package useractivity

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Engine reads user activity data from the history store tables.
type Engine struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewEngine creates a user activity engine (read-only).
func NewEngine(db *sql.DB, logger *slog.Logger) *Engine {
	return &Engine{db: db, logger: logger}
}

// Report generates a user activity report for the given period.
func (e *Engine) Report(ctx context.Context, userID, period string) (*UserActivityReport, error) {
	if period == "" {
		period = "30d"
	}
	since := parsePeriod(period)

	report := &UserActivityReport{
		UserID: userID,
		Period: period,
	}

	if err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM negotiation_sessions WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&report.TotalSessions); err != nil {
		e.logger.Warn("useractivity: sessions query failed", "error", err)
	}

	if err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM negotiation_sessions WHERE created_at >= ? AND outcome IN ('won','lost')
	`, since.Format(time.RFC3339)).Scan(&report.CompletedNegotiations); err != nil {
		e.logger.Warn("useractivity: completed query failed", "error", err)
	}

	if err := e.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(list_price - final_price), 0) FROM deal_outcomes WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&report.TotalSavings); err != nil {
		e.logger.Warn("useractivity: savings query failed", "error", err)
	}

	if err := e.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT DATE(created_at)) FROM negotiation_sessions WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&report.ActiveDays); err != nil {
		e.logger.Warn("useractivity: active days query failed", "error", err)
	}

	var lastActive sql.NullString
	if err := e.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM negotiation_sessions WHERE created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&lastActive); err != nil {
		e.logger.Warn("useractivity: last active query failed", "error", err)
	}
	if lastActive.Valid {
		report.LastActive = lastActive.String
	}

	rows, err := e.db.QueryContext(ctx, `
		SELECT strategy, COUNT(*) AS cnt FROM negotiation_sessions
		WHERE created_at >= ? AND strategy != ''
		GROUP BY strategy ORDER BY cnt DESC
	`, since.Format(time.RFC3339))
	if err != nil {
		e.logger.Warn("useractivity: strategies query failed", "error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var su StrategyUsage
			if err := rows.Scan(&su.Strategy, &su.Count); err != nil {
				break
			}
			report.FavoriteStrategies = append(report.FavoriteStrategies, su)
		}
	}

	rows2, err := e.db.QueryContext(ctx, `
		SELECT vendor, COUNT(*) AS cnt FROM negotiation_sessions
		WHERE created_at >= ? AND vendor != ''
		GROUP BY vendor ORDER BY cnt DESC LIMIT 10
	`, since.Format(time.RFC3339))
	if err != nil {
		e.logger.Warn("useractivity: vendors query failed", "error", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var vu VendorUsage
			if err := rows2.Scan(&vu.Vendor, &vu.Count); err != nil {
				break
			}
			report.TopVendors = append(report.TopVendors, vu)
		}
	}

	return report, nil
}

func parsePeriod(period string) time.Time {
	now := time.Now().UTC()
	switch {
	case len(period) > 1 && period[len(period)-1] == 'y':
		var n int
		fmt.Sscanf(period, "%dy", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(-n, 0, 0)
	case len(period) > 2 && period[len(period)-1] == 'm':
		var n int
		fmt.Sscanf(period, "%dm", &n)
		if n <= 0 {
			n = 1
		}
		return now.AddDate(0, -n, 0)
	case len(period) > 1 && period[len(period)-1] == 'd':
		var n int
		fmt.Sscanf(period, "%dd", &n)
		if n <= 0 {
			n = 30
		}
		return now.AddDate(0, 0, -n)
	default:
		return now.AddDate(0, 0, -30)
	}
}
