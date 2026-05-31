package pushnotif

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages push notification device registrations in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a PushStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate pushnotif: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS push_devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token TEXT NOT NULL UNIQUE,
		platform TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// RegisterDevice registers a device token. If the token already exists, it is ignored.
func (s *Store) RegisterDevice(ctx context.Context, token, platform string) (*Device, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO push_devices (token, platform, created_at)
		VALUES (?, ?, ?)
	`, token, platform, now)
	if err != nil {
		return nil, fmt.Errorf("register device: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("register device last insert id: %w", err)
	}

	// If INSERT OR IGNORE matched an existing row, LastInsertId returns 0.
	// In that case, fetch the existing device by token.
	if id == 0 {
		existing, err := s.getDeviceByToken(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("register device lookup: %w", err)
		}
		return existing, nil
	}

	return &Device{
		ID:        int(id),
		Token:     token,
		Platform:  platform,
		CreatedAt: now,
	}, nil
}

func (s *Store) getDeviceByToken(ctx context.Context, token string) (*Device, error) {
	var d Device
	err := s.db.QueryRowContext(ctx, `
		SELECT id, token, platform, created_at
		FROM push_devices WHERE token = ?
	`, token).Scan(&d.ID, &d.Token, &d.Platform, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get device by token: %w", err)
	}
	return &d, nil
}

// ListDevices returns all registered push devices.
func (s *Store) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, token, platform, created_at
		FROM push_devices ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Token, &d.Platform, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if devices == nil {
		devices = []Device{}
	}
	return devices, nil
}
