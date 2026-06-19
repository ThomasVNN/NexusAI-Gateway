package middleware

import (
	"net/http"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerMiddleware implements the circuit breaker pattern
type CircuitBreakerMiddleware struct {
	mu           sync.RWMutex
	failureCount int
	successCount int
	requestCount int
	lastFailure  time.Time
	state        CircuitState
	threshold    int
	timeout      time.Duration
}

// NewCircuitBreakerMiddleware creates a new circuit breaker middleware
func NewCircuitBreakerMiddleware(threshold int, timeout time.Duration) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		threshold: threshold,
		timeout:   timeout,
		state:     CircuitClosed,
	}
}

// NewCircuitBreakerMiddlewareWithTimeout creates a circuit breaker with request timeout
func NewCircuitBreakerMiddlewareWithTimeout(threshold int, cbTimeout, reqTimeout time.Duration) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		threshold: threshold,
		timeout:   cbTimeout,
		state:     CircuitClosed,
	}
}

// Middleware returns the HTTP middleware function
func (m *CircuitBreakerMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		state := m.state
		m.mu.RUnlock()

		// Check if circuit is open
		if state == CircuitOpen {
			// Check if timeout has passed
			m.mu.RLock()
			timeSinceFailure := time.Since(m.lastFailure)
			m.mu.RUnlock()

			if timeSinceFailure > m.timeout {
				m.mu.Lock()
				m.state = CircuitHalfOpen
				m.mu.Unlock()
			} else {
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
		}

		// Execute request
		// Create a response writer that captures status
		rw := &statusCapturingWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		// Record result
		m.mu.Lock()
		m.requestCount++

		if rw.statusCode >= 500 {
			m.failureCount++
			m.lastFailure = time.Now()

			if m.failureCount >= m.threshold {
				m.state = CircuitOpen
			}
		} else {
			m.successCount++

			if m.state == CircuitHalfOpen {
				// Success in half-open state closes the circuit
				m.state = CircuitClosed
				m.failureCount = 0
			}
		}
		m.mu.Unlock()
	})
}

// GetCircuitState returns the current circuit state
func (m *CircuitBreakerMiddleware) GetCircuitState() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.String()
}

// GetStats returns circuit breaker statistics
func (m *CircuitBreakerMiddleware) GetStats() CircuitStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return CircuitStats{
		State:       m.state.String(),
		Requests:    m.requestCount,
		Failures:    m.failureCount,
		Successes:   m.successCount,
		LastFailure: m.lastFailure,
	}
}

// Reset resets the circuit breaker to initial state
func (m *CircuitBreakerMiddleware) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = CircuitClosed
	m.failureCount = 0
	m.successCount = 0
	m.requestCount = 0
}

// CircuitStats holds statistics about the circuit breaker
type CircuitStats struct {
	State       string
	Requests    int
	Failures    int
	Successes   int
	LastFailure time.Time
}

// statusCapturingWriter captures the response status code
type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
