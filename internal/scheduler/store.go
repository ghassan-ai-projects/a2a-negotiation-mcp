package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages scheduled negotiation runs in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a SchedulerStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate scheduler: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS negotiation_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor TEXT NOT NULL,
		strategy TEXT NOT NULL,
		cron_expr TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS schedule_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		schedule_id INTEGER NOT NULL,
		run_at TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		summary TEXT NOT NULL DEFAULT ''
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateSchedule creates a new negotiation schedule and returns it with the generated ID and timestamp.
func (s *Store) CreateSchedule(ctx context.Context, vendor, strategy, cronExpr string) (*Schedule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO negotiation_schedules (vendor, strategy, cron_expr, enabled, created_at)
		VALUES (?, ?, ?, 1, ?)
	`, vendor, strategy, cronExpr, now)
	if err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create schedule last insert id: %w", err)
	}
	return &Schedule{
		ID:        int(id),
		Vendor:    vendor,
		Strategy:  strategy,
		CronExpr:  cronExpr,
		Enabled:   true,
		CreatedAt: now,
	}, nil
}

// ListSchedules returns all negotiation schedules ordered by created_at DESC.
func (s *Store) ListSchedules(ctx context.Context) ([]Schedule, error) {
	query := `SELECT id, vendor, strategy, cron_expr, enabled, created_at FROM negotiation_schedules ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var sch Schedule
		if err := rows.Scan(&sch.ID, &sch.Vendor, &sch.Strategy, &sch.CronExpr, &sch.Enabled, &sch.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, sch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []Schedule{}
	}
	return schedules, nil
}

// DeleteSchedule removes a negotiation schedule by ID.
func (s *Store) DeleteSchedule(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM negotiation_schedules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

// GetScheduleResults returns all results for a given schedule ID, ordered by run_at DESC.
func (s *Store) GetScheduleResults(ctx context.Context, scheduleID int) ([]ScheduleResult, error) {
	query := `SELECT id, schedule_id, run_at, status, summary FROM schedule_results WHERE schedule_id = ? ORDER BY run_at DESC`
	rows, err := s.db.QueryContext(ctx, query, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("get schedule results: %w", err)
	}
	defer rows.Close()

	var results []ScheduleResult
	for rows.Next() {
		var r ScheduleResult
		if err := rows.Scan(&r.ID, &r.ScheduleID, &r.RunAt, &r.Status, &r.Summary); err != nil {
			return nil, fmt.Errorf("scan schedule result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []ScheduleResult{}
	}
	return results, nil
}
