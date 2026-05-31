package metrics

// MetricsPayload holds the generated Prometheus exposition-format output.
type MetricsPayload struct {
	Content string `json:"content"`
}

// MetricLine represents a single Prometheus metric line with optional labels.
type MetricLine struct {
	Name   string            `json:"name"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
}
