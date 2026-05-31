package ratelimitdashboard

// Color represents the rate limit status color.
type Color string

const (
	ColorGreen  Color = "green"
	ColorYellow Color = "yellow"
	ColorRed    Color = "red"
)

// RateLimitStatus represents the current rate limit usage.
type RateLimitStatus struct {
	RateLimitConfig    PerSecond `json:"rate_limit_config"`
	RequestsThisMinute int       `json:"requests_this_minute"`
	RequestsThisHour   int       `json:"requests_this_hour"`
	RequestsToday      int       `json:"requests_today"`
	RemainingBudget    int       `json:"remaining_budget"`
	Status             Color     `json:"status"`
}

// PerSecond represents the rate limit per second.
type PerSecond struct {
	PerSecond int `json:"per_second"`
}

// APIUsageEntry is a single API usage log record.
type APIUsageEntry struct {
	ID        int64  `json:"id"`
	APIKeyID  string `json:"api_key_id"`
	Endpoint  string `json:"endpoint"`
	Timestamp string `json:"timestamp"`
}
