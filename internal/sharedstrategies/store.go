package sharedstrategies

import (
	"context"
	"database/sql"
	"fmt"
)

// Store persists shared strategies.
type Store struct {
	db *sql.DB
}

// NewStore creates a new shared strategies store sharing the given DB.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("sharedstrategies migrate: %w", err)
	}
	return s, nil
}

// DB exposes the underlying DB for sharing.
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS shared_strategies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			notes TEXT,
			strategy_type TEXT DEFAULT 'balanced',
			usage_count INTEGER DEFAULT 0,
			created_at TEXT
		)
	`)
	return err
}

// Create inserts a new shared strategy.
func (s *Store) Create(ctx context.Context, st *SharedStrategy) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO shared_strategies (id, name, notes, strategy_type, usage_count, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		st.ID, st.Name, st.Notes, st.StrategyType, st.UsageCount, st.CreatedAt,
	)
	return err
}

// List returns all shared strategies.
func (s *Store) List(ctx context.Context) ([]SharedStrategy, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, notes, strategy_type, usage_count, created_at FROM shared_strategies ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SharedStrategy
	for rows.Next() {
		var st SharedStrategy
		if err := rows.Scan(&st.ID, &st.Name, &st.Notes, &st.StrategyType, &st.UsageCount, &st.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

// Get returns a single shared strategy by ID.
func (s *Store) Get(ctx context.Context, id string) (*SharedStrategy, error) {
	var st SharedStrategy
	err := s.db.QueryRowContext(ctx, `SELECT id, name, notes, strategy_type, usage_count, created_at FROM shared_strategies WHERE id = ?`, id).
		Scan(&st.ID, &st.Name, &st.Notes, &st.StrategyType, &st.UsageCount, &st.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// IncrementUsage increments the usage count for a shared strategy.
func (s *Store) IncrementUsage(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE shared_strategies SET usage_count = usage_count + 1 WHERE id = ?`, id)
	return err
}

// Delete removes a shared strategy by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shared_strategies WHERE id = ?`, id)
	return err
}
