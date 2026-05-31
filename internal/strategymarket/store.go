package strategymarket

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store manages negotiation strategies in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a StrategyMarketStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate strategymarket: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS strategy_marketplace (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		config TEXT NOT NULL DEFAULT '{}',
		category TEXT NOT NULL DEFAULT '',
		rating REAL NOT NULL DEFAULT 0,
		rating_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// PublishStrategy creates a new strategy and returns it with the generated ID and timestamp.
func (s *Store) PublishStrategy(ctx context.Context, name, description, config, category string) (*Strategy, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO strategy_marketplace (name, description, config, category, rating, rating_count, created_at)
		VALUES (?, ?, ?, ?, 0, 0, ?)
	`, name, description, config, category, now)
	if err != nil {
		return nil, fmt.Errorf("publish strategy: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("publish strategy last insert id: %w", err)
	}
	return &Strategy{
		ID:          int(id),
		Name:        name,
		Description: description,
		Config:      config,
		Category:    category,
		Rating:      0,
		RatingCount: 0,
		CreatedAt:   now,
	}, nil
}

// BrowseStrategies returns strategies, optionally filtered by category, sorted as requested.
func (s *Store) BrowseStrategies(ctx context.Context, category, sort string) ([]Strategy, error) {
	query := `SELECT id, name, description, config, category, rating, rating_count, created_at FROM strategy_marketplace`
	args := []any{}
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	switch sort {
	case "rating":
		query += ` ORDER BY rating DESC`
	default:
		query += ` ORDER BY created_at DESC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("browse strategies: %w", err)
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var st Strategy
		if err := rows.Scan(&st.ID, &st.Name, &st.Description, &st.Config, &st.Category, &st.Rating, &st.RatingCount, &st.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan strategy: %w", err)
		}
		strategies = append(strategies, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if strategies == nil {
		strategies = []Strategy{}
	}
	return strategies, nil
}

// ImportStrategy retrieves a single strategy by ID.
func (s *Store) ImportStrategy(ctx context.Context, id int) (*Strategy, error) {
	var st Strategy
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, config, category, rating, rating_count, created_at
		FROM strategy_marketplace WHERE id = ?
	`, id).Scan(&st.ID, &st.Name, &st.Description, &st.Config, &st.Category, &st.Rating, &st.RatingCount, &st.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("strategy not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("import strategy: %w", err)
	}
	return &st, nil
}

// RateStrategy updates a strategy's rating using a weighted average.
func (s *Store) RateStrategy(ctx context.Context, id int, rating float64) (*Strategy, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("rate strategy begin tx: %w", err)
	}
	defer tx.Rollback()

	var oldRating float64
	var oldCount int
	err = tx.QueryRowContext(ctx, `
		SELECT rating, rating_count FROM strategy_marketplace WHERE id = ?
	`, id).Scan(&oldRating, &oldCount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("strategy not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("rate strategy fetch: %w", err)
	}

	newCount := oldCount + 1
	newRating := (oldRating*float64(oldCount) + rating) / float64(newCount)

	_, err = tx.ExecContext(ctx, `
		UPDATE strategy_marketplace SET rating = ?, rating_count = ? WHERE id = ?
	`, newRating, newCount, id)
	if err != nil {
		return nil, fmt.Errorf("rate strategy update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("rate strategy commit: %w", err)
	}

	return &Strategy{
		ID:          id,
		Rating:      newRating,
		RatingCount: newCount,
	}, nil
}
