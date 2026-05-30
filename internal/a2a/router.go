package a2a

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/history"
	"github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/pricing"
)

// NewRouter creates an http.Handler with all A2A routes registered.
func NewRouter(pricingStore *pricing.Store, historyStore *history.Store, mandateStore *MandateStore, logger *slog.Logger, baseURL string) http.Handler {
	handler := NewA2AHandler(pricingStore, historyStore, mandateStore, logger, baseURL)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /a2a/task", handler.HandleTask)
	mux.HandleFunc("GET /a2a/task/{id}", handler.HandleGetTask)
	mux.HandleFunc("POST /a2a/negotiate", handler.HandleNegotiate)
	mux.HandleFunc("GET /.well-known/agent-card.json", handler.HandleAgentCard)

	// Apply middleware chain
	return withMiddleware(mux, logger)
}

// withMiddleware wraps a handler with CORS, logging, and correlation ID.
func withMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Correlation ID
		cid := r.Header.Get("X-Correlation-ID")
		if cid == "" {
			cid = generateID()
		}
		w.Header().Set("X-Correlation-ID", cid)

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Correlation-ID, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Request logging
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		logger.Info("a2a request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"correlation_id", cid,
		)
	})
}

// loggingResponseWriter captures the status code for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func generateID() string {
	return "a2a-" + time.Now().Format("150405") + "-" + randomSuffix()
}

func randomSuffix() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1) // ensure different values
	}
	return string(b)
}
