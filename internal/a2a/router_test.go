package a2a

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func TestAuthMiddleware_ValidKey(t *testing.T) {
	store := NewAPIKeyStore()
	key, _ := store.GenerateKey("test-owner")

	mw := AuthMiddleware(store)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	store := NewAPIKeyStore()
	store.GenerateKey("test-owner")

	mw := AuthMiddleware(store)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_MissingKey(t *testing.T) {
	store := NewAPIKeyStore()
	store.GenerateKey("test-owner")

	mw := AuthMiddleware(store)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_NilStore(t *testing.T) {
	mw := AuthMiddleware(nil)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRateLimitMiddleware_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)

	mw := RateLimitMiddleware(rl)
	handler := mw(http.HandlerFunc(okHandler))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "test-key")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}
}

func TestRateLimitMiddleware_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Minute)

	mw := RateLimitMiddleware(rl)
	handler := mw(http.HandlerFunc(okHandler))

	// First request — allowed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second request — should be 429
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-API-Key", "test-key")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitMiddleware_RateLimitHeader(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Minute)

	mw := RateLimitMiddleware(rl)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining != "2" {
		t.Errorf("X-RateLimit-Remaining = %q, want %q", remaining, "2")
	}
}

func TestRateLimitMiddleware_IPFallback(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Minute)

	mw := RateLimitMiddleware(rl)
	handler := mw(http.HandlerFunc(okHandler))

	// Request without X-API-Key — should use IP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second request from same IP — should be 429
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitMiddleware_NilLimiter(t *testing.T) {
	mw := RateLimitMiddleware(nil)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWithMiddleware_AuthAndRateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	store := NewAPIKeyStore()
	key, _ := store.GenerateKey("test-owner")
	rl := NewRateLimiter(10, 1*time.Minute)

	handler := withMiddleware(http.HandlerFunc(okHandler), logger, store, rl)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining == "" {
		t.Error("X-RateLimit-Remaining header is missing")
	}
}

func TestWithMiddleware_AuthOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	store := NewAPIKeyStore()
	key, _ := store.GenerateKey("test-owner")

	handler := withMiddleware(http.HandlerFunc(okHandler), logger, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWithMiddleware_NoAuthOrRateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	handler := withMiddleware(http.HandlerFunc(okHandler), logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
