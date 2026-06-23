// Package circuitbreaker implements the 3-layer resilience pattern for NexusAI Gateway
// 
// Layer 1: Circuit Breaker (provider level) - CLOSED/HALF_OPEN/OPEN states
// Layer 2: Connection Cooldown (per account/key) - rateLimitUntil
// Layer 3: Model Lockout (provider+model) - granular lockout
//
// This implementation is thread-safe and suitable for high-throughput gateway use.
package circuitbreaker

import (
	"context"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed - Normal operation, requests pass through
	StateClosed CircuitState = iota
	// StateHalfOpen - Testing if provider recovered
	StateHalfOpen
	// StateOpen - Failing fast, requests blocked
	StateOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open before closing
	SuccessThreshold int
	// OpenDuration is how long the circuit stays open
	OpenDuration time.Duration
	// HalfOpenMaxCalls is max calls allowed in half-open state
	HalfOpenMaxCalls int
}

// DefaultConfig returns sensible defaults for a gateway
func DefaultConfig() Config {
	return Config{
		FailureThreshold:  5,
		SuccessThreshold: 3,
		OpenDuration:     30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// Circuit represents a single provider's circuit breaker
type Circuit struct {
	mu sync.RWMutex
	provider string
	
	state            CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time
	
	config Config
	
	// Callbacks for state changes (optional)
	onStateChange func(provider string, from, to CircuitState)
}

// NewCircuit creates a new circuit breaker for a provider
func NewCircuit(provider string, config Config) *Circuit {
	return &Circuit{
		provider:        provider,
		state:           StateClosed,
		config:          config,
		lastStateChange: time.Now(),
	}
}

// Provider returns the provider name
func (c *Circuit) Provider() string {
	return c.provider
}

// State returns the current circuit state
func (c *Circuit) State() CircuitState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Allow checks if a request should be allowed
func (c *Circuit) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	switch c.state {
	case StateClosed:
		return true
		
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(c.lastFailureTime) >= c.config.OpenDuration {
			c.transitionTo(StateHalfOpen)
			return true
		}
		return false
		
	case StateHalfOpen:
		// Allow limited calls in half-open
		if c.successes+c.failures < c.config.HalfOpenMaxCalls {
			return true
		}
		return false
		
	default:
		return false
	}
}

// RecordSuccess records a successful call
func (c *Circuit) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	switch c.state {
	case StateClosed:
		c.failures = 0 // Reset on success
		c.successes++   // Count successes even in closed state
	
	case StateHalfOpen:
		c.successes++
		if c.successes >= c.config.SuccessThreshold {
			c.transitionTo(StateClosed)
		}
	}
}

// RecordFailure records a failed call
func (c *Circuit) RecordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.lastFailureTime = time.Now()
	
	switch c.state {
	case StateClosed:
		c.failures++
		c.successes = 0
		if c.failures >= c.config.FailureThreshold {
			c.transitionTo(StateOpen)
		}
		
	case StateHalfOpen:
		c.failures++
		// Any failure in half-open opens the circuit
		c.transitionTo(StateOpen)
	}
}

// transitionTo changes state and fires callback
func (c *Circuit) transitionTo(newState CircuitState) {
	if c.state == newState {
		return
	}
	
	oldState := c.state
	c.state = newState
	c.lastStateChange = time.Now()
	
	// Reset counters on state change
	switch newState {
	case StateClosed:
		c.failures = 0
		c.successes = 0
	case StateHalfOpen:
		c.successes = 0
		c.failures = 0
	case StateOpen:
		// Keep lastFailureTime
	}
	
	// Fire callback if set
	if c.onStateChange != nil {
		go c.onStateChange(c.provider, oldState, newState)
	}
}

// Stats returns current statistics
type Stats struct {
	Provider       string        `json:"provider"`
	State          string       `json:"state"`
	Failures       int          `json:"failures"`
	Successes      int          `json:"successes"`
	LastFailure    time.Time    `json:"lastFailure"`
	LastStateChange time.Time   `json:"lastStateChange"`
	TimeInState    time.Duration `json:"timeInState"`
}

// Stats returns circuit statistics
func (c *Circuit) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return Stats{
		Provider:        c.provider,
		State:           c.state.String(),
		Failures:        c.failures,
		Successes:       c.successes,
		LastFailure:     c.lastFailureTime,
		LastStateChange: c.lastStateChange,
		TimeInState:     time.Since(c.lastStateChange),
	}
}

// Reset manually resets the circuit to closed state
func (c *Circuit) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.transitionTo(StateClosed)
}

// Manager manages multiple circuit breakers
type Manager struct {
	mu       sync.RWMutex
	circuits map[string]*Circuit
	config   Config
	
	// PostgreSQL persistence
	persistFunc func(ctx context.Context, stats []Stats) error
}

// NewManager creates a circuit breaker manager
func NewManager(config Config) *Manager {
	return &Manager{
		circuits: make(map[string]*Circuit),
		config:   config,
	}
}

// GetOrCreate gets or creates a circuit for a provider
func (m *Manager) GetOrCreate(provider string) *Circuit {
	m.mu.RLock()
	circuit, exists := m.circuits[provider]
	m.mu.RUnlock()
	
	if exists {
		return circuit
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check after acquiring write lock
	if circuit, exists = m.circuits[provider]; exists {
		return circuit
	}
	
	circuit = NewCircuit(provider, m.config)
	m.circuits[provider] = circuit
	return circuit
}

// Get returns an existing circuit or nil
func (m *Manager) Get(provider string) *Circuit {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.circuits[provider]
}

// List returns all circuits
func (m *Manager) List() []*Circuit {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*Circuit, 0, len(m.circuits))
	for _, c := range m.circuits {
		result = append(result, c)
	}
	return result
}

// SetPersistenceFunc sets the function to persist stats to PostgreSQL
func (m *Manager) SetPersistenceFunc(f func(ctx context.Context, stats []Stats) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistFunc = f
}

// PersistStats saves all circuit stats to PostgreSQL
func (m *Manager) PersistStats(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.persistFunc == nil {
		return nil
	}
	
	stats := make([]Stats, 0, len(m.circuits))
	for _, c := range m.circuits {
		stats = append(stats, c.Stats())
	}
	
	return m.persistFunc(ctx, stats)
}
