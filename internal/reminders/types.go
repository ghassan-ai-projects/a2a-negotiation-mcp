package reminders

// RenewalCheckResult groups renewal items by urgency.
type RenewalCheckResult struct {
	Critical []RenewalItem `json:"critical"`
	Soon     []RenewalItem `json:"soon"`
	Upcoming []RenewalItem `json:"upcoming"`
}

// RenewalItem describes a single contract renewal.
type RenewalItem struct {
	ContractID    string `json:"contract_id"`
	Vendor        string `json:"vendor"`
	SKU           string `json:"sku"`
	RenewalDate   string `json:"renewal_date"`
	DaysUntil     int    `json:"days_until"`
	AutoNegotiate bool   `json:"auto_negotiate"`
	Notified      bool   `json:"notified"`
}
