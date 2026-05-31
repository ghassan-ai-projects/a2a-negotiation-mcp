package chartexport

// ChartRequest represents an export chart request.
type ChartRequest struct {
	DataSource string `json:"data_source"`
	ChartType  string `json:"chart_type"`
	Format     string `json:"format"`
}

// ChartResult holds the generated chart output.
type ChartResult struct {
	Format    string `json:"format"`
	ChartType string `json:"chart_type"`
	Data      string `json:"data"` // simulated SVG/PNG data as string
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	MimeType  string `json:"mime_type"`
}

// ChartTemplate describes a predefined chart template.
type ChartTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ChartType   string `json:"chart_type"`
	ColorScheme string `json:"color_scheme"`
}
