package routing

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           // Failures before opening
	ResetTimeout     time.Duration // Time before half-open
	HalfOpenRequests int           // Successful requests to close
}

// DefaultOAuthConfig for OAuth providers
var DefaultOAuthConfig = CircuitBreakerConfig{
	FailureThreshold: 3,
	ResetTimeout:     60 * time.Second,
	HalfOpenRequests: 1,
}

// DefaultAPIKeyConfig for API key providers
var DefaultAPIKeyConfig = CircuitBreakerConfig{
	FailureThreshold: 5,
	ResetTimeout:     30 * time.Second,
	HalfOpenRequests: 2,
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu       sync.RWMutex
	provider map[string]*ProviderBreaker
	config   map[string]CircuitBreakerConfig
}

// ProviderBreaker tracks circuit state for a provider
type ProviderBreaker struct {
	State       CircuitBreakerState
	Failures    int
	Successes   int
	LastFailure time.Time
	OpenedAt    time.Time
	Config      CircuitBreakerConfig
}

// ConnectionBreaker tracks circuit state for a connection
type ConnectionBreaker struct {
	ID               string
	ProviderID       string
	RateLimitedUntil time.Time
	BackoffLevel     int
	LastError        string
	ErrorType        string
}

// ModelLockout tracks locked out models
type ModelLockout struct {
	mu       sync.RWMutex
	lockouts map[string]*LockoutEntry // key: "provider:model"
}

// LockoutEntry represents a single model lockout
type LockoutEntry struct {
	ProviderID string
	ModelID    string
	ExpiresAt  time.Time
	Reason     string
}

// NewCircuitBreaker creates a new circuit breaker manager
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		provider: make(map[string]*ProviderBreaker),
		config:   make(map[string]CircuitBreakerConfig),
	}
}

// RegisterProvider registers a provider with the circuit breaker
func (cb *CircuitBreaker) RegisterProvider(providerID string, config CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.provider[providerID] = &ProviderBreaker{
		State:  CircuitClosed,
		Config: config,
	}
	cb.config[providerID] = config
}

// RecordFailure records a failure for a provider
func (cb *CircuitBreaker) RecordFailure(providerID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	breaker, ok := cb.provider[providerID]
	if !ok {
		return
	}

	breaker.Failures++
	breaker.LastFailure = time.Now()

	if breaker.State == CircuitClosed && breaker.Failures >= breaker.Config.FailureThreshold {
		breaker.State = CircuitOpen
		breaker.OpenedAt = time.Now()
	}
}

// RecordSuccess records a success for a provider
func (cb *CircuitBreaker) RecordSuccess(providerID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	breaker, ok := cb.provider[providerID]
	if !ok {
		return
	}

	if breaker.State == CircuitHalfOpen {
		breaker.Successes++
		if breaker.Successes >= breaker.Config.HalfOpenRequests {
			breaker.State = CircuitClosed
			breaker.Failures = 0
			breaker.Successes = 0
		}
	} else if breaker.State == CircuitClosed {
		breaker.Failures = 0
	}
}

// CanExecute checks if requests can be sent to a provider
func (cb *CircuitBreaker) CanExecute(providerID string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	breaker, ok := cb.provider[providerID]
	if !ok {
		return true
	}

	switch breaker.State {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(breaker.OpenedAt) >= breaker.Config.ResetTimeout {
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return true
	}
}

// GetStatus returns the current status of a provider's circuit
func (cb *CircuitBreaker) GetStatus(providerID string) CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	breaker, ok := cb.provider[providerID]
	if !ok {
		return CircuitClosed
	}

	if breaker.State == CircuitOpen && time.Since(breaker.OpenedAt) >= breaker.Config.ResetTimeout {
		return CircuitHalfOpen
	}

	return breaker.State
}

// GetAllStatus returns the status of all providers
func (cb *CircuitBreaker) GetAllStatus() map[string]CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	status := make(map[string]CircuitBreakerState)
	for providerID, breaker := range cb.provider {
		state := breaker.State
		if state == CircuitOpen && time.Since(breaker.OpenedAt) >= breaker.Config.ResetTimeout {
			state = CircuitHalfOpen
		}
		status[providerID] = state
	}

	return status
}

// Reset resets a provider's circuit breaker
func (cb *CircuitBreaker) Reset(providerID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	breaker, ok := cb.provider[providerID]
	if !ok {
		return
	}

	breaker.State = CircuitClosed
	breaker.Failures = 0
	breaker.Successes = 0
}

// ConnectionCooldown manages connection-level cooldowns
type ConnectionCooldown struct {
	mu        sync.RWMutex
	cooldowns map[string]*ConnectionCooldownEntry
}

// ConnectionCooldownEntry represents a connection in cooldown
type ConnectionCooldownEntry struct {
	ID               string
	ProviderID       string
	RateLimitedUntil time.Time
	BackoffLevel     int
	LastError        string
	ErrorType        string
}

// NewConnectionCooldown creates a new connection cooldown manager
func NewConnectionCooldown() *ConnectionCooldown {
	return &ConnectionCooldown{
		cooldowns: make(map[string]*ConnectionCooldownEntry),
	}
}

