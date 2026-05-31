package scheduler

type Schedule struct {
	ID        int    `json:"id"`
	Vendor    string `json:"vendor"`
	Strategy  string `json:"strategy"`
	CronExpr  string `json:"cron_expr"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type ScheduleResult struct {
	ID         int    `json:"id"`
	ScheduleID int    `json:"schedule_id"`
	RunAt      string `json:"run_at"`
	Status     string `json:"status"`
	Summary    string `json:"summary"`
}
