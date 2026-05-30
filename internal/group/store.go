package group

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store provides buying group and member data operations backed by SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store using an existing DB connection and ensures schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate group: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS buying_groups (
		id TEXT PRIMARY KEY,
		target_vendor TEXT,
		target_sku TEXT,
		target_price REAL,
		min_members INTEGER DEFAULT 1,
		status TEXT DEFAULT 'forming',
		created_at TEXT,
		expires_at TEXT
	);

	CREATE TABLE IF NOT EXISTS group_members (
		id TEXT PRIMARY KEY,
		group_id TEXT REFERENCES buying_groups(id),
		user_id TEXT,
		quantity INTEGER DEFAULT 1,
		max_price REAL,
		committed_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_group_members_group ON group_members(group_id);
	CREATE INDEX IF NOT EXISTS idx_buying_groups_status ON buying_groups(status);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateGroup inserts a new buying group.
func (s *Store) CreateGroup(ctx context.Context, g *BuyingGroup) error {
	g.ID = uuid.New().String()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO buying_groups (id, target_vendor, target_sku, target_price, min_members, status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, g.ID, g.TargetVendor, g.TargetSKU, g.TargetPrice, g.MinMembers, g.Status,
		g.CreatedAt.Format(time.RFC3339), g.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return nil
}

// GetGroup retrieves a buying group by ID.
func (s *Store) GetGroup(ctx context.Context, id string) (*BuyingGroup, error) {
	var g BuyingGroup
	var createdAt, expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, target_vendor, target_sku, target_price, min_members, status, created_at, expires_at
		FROM buying_groups WHERE id = ?
	`, id).Scan(&g.ID, &g.TargetVendor, &g.TargetSKU, &g.TargetPrice, &g.MinMembers,
		&g.Status, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	g.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	return &g, nil
}

// JoinGroup adds a member to an existing group. Returns error if group is not in "forming" status.
func (s *Store) JoinGroup(ctx context.Context, m *GroupMember) error {
	// Verify group exists and is in forming status
	var status string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM buying_groups WHERE id = ?", m.GroupID).Scan(&status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("group not found: %s", m.GroupID)
	}
	if err != nil {
		return fmt.Errorf("check group: %w", err)
	}
	if status != "forming" {
		return fmt.Errorf("group %s is not accepting members (status: %s)", m.GroupID, status)
	}

	m.ID = uuid.New().String()
	m.CommittedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO group_members (id, group_id, user_id, quantity, max_price, committed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, m.ID, m.GroupID, m.UserID, m.Quantity, m.MaxPrice, m.CommittedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("join group: %w", err)
	}
	return nil
}

// GetMembers returns all members of a group.
func (s *Store) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, user_id, quantity, max_price, committed_at
		FROM group_members WHERE group_id = ? ORDER BY committed_at
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		var committedAt string
		if err := rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.Quantity, &m.MaxPrice, &committedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		m.CommittedAt, _ = time.Parse(time.RFC3339, committedAt)
		members = append(members, m)
	}
	return members, rows.Err()
}

// GetMemberCount returns the number of members in a group.
func (s *Store) GetMemberCount(ctx context.Context, groupID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM group_members WHERE group_id = ?", groupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get member count: %w", err)
	}
	return count, nil
}

// UpdateGroupStatus updates the status of a buying group.
func (s *Store) UpdateGroupStatus(ctx context.Context, groupID, status string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE buying_groups SET status = ? WHERE id = ?", status, groupID)
	if err != nil {
		return fmt.Errorf("update group status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("group not found: %s", groupID)
	}
	return nil
}

// GetActiveGroups returns all groups that are not expired or completed.
func (s *Store) GetActiveGroups(ctx context.Context) ([]BuyingGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, target_vendor, target_sku, target_price, min_members, status, created_at, expires_at
		FROM buying_groups WHERE status IN ('forming', 'negotiating') ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get active groups: %w", err)
	}
	defer rows.Close()

	var groups []BuyingGroup
	for rows.Next() {
		var g BuyingGroup
		var createdAt, expiresAt string
		if err := rows.Scan(&g.ID, &g.TargetVendor, &g.TargetSKU, &g.TargetPrice, &g.MinMembers,
			&g.Status, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		g.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
