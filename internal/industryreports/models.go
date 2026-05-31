package industryreports

// IndustryReport represents a stored industry research report.
type IndustryReport struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}
