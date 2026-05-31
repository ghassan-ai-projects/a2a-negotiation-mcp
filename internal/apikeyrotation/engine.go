package apikeyrotation

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultExpiryDays = 90

// Engine manages API key lifecycle — creation, rotation, and health reporting.
type Engine struct {
	mu      sync.Mutex
	keys    []KeyHealthEntry
	counter int64
}

// NewEngine creates a new Engine with no pre-existing keys.
func NewEngine() *Engine {
	return &Engine{}
}

// AddKey creates a new API key for the given owner with a 90-day expiry.
// Returns the newly created KeyHealthEntry.
func (e *Engine) AddKey(owner string) (*KeyHealthEntry, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner must not be empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()
	keyID := uuid.New().String()
	e.counter++

	entry := KeyHealthEntry{
		KeyID:       keyID,
		Owner:       owner,
		Status:      "active",
		CreatedAt:   now.Format(time.RFC3339),
		ExpiresAt:   now.AddDate(0, 0, defaultExpiryDays).Format(time.RFC3339),
		LastRotated: now.Format(time.RFC3339),
	}
	e.keys = append(e.keys, entry)
	return &entry, nil
}

// RotateKey marks the existing key as revoked, generates a new UUID key,
// and creates a fresh KeyHealthEntry for the replacement.
func (e *Engine) RotateKey(keyID string) (*RotationResult, error) {
	if keyID == "" {
		return nil, fmt.Errorf("key_id must not be empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find the key and revoke it
	var found bool
	for i := range e.keys {
		if e.keys[i].KeyID == keyID {
			if e.keys[i].Status != "active" {
				return nil, fmt.Errorf("key %s is not active (status: %s)", keyID, e.keys[i].Status)
			}
			e.keys[i].Status = "revoked"
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("key %s not found", keyID)
	}

	// Create replacement key
	now := time.Now().UTC()
	newKeyID := uuid.New().String()
	e.counter++

	newEntry := KeyHealthEntry{
		KeyID:       newKeyID,
		Owner:       "", // caller can update owner via metadata if needed
		Status:      "active",
		CreatedAt:   now.Format(time.RFC3339),
		ExpiresAt:   now.AddDate(0, 0, defaultExpiryDays).Format(time.RFC3339),
		LastRotated: now.Format(time.RFC3339),
	}
	e.keys = append(e.keys, newEntry)

	return &RotationResult{
		OldKeyID: keyID,
		NewKeyID: newKeyID,
		Status:   "rotated",
	}, nil
}

// KeyHealth returns all API keys with their current health status.
func (e *Engine) KeyHealth() ([]KeyHealthEntry, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Return a copy to avoid race conditions on the caller side
	result := make([]KeyHealthEntry, len(e.keys))
	copy(result, e.keys)
	return result, nil
}
