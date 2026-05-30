package a2a

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

// APIKeyStore manages API keys for authentication.
type APIKeyStore struct {
	mu       sync.RWMutex
	keys     map[string]string // key -> owner
	filePath string            // optional file for persistence
}

// NewAPIKeyStore creates a new empty APIKeyStore.
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		keys: make(map[string]string),
	}
}

// GenerateKey creates a new UUID-based API key, stores it, and returns it.
func (s *APIKeyStore) GenerateKey(owner string) (string, error) {
	if owner == "" {
		return "", fmt.Errorf("owner must not be empty")
	}

	key := uuid.New().String()

	s.mu.Lock()
	s.keys[key] = owner
	s.mu.Unlock()

	return key, nil
}

// ValidateKey checks if a key exists and returns the owner.
func (s *APIKeyStore) ValidateKey(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	owner, ok := s.keys[key]
	return owner, ok
}

// LoadFromFile loads keys from a JSON file.
// The file format is: {"key1": "owner1", "key2": "owner2"}
// KeyCount returns the number of stored API keys.
func (s *APIKeyStore) KeyCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.keys)
}

func (s *APIKeyStore) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read key file: %w", err)
	}

	var loaded map[string]string
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse key file: %w", err)
	}

	s.mu.Lock()
	s.filePath = path
	for k, v := range loaded {
		s.keys[k] = v
	}
	s.mu.Unlock()

	return nil
}

// SaveToFile persists keys to the configured JSON file.
func (s *APIKeyStore) SaveToFile() error {
	s.mu.RLock()
	filePath := s.filePath
	data := s.keys
	s.mu.RUnlock()

	if filePath == "" {
		return fmt.Errorf("no file path configured, use LoadFromFile first")
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keys: %w", err)
	}

	if err := os.WriteFile(filePath, b, 0644); err != nil {
		return fmt.Errorf("write key file: %w", err)
	}

	return nil
}
