package dashboard

// Widget represents a single dashboard widget configuration.
type Widget struct {
	ID        int    `json:"id"`
	WidgetType string `json:"widget_type"`
	Title     string `json:"title"`
	Config    string `json:"config"`
	CreatedAt string `json:"created_at"`
}

// Dashboard represents a collection of widgets with a count.
type Dashboard struct {
	Widgets []Widget `json:"widgets"`
	Count   int      `json:"count"`
}
