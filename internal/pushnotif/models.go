package pushnotif

// Device represents a registered mobile device for push notifications.
type Device struct {
	ID        int    `json:"id"`
	Token     string `json:"token"`
	Platform  string `json:"platform"`
	CreatedAt string `json:"created_at"`
}

// PushResult represents the outcome of a simulated push notification send.
type PushResult struct {
	DeviceID int    `json:"device_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Status   string `json:"status"`
}
