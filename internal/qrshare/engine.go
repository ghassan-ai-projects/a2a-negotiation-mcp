package qrshare

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Engine is a stateless engine for generating simulated QR codes.
type Engine struct{}

// NewEngine creates a new Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// GenerateQR generates a simulated QR code for the given session ID.
// The QR data is the base64 encoding of the session ID itself (a mock).
func (e *Engine) GenerateQR(ctx context.Context, sessionID string) (*QRResult, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID must not be empty")
	}

	qrData := base64.StdEncoding.EncodeToString([]byte(sessionID))

	return &QRResult{
		SessionID:   sessionID,
		QRData:      qrData,
		Format:      "png",
		Description: fmt.Sprintf("QR code for session %s", sessionID),
	}, nil
}
