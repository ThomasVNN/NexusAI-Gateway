package retry

import (
	"log/slog"
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

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failureCount     int
	successCount     int
	failureThreshold int           // Number of failures before opening
	successThreshold int           // Number of successes before closing
	timeout          time.Duration // Time before transitioning from open to half-open
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Allow checks if a request should be allowed through
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return nil

	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			slog.Info("Circuit breaker transitioning to half-open")
			return nil
		}
		return ErrCircuitOpen

	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		return nil
	}

	return nil
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0

	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.successCount = 0
			slog.Info("Circuit breaker closed after successful requests")
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitOpen
			slog.Warn("Circuit breaker opened",
				slog.Int("failure_count", cb.failureCount),
			)
		}

	case CircuitHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.state = CircuitOpen
		cb.failureCount = 1
		slog.Warn("Circuit breaker reopened after half-open failure")
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}
}

// Stats returns current circuit breaker statistics
type CircuitBreakerStats struct {
	State            string    `json:"state"`
	FailureCount     int       `json:"failure_count"`
	SuccessCount     int       `json:"success_count"`
	LastFailure      time.Time `json:"last_failure"`
	FailureThreshold int       `json:"failure_threshold"`
	SuccessThreshold int       `json:"success_threshold"`
}

// Stats returns current statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:            cb.state.String(),
		FailureCount:     cb.failureCount,
		SuccessCount:     cb.successCount,
		LastFailure:      cb.lastFailureTime,
		FailureThreshold: cb.failureThreshold,
		SuccessThreshold: cb.successThreshold,
	}
}
