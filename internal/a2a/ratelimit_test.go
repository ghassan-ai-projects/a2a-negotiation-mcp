package a2a

import (
	"testing"
	"time"
)

func TestRateLimiter_FirstRequestAllowed(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)

	allowed, remaining := rl.Allow("test-key")
	if !allowed {
		t.Fatal("first request should be allowed")
	}
	if remaining != 4 {
		t.Errorf("remaining = %d, want 4", remaining)
	}
}

func TestRateLimiter_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Minute)

	for i := 0; i < 3; i++ {
		allowed, remaining := rl.Allow("test-key")
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		expected := 2 - i
		if remaining != expected {
			t.Errorf("request %d: remaining = %d, want %d", i+1, remaining, expected)
		}
	}
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Minute)

	rl.Allow("test-key") // 1st
	rl.Allow("test-key") // 2nd

	allowed, remaining := rl.Allow("test-key") // 3rd — should be denied
	if allowed {
		t.Fatal("request should be denied after limit exceeded")
	}
	if remaining != 0 {
		t.Errorf("remaining = %d, want 0", remaining)
	}
}

func TestRateLimiter_ResetAfterWindow(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)

	allowed, _ := rl.Allow("test-key")
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	// Should be denied within the window
	allowed, _ = rl.Allow("test-key")
	if allowed {
		t.Fatal("second request should be denied within window")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	allowed, _ = rl.Allow("test-key")
	if !allowed {
		t.Fatal("request should be allowed after window reset")
	}
}

func TestRateLimiter_IsolatedKeys(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Minute)

	rl.Allow("key-a")
	allowed, _ := rl.Allow("key-a")
	if allowed {
		t.Fatal("key-a should be rate limited")
	}

	allowed, _ = rl.Allow("key-b")
	if !allowed {
		t.Fatal("key-b should be allowed (different key)")
	}
}
