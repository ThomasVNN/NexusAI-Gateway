package router

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/observability"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/quota"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// Middleware functions that wrap handlers with observability
var Middleware = struct {
	Tracing       func(http.Handler) http.Handler
	CorrelationID func(http.Handler) http.Handler
	StructuredLog func(http.Handler) http.Handler
	Recovery      func(http.Handler) http.Handler
	RateLimiting  func(http.Handler) http.Handler
	Metrics       func(http.Handler) http.Handler
}{
	Tracing:       WithTracing,
	CorrelationID: WithCorrelationID,
	StructuredLog: WithStructuredLogging,
	Recovery:      WithRecovery,
	RateLimiting:  WithRateLimiting,
	Metrics:       WithTracing, // Tracing includes metrics
}

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

// GetTraceID extracts the OpenTelemetry trace ID from context
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return observability.GetTraceID(ctx)
}

// GetSpanID extracts the OpenTelemetry span ID from context
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return observability.GetSpanID(ctx)
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

// WithRecovery recover from panics gracefully, captures stack trace, and writes structural error payload
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

// WithCORS adds Cross-Origin Resource Sharing headers based on configured origins
func WithCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is in allowed list
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" {
					allowed = true
					break
				}
				if allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WithSecurityHeaders adds standard security headers to all HTTP responses
func WithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable browser XSS filtering
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information leakage
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy for API responses
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Strict Transport Security (HSTS) - only in production
		// Note: Enable after verifying HTTPS is working correctly
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Cache control for sensitive API responses
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")

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

// WithTracing wraps HTTP handlers with OpenTelemetry tracing
func WithTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		corrID := GetCorrelationID(r.Context())
		traceID := GetTraceID(r.Context())
		spanID := GetSpanID(r.Context())

		wrappedWriter := &responseWriterWrapper{ResponseWriter: w}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(startTime)

		// Record metrics if available
		m := observability.GetGlobalMetrics()
		if m != nil {
			m.ObserveRequest(r.Method, r.URL.Path, wrappedWriter.statusCode, duration)
		}

		// Log with full trace context
		slog.InfoContext(r.Context(), "HTTP request completed",
			slog.String("service", "nexusai-gateway"),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("correlation_id", corrID),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.Int("status_code", wrappedWriter.statusCode),
			slog.Int64("latency_ms", duration.Milliseconds()),
		)
	})
}

type ipLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

var globalLimiter = &ipLimiter{
	requests: make(map[string][]time.Time),
}

// WithRateLimiting enforces a secure sliding-window rate limit to protect system capacity.
// Falls back to IP-based limiting if tenant context is not available.
func WithRateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}

		// Get tenant from context if available
		tenant, err := tenancy.GetTenant(r.Context())
		if err == nil && tenant.IsActive {
			// Use tenant-aware rate limiting
			plan := tenant.Plan
			if plan == "" {
				plan = "standard"
			}
			tenantRateLimiter := quota.NewTenantRateLimiter(plan)

			allowed := tenantRateLimiter.Allow(tenant.ID)
			if !allowed {
				corrID := GetCorrelationID(r.Context())
				slog.WarnContext(r.Context(), "Tenant rate limit exceeded",
					slog.String("tenant_id", tenant.ID),
					slog.String("tenant_slug", tenant.Slug),
					slog.String("plan", plan),
					slog.String("correlation_id", corrID),
				)
				plans := quota.DefaultQuotaPlans()
				planLimits, _ := plans[plan]
				var rpm int = 60
				if planLimits != nil {
					rpm = planLimits.RequestsPerMin
				}
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rpm))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", tenantRateLimiter.GetRemaining(tenant.ID)))
				w.Header().Set("Retry-After", "60")
				handler.WriteError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
					fmt.Sprintf("Rate limit exceeded for tenant %s. Limit: %d/min. Retry after 60 seconds.",
						tenant.Slug, rpm))
				return
			}
		} else {
			// Fallback to IP-based rate limiting
			limitKey := ip
			authHeader := r.Header.Get("Authorization")
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
		}

		next.ServeHTTP(w, r)
	})
}
