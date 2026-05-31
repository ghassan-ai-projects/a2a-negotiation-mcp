package serverconfig

// ConfigEntry represents a single server configuration key-value pair.
type ConfigEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}