// SetCooldown sets a cooldown for a connection
func (cc *ConnectionCooldown) SetCooldown(connectionID string, providerID string, duration time.Duration, errType string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	entry, ok := cc.cooldowns[connectionID]
	if !ok {
		entry = &ConnectionCooldownEntry{
			ID:           connectionID,
			ProviderID:   providerID,
			BackoffLevel: 0,
		}
		cc.cooldowns[connectionID] = entry
	}

	entry.RateLimitedUntil = time.Now().Add(duration * (1 << entry.BackoffLevel))
	entry.LastError = ""
	entry.ErrorType = errType

	if entry.BackoffLevel < 5 {
		entry.BackoffLevel++
	}
}

// IsCoolingDown checks if a connection is in cooldown
func (cc *ConnectionCooldown) IsCoolingDown(connectionID string) bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	entry, ok := cc.cooldowns[connectionID]
	if !ok {
		return false
	}

	return time.Now().Before(entry.RateLimitedUntil)
}

// ClearCooldown clears a connection's cooldown
func (cc *ConnectionCooldown) ClearCooldown(connectionID string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	delete(cc.cooldowns, connectionID)
}

// GetCooldownRemaining returns the remaining cooldown time for a connection
func (cc *ConnectionCooldown) GetCooldownRemaining(connectionID string) time.Duration {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	entry, ok := cc.cooldowns[connectionID]
	if !ok {
		return 0
	}

	remaining := time.Until(entry.RateLimitedUntil)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ModelLockoutManager manages model-level lockouts
type ModelLockoutManager struct {
	mu       sync.RWMutex
	lockouts map[string]*LockoutEntry // key: "providerID:modelID"
}

// NewModelLockoutManager creates a new lockout manager
func NewModelLockoutManager() *ModelLockoutManager {
	return &ModelLockoutManager{
		lockouts: make(map[string]*LockoutEntry),
	}
}

// LockModel locks a model
func (m *ModelLockoutManager) LockModel(providerID, modelID string, duration time.Duration, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := providerID + ":" + modelID
	m.lockouts[key] = &LockoutEntry{
		ProviderID: providerID,
		ModelID:    modelID,
		ExpiresAt:  time.Now().Add(duration),
		Reason:     reason,
	}
}

// IsLocked checks if a model is locked
func (m *ModelLockoutManager) IsLocked(providerID, modelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := providerID + ":" + modelID
	entry, ok := m.lockouts[key]
	if !ok {
		return false
	}

	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return true
}

// UnlockModel unlocks a model
func (m *ModelLockoutManager) UnlockModel(providerID, modelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := providerID + ":" + modelID
	delete(m.lockouts, key)
}

// GetAllLockouts returns all active lockouts
func (m *ModelLockoutManager) GetAllLockouts() []*LockoutEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	lockouts := make([]*LockoutEntry, 0)

	for _, entry := range m.lockouts {
		if now.Before(entry.ExpiresAt) {
			lockouts = append(lockouts, entry)
		}
	}

	return lockouts
}

// ResilienceManager orchestrates all resilience mechanisms
type ResilienceManager struct {
	CircuitBreaker     *CircuitBreaker
	ConnectionCooldown *ConnectionCooldown
	ModelLockout       *ModelLockoutManager
}

// NewResilienceManager creates a new resilience manager
func NewResilienceManager() *ResilienceManager {
	return &ResilienceManager{
		CircuitBreaker:     NewCircuitBreaker(),
		ConnectionCooldown: NewConnectionCooldown(),
		ModelLockout:       NewModelLockoutManager(),
	}
}

// CanExecute checks if a request can be executed considering all resilience mechanisms
func (rm *ResilienceManager) CanExecute(providerID, connectionID, modelID string) bool {
	if !rm.CircuitBreaker.CanExecute(providerID) {
		return false
	}

	if rm.ConnectionCooldown.IsCoolingDown(connectionID) {
		return false
	}

	if rm.ModelLockout.IsLocked(providerID, modelID) {
		return false
	}

	return true
}

// RecordFailure records a failure for a provider/connection/model
func (rm *ResilienceManager) RecordFailure(providerID, connectionID, modelID string, errType string) {
	rm.CircuitBreaker.RecordFailure(providerID)
	rm.ConnectionCooldown.SetCooldown(connectionID, providerID, 30*time.Second, errType)

	if errType == "quota_exceeded" || errType == "model_unavailable" {
		rm.ModelLockout.LockModel(providerID, modelID, 5*time.Minute, errType)
	}
}

// RecordSuccess records a success
func (rm *ResilienceManager) RecordSuccess(providerID, connectionID, modelID string) {
	rm.CircuitBreaker.RecordSuccess(providerID)
	rm.ConnectionCooldown.ClearCooldown(connectionID)
}

// ResilienceStatus represents the current resilience state
type ResilienceStatus struct {
	ProviderCircuits map[string]string `json:"provider_circuits"`
	ActiveCooldowns  map[string]int64  `json:"active_cooldowns"` // seconds remaining
	ModelLockouts    []*LockoutEntry   `json:"model_lockouts"`
}

// GetStatus returns a snapshot of resilience status
func (rm *ResilienceManager) GetStatus() *ResilienceStatus {
	status := &ResilienceStatus{
		ProviderCircuits: make(map[string]string),
		ActiveCooldowns:  make(map[string]int64),
		ModelLockouts:    make([]*LockoutEntry, 0),
	}

	for providerID, state := range rm.CircuitBreaker.GetAllStatus() {
		status.ProviderCircuits[providerID] = state.String()
	}

	status.ModelLockouts = rm.ModelLockout.GetAllLockouts()

	return status
}
