package ipwhitelist

// WhitelistEntry represents a single IP address on the whitelist.
type WhitelistEntry struct {
	IP        string `json:"ip"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at"`
}
