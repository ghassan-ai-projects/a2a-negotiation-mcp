package apikeyrotation

// RotationResult represents the result of rotating an API key.
type RotationResult struct {
	OldKeyID string `json:"old_key_id"`
	NewKeyID string `json:"new_key_id"`
	Status   string `json:"status"`
}

// KeyHealthEntry represents the health status of a single API key.
type KeyHealthEntry struct {
	KeyID       string `json:"key_id"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at"`
	LastRotated string `json:"last_rotated"`
}
