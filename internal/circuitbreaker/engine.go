package circuitbreaker

import (
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// CircuitBreaker represents a circuit breaker instance
type CircuitBreaker struct {
	mu sync.RWMutex

	name            string
	provider        string
	state           State
	failureCount    int
	successCount    int
	failureLimit    int
	successLimit    int
	timeout         time.Duration
	lastFailure     time.Time
	lastStateChange time.Time

	// Statistics
	totalRequests  int64
	totalFailures  int64
	totalSuccesses int64
	totalTimeouts  int64
}

// CircuitBreakerConfig contains configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name         string
	Provider     string
	FailureLimit int
	SuccessLimit int
	Timeout      time.Duration
}

// New creates a new circuit breaker with the given config
func New(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureLimit == 0 {
		cfg.FailureLimit = 5
	}
	if cfg.SuccessLimit == 0 {
		cfg.SuccessLimit = 3
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &CircuitBreaker{
		name:         cfg.Name,
		provider:     cfg.Provider,
		state:        StateClosed,
		failureLimit: cfg.FailureLimit,
		successLimit: cfg.SuccessLimit,
		timeout:      cfg.Timeout,
	}
}

// Allow checks if a request should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastStateChange) > cb.timeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false

	case StateHalfOpen:
		return true
	}

	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalSuccesses++
	cb.failureCount = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successLimit {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		// Reset failure count on success
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalFailures++
	cb.lastFailure = time.Now()
	cb.successCount = 0

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureLimit {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.transitionTo(StateOpen)
	}
}

// RecordTimeout records a timeout
func (cb *CircuitBreaker) RecordTimeout() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalTimeouts++
	cb.lastFailure = time.Now()
	cb.successCount = 0

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureLimit {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

// transitionTo changes the state and records the transition
func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.failureCount = 0
	cb.successCount = 0
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":              cb.name,
		"provider":          cb.provider,
		"state":             cb.state.String(),
		"failure_count":     cb.failureCount,
		"success_count":     cb.successCount,
		"failure_limit":     cb.failureLimit,
		"success_limit":     cb.successLimit,
		"timeout_seconds":   cb.timeout.Seconds(),
		"total_requests":    cb.totalRequests,
		"total_failures":    cb.totalFailures,
		"total_successes":   cb.totalSuccesses,
		"total_timeouts":    cb.totalTimeouts,
		"last_failure":      cb.lastFailure.Format(time.RFC3339),
		"last_state_change": cb.lastStateChange.Format(time.RFC3339),
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.transitionTo(StateClosed)
	cb.totalRequests = 0
	cb.totalFailures = 0
	cb.totalSuccesses = 0
	cb.totalTimeouts = 0
}

// Manager manages multiple circuit breakers
type Manager struct {
	mu           sync.RWMutex
	breakers     map[string]*CircuitBreaker
	failureLimit int
	successLimit int
	timeout      time.Duration
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers:     make(map[string]*CircuitBreaker),
		failureLimit: 5,
		successLimit: 3,
		timeout:      30 * time.Second,
	}
}

// GetOrCreate gets or creates a circuit breaker for a provider
func (m *Manager) GetOrCreate(provider string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.breakers[provider]; exists {
		return cb
	}

	cb := New(CircuitBreakerConfig{
		Name:         provider + "-circuit",
		Provider:     provider,
		FailureLimit: m.failureLimit,
		SuccessLimit: m.successLimit,
		Timeout:      m.timeout,
	})
	m.breakers[provider] = cb
	return cb
}

// Get returns a circuit breaker if it exists
func (m *Manager) Get(provider string) *CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.breakers[provider]
}

// List returns all circuit breakers
func (m *Manager) List() []*CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*CircuitBreaker, 0, len(m.breakers))
	for _, cb := range m.breakers {
		result = append(result, cb)
	}
	return result
}

// Reset resets a specific circuit breaker
func (m *Manager) Reset(provider string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.breakers[provider]; exists {
		cb.Reset()
		return true
	}
	return false
}

// ResetAll resets all circuit breakers
func (m *Manager) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cb := range m.breakers {
		cb.Reset()
	}
}

// Delete removes a circuit breaker
func (m *Manager) Delete(provider string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.breakers[provider]; exists {
		delete(m.breakers, provider)
		return true
	}
	return false
}

// GetAllStats returns statistics for all circuit breakers
func (m *Manager) GetAllStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breakers := make([]map[string]interface{}, 0, len(m.breakers))
	for _, cb := range m.breakers {
		breakers = append(breakers, cb.GetStats())
	}

	closed := 0
	open := 0
	halfOpen := 0

	for _, cb := range m.breakers {
		switch cb.GetState() {
		case StateClosed:
			closed++
		case StateOpen:
			open++
		case StateHalfOpen:
			halfOpen++
		}
	}

	return map[string]interface{}{
		"total_breakers": len(m.breakers),
		"closed":         closed,
		"open":           open,
		"half_open":      halfOpen,
		"breakers":       breakers,
	}
}

// Global circuit breaker manager
var globalManager *Manager
var managerOnce sync.Once

// GetManager returns the global circuit breaker manager
func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = NewManager()
	})
	return globalManager
}
