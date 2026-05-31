package pricechart

// ChartData holds monthly price chart data and a summary.
type ChartData struct {
	Labels   []string        `json:"labels"`   // monthly labels "2026-01", "2026-02"
	Datasets []ChartDataset  `json:"datasets"`
	Summary  ChartSummary    `json:"summary"`
}

// ChartDataset is a single series in the chart.
type ChartDataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
	Color string    `json:"border_color"`
}

// ChartSummary provides aggregate metrics.
type ChartSummary struct {
	AvgPrice     float64 `json:"avg_price"`
	TotalSavings float64 `json:"total_savings"`
	BestDeal     float64 `json:"best_deal"`
	WorstDeal    float64 `json:"worst_deal"`
}
