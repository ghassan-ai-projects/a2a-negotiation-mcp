package dataretention

// RetentionPolicy defines how long a data type is retained and what action to
// take when the retention period expires.
type RetentionPolicy struct {
	DataType      string `json:"data_type"`
	RetentionDays int    `json:"retention_days"`
	Action        string `json:"action"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// PurgeResult describes the outcome of a purge operation for a single data type.
type PurgeResult struct {
	DataType       string `json:"data_type"`
	RecordsDeleted int    `json:"records_deleted"`
	DryRun         bool   `json:"dry_run"`
}
