package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Store manages dashboard widgets in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a DashboardStore using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate dashboard: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS dashboard_widgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		widget_type TEXT NOT NULL,
		title TEXT NOT NULL,
		config TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateWidget saves a new dashboard widget and returns it with the generated ID and timestamp.
func (s *Store) CreateWidget(ctx context.Context, widgetType, title, config string) (*Widget, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_widgets (widget_type, title, config, created_at)
		VALUES (?, ?, ?, ?)
	`, widgetType, title, config, now)
	if err != nil {
		return nil, fmt.Errorf("create widget: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create widget last insert id: %w", err)
	}
	return &Widget{
		ID:         int(id),
		WidgetType: widgetType,
		Title:      title,
		Config:     config,
		CreatedAt:  now,
	}, nil
}

// ListWidgets returns all dashboard widgets ordered by created_at DESC.
func (s *Store) ListWidgets(ctx context.Context) ([]Widget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, widget_type, title, config, created_at
		FROM dashboard_widgets
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}
	defer rows.Close()

	var widgets []Widget
	for rows.Next() {
		var w Widget
		if err := rows.Scan(&w.ID, &w.WidgetType, &w.Title, &w.Config, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan widget: %w", err)
		}
		widgets = append(widgets, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if widgets == nil {
		widgets = []Widget{}
	}
	return widgets, nil
}

// RenderDashboard returns a Dashboard containing only the widgets matching the given IDs.
func (s *Store) RenderDashboard(ctx context.Context, widgetIDs []int) (*Dashboard, error) {
	if len(widgetIDs) == 0 {
		return &Dashboard{Widgets: []Widget{}, Count: 0}, nil
	}

	placeholders := make([]string, len(widgetIDs))
	args := make([]any, len(widgetIDs))
	for i, id := range widgetIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, widget_type, title, config, created_at
		FROM dashboard_widgets
		WHERE id IN (%s)
		ORDER BY created_at DESC
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("render dashboard: %w", err)
	}
	defer rows.Close()

	var widgets []Widget
	for rows.Next() {
		var w Widget
		if err := rows.Scan(&w.ID, &w.WidgetType, &w.Title, &w.Config, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan widget: %w", err)
		}
		widgets = append(widgets, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if widgets == nil {
		widgets = []Widget{}
	}
	return &Dashboard{Widgets: widgets, Count: len(widgets)}, nil
}

// ExportDashboard returns a JSON string of all widgets. If format="json", it returns pretty-printed JSON.
func (s *Store) ExportDashboard(ctx context.Context, format string) (string, error) {
	widgets, err := s.ListWidgets(ctx)
	if err != nil {
		return "", fmt.Errorf("export dashboard: %w", err)
	}

	var b []byte
	if format == "json" {
		b, err = json.MarshalIndent(widgets, "", "  ")
	} else {
		b, err = json.Marshal(widgets)
	}
	if err != nil {
		return "", fmt.Errorf("export dashboard marshal: %w", err)
	}
	return string(b), nil
}
