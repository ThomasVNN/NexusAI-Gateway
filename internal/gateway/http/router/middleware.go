package router

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
)

type contextKey string

const CorrelationIDKey contextKey = "correlation_id"

// responseWriterWrapper wraps standard http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// generateCorrelationID generates a secure random correlation ID string
func generateCorrelationID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "fallback-correlation-id"
	}
	return hex.EncodeToString(bytes)
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return val
	}
	return ""
}

// WithCorrelationID injects a unique request tracer header if not present
func WithCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Header.Get("X-Correlation-ID")
		if corrID == "" {
			corrID = generateCorrelationID()
		}

		// Inject into context and headers
		ctx := context.WithValue(r.Context(), CorrelationIDKey, corrID)
		r.Header.Set("X-Correlation-ID", corrID)
		w.Header().Set("X-Correlation-ID", corrID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithStructuredLogging logs request endpoints, duration, and status codes in structured JSON format via slog
func WithStructuredLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		corrID := GetCorrelationID(r.Context())

		wrappedWriter := &responseWriterWrapper{ResponseWriter: w}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(startTime)

		// Log using standard structured slog JSON format
		slog.InfoContext(r.Context(), "HTTP request completed",
			slog.String("service", "nexusai-gateway"),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("correlation_id", corrID),
			slog.Int("status_code", wrappedWriter.statusCode),
			slog.Int64("latency_ms", duration.Milliseconds()),
		)
	})
}

// WithRecovery recovers from panics gracefully, captures stack trace, and writes structural error payload
func WithRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 2048)
				length := runtime.Stack(stack, false)
				corrID := GetCorrelationID(r.Context())

				slog.ErrorContext(r.Context(), "Panic recovered in HTTP handler",
					slog.Any("error", err),
					slog.String("stack", string(stack[:length])),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("correlation_id", corrID),
				)

				handler.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal Server Error: An unexpected panic occurred inside the server")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// WithTimeout applies a context timeout to the request
func WithTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type ipLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

var globalLimiter = &ipLimiter{
	requests: make(map[string][]time.Time),
}

// WithRateLimiting enforces a secure sliding-window rate limit to protect system capacity
func WithRateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}

		authHeader := r.Header.Get("Authorization")
		limitKey := ip
		if authHeader != "" {
			limitKey = authHeader
		}

		globalLimiter.mu.Lock()
		now := time.Now()
		window := 1 * time.Minute
		maxRequests := 300 // Generous production-ready fallback limit for gateway

		var activeRequests []time.Time
		for _, t := range globalLimiter.requests[limitKey] {
			if now.Sub(t) < window {
				activeRequests = append(activeRequests, t)
			}
		}

		if len(activeRequests) >= maxRequests {
			globalLimiter.mu.Unlock()
			corrID := GetCorrelationID(r.Context())

			slog.WarnContext(r.Context(), "Rate limit exceeded",
				slog.String("limit_key", limitKey),
				slog.String("ip", ip),
				slog.String("path", r.URL.Path),
				slog.String("correlation_id", corrID),
			)
			handler.WriteError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too Many Requests: Rate limit exceeded. Please try again later.")
			return
		}

		activeRequests = append(activeRequests, now)
		globalLimiter.requests[limitKey] = activeRequests
		globalLimiter.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
