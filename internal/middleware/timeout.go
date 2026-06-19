package middleware

import (
	"context"
	"net/http"
	"time"
)

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// TimeoutMiddleware creates a timeout middleware
func TimeoutMiddleware(config TimeoutConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), config.ReadTimeout)
			defer cancel()

			// Replace request context
			r = r.WithContext(ctx)

			// Create timeout writer
			done := make(chan struct{})
			tw := &timeoutWriter{
				ResponseWriter: w,
				code:           http.StatusOK,
				done:           done,
			}

			// Start timeout handler
			go func() {
				next.ServeHTTP(tw, r)
				close(done)
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed
				if !tw.wroteHeader {
					tw.WriteHeader(tw.code)
				}
			case <-ctx.Done():
				// Timeout
				tw.WriteHeader(http.StatusGatewayTimeout)
			}
		})
	}
}

// timeoutWriter wraps http.ResponseWriter with timeout handling
type timeoutWriter struct {
	http.ResponseWriter
	code        int
	wroteHeader bool
	done        chan struct{}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	if !tw.wroteHeader {
		tw.code = code
		tw.wroteHeader = true
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	if !tw.wroteHeader {
		tw.WriteHeader(tw.code)
	}
	return tw.ResponseWriter.Write(b)
}

// WithTimeout creates a timeout middleware with a specific duration
func WithTimeout(duration time.Duration) func(http.Handler) http.Handler {
	return TimeoutMiddleware(TimeoutConfig{
		ReadTimeout:  duration,
		WriteTimeout: duration,
		IdleTimeout:  duration * 2,
	})
}

// TimeoutHandler creates a handler that executes with a timeout
func TimeoutHandler(timeout time.Duration, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		done := make(chan error, 1)

		go func() {
			handler.ServeHTTP(w, r)
			done <- nil
		}()

		select {
		case <-ctx.Done():
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		case <-done:
			// Completed normally
		}
	})
}

// ReadTimeoutMiddleware adds read timeout handling
func ReadTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WriteTimeoutMiddleware adds write timeout handling
func WriteTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-time.After(timeout):
				// Timeout on write
				return
			}
		})
	}
}
