package pushnotif

import (
	"context"
)

// Engine simulates sending push notifications to mobile devices.
type Engine struct{}

// NewEngine creates a new push notification engine.
func NewEngine() *Engine {
	return &Engine{}
}

// SendPush simulates sending a push notification to a device, returning status "sent".
func (e *Engine) SendPush(ctx context.Context, deviceID int, title, body string) (*PushResult, error) {
	return &PushResult{
		DeviceID: deviceID,
		Title:    title,
		Body:     body,
		Status:   "sent",
	}, nil
}
