package ipwhitelist

import (
	"fmt"
	"sync"
	"time"
)

// Engine manages an in-memory IP whitelist.
type Engine struct {
	mu    sync.Mutex
	items map[string]WhitelistEntry
}

// NewEngine creates a new Engine with no entries.
func NewEngine() *Engine {
	return &Engine{
		items: make(map[string]WhitelistEntry),
	}
}

// AddIP adds an IP address to the whitelist. Returns an error if the IP already exists.
func (e *Engine) AddIP(ip, label string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.items[ip]; exists {
		return fmt.Errorf("IP %s already exists in whitelist", ip)
	}

	e.items[ip] = WhitelistEntry{
		IP:        ip,
		Label:     label,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return nil
}

// RemoveIP removes an IP address from the whitelist. Returns an error if not found.
func (e *Engine) RemoveIP(ip string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.items[ip]; !exists {
		return fmt.Errorf("IP %s not found in whitelist", ip)
	}

	delete(e.items, ip)
	return nil
}

// List returns all whitelist entries.
func (e *Engine) List() []WhitelistEntry {
	e.mu.Lock()
	defer e.mu.Unlock()

	entries := make([]WhitelistEntry, 0, len(e.items))
	for _, v := range e.items {
		entries = append(entries, v)
	}
	return entries
}
