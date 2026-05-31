package backupmgr

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

// Store manages system backups in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a BackupStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate backupmgr: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS system_backups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tables TEXT NOT NULL DEFAULT 'all',
		size_bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'completed',
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS backup_schedule (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		cron TEXT NOT NULL,
		tables TEXT NOT NULL DEFAULT 'all',
		updated_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// simulatedSize returns a random size between 1KB and 10MB for simulation.
func simulatedSize() int {
	n, err := rand.Int(rand.Reader, big.NewInt(10*1024*1024-1024+1))
	if err != nil {
		return 1024
	}
	return int(n.Int64()) + 1024
}

// CreateBackup creates a new backup record with a simulated size.
func (s *Store) CreateBackup(ctx context.Context, tables string) (*Backup, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	size := simulatedSize()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO system_backups (tables, size_bytes, status, created_at)
		VALUES (?, ?, 'completed', ?)
	`, tables, size, now)
	if err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create backup last insert id: %w", err)
	}
	return &Backup{
		ID:        int(id),
		Tables:    tables,
		SizeBytes: size,
		Status:    "completed",
		CreatedAt: now,
	}, nil
}

// ListBackups returns all backups ordered by created_at DESC.
func (s *Store) ListBackups(ctx context.Context) ([]Backup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tables, size_bytes, status, created_at
		FROM system_backups
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var b Backup
		if err := rows.Scan(&b.ID, &b.Tables, &b.SizeBytes, &b.Status, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if backups == nil {
		backups = []Backup{}
	}
	return backups, nil
}

// RestoreBackup retrieves a backup by ID and updates its status to "restored".
func (s *Store) RestoreBackup(ctx context.Context, id int) (*Backup, error) {
	var b Backup
	err := s.db.QueryRowContext(ctx, `
		SELECT id, tables, size_bytes, status, created_at
		FROM system_backups WHERE id = ?
	`, id).Scan(&b.ID, &b.Tables, &b.SizeBytes, &b.Status, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backup not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("restore backup: %w", err)
	}

	b.Status = "restored"
	_, err = s.db.ExecContext(ctx, `
		UPDATE system_backups SET status = 'restored' WHERE id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("restore backup update status: %w", err)
	}
	return &b, nil
}

// SetBackupSchedule sets or updates the backup schedule (single row, upsert).
func (s *Store) SetBackupSchedule(ctx context.Context, cron, tables string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO backup_schedule (id, cron, tables, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			cron = excluded.cron,
			tables = excluded.tables,
			updated_at = excluded.updated_at
	`, cron, tables, now)
	if err != nil {
		return fmt.Errorf("set backup schedule: %w", err)
	}
	return nil
}
