package qrshare

// QRRequest represents a request to generate a QR code for a session.
type QRRequest struct {
	SessionID string `json:"session_id"`
}

// QRResult represents the result of a QR code generation.
type QRResult struct {
	SessionID   string `json:"session_id"`
	QRData      string `json:"qr_data"`      // base64-encoded simulated QR code
	Format      string `json:"format"`
	Description string `json:"description"`
}
