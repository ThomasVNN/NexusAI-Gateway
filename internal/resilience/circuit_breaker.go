package resilience

import (
	"errors"
	"sync"
	"time"
)

// CBState represents the state of a circuit breaker
type CBState string

const (
	CBStateClosed    CBState = "CLOSED"
	CBStateHalfOpen          = "HALF_OPEN"
	CBStateOpen              = "OPEN"
	CBStateDegraded          = "DEGRADED"
)

func (s CBState) String() string {
	return string(s)
}

// ProviderCB is the provider-level circuit breaker
type ProviderCB struct {
	mu              sync.RWMutex
	ProviderID      string
	State          CBState
	FailureCount    int
	SuccessCount    int
	LastFailure     time.Time
	CooldownEnds   time.Time
	LastStateChange time.Time
	
	// Configuration
	FailureThreshold   int
	SuccessThreshold  int
	CooldownDuration  time.Duration
	DegradedThreshold int
	
	// Metrics
	TotalRequests  int64
	TotalFailures  int64
	TotalSuccesses int64
}

// NewProviderCB creates a new provider circuit breaker
func NewProviderCB(providerID string) *ProviderCB {
	return &ProviderCB{
		ProviderID:       providerID,
		State:           CBStateClosed,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		CooldownDuration: 30 * time.Second,
		DegradedThreshold: 10,
	}
}

// Allow checks if a request should be allowed
func (cb *ProviderCB) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.State {
	case CBStateClosed:
		return true

	case CBStateHalfOpen:
		return true

	case CBStateOpen:
		if time.Now().After(cb.CooldownEnds) {
			cb.transitionToLocked(CBStateHalfOpen)
			return true
		}
		return false

	case CBStateDegraded:
		return true
	}

	return true
}

// RecordSuccess records a successful request
func (cb *ProviderCB) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.TotalRequests++
	cb.TotalSuccesses++

	switch cb.State {
	case CBStateClosed:
		cb.FailureCount = 0

	case CBStateHalfOpen:
		cb.SuccessCount++
		if cb.SuccessCount >= cb.SuccessThreshold {
			cb.transitionToLocked(CBStateClosed)
			cb.FailureCount = 0
			cb.SuccessCount = 0
		}

	case CBStateOpen:
		cb.transitionToLocked(CBStateHalfOpen)
		cb.SuccessCount = 1

	case CBStateDegraded:
		cb.FailureCount--
		if cb.FailureCount <= 0 {
			cb.transitionToLocked(CBStateClosed)
			cb.FailureCount = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *ProviderCB) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.TotalRequests++
	cb.TotalFailures++
	cb.LastFailure = time.Now()

	switch cb.State {
	case CBStateClosed:
		cb.FailureCount++
		cb.SuccessCount = 0
		if cb.FailureCount >= cb.FailureThreshold {
			cb.transitionToLocked(CBStateOpen)
			return
		}
		if cb.FailureCount >= cb.DegradedThreshold {
			cb.transitionToLocked(CBStateDegraded)
		}

	case CBStateHalfOpen:
		cb.transitionToLocked(CBStateOpen)

	case CBStateOpen:
		cb.CooldownEnds = time.Now().Add(cb.CooldownDuration)

	case CBStateDegraded:
		cb.FailureCount++
		if cb.FailureCount >= cb.DegradedThreshold {
			cb.transitionToLocked(CBStateOpen)
		}
	}
}

func (cb *ProviderCB) transitionTo(state CBState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionToLocked(state)
}

func (cb *ProviderCB) transitionToLocked(state CBState) {
	if cb.State == state {
		return
	}
	cb.State = state
	cb.LastStateChange = time.Now()
	
	if state == CBStateOpen {
		cb.CooldownEnds = time.Now().Add(cb.CooldownDuration)
	}
}

// GetState returns the current state
func (cb *ProviderCB) GetState() CBState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.State
}

// Stats returns current statistics
func (cb *ProviderCB) Stats() *CBStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return &CBStats{
		ProviderID:      cb.ProviderID,
		State:          cb.State.String(),
		FailureCount:   cb.FailureCount,
		SuccessCount:   cb.SuccessCount,
		LastFailure:    cb.LastFailure,
		LastStateChange: cb.LastStateChange,
		TotalRequests:  cb.TotalRequests,
		TotalFailures:  cb.TotalFailures,
		TotalSuccesses: cb.TotalSuccesses,
		CooldownEnds:   cb.CooldownEnds,
	}
}

// CBStats contains statistics for a circuit breaker
type CBStats struct {
	ProviderID      string    `json:"provider_id"`
	State          string    `json:"state"`
	FailureCount   int       `json:"failure_count"`
	SuccessCount   int       `json:"success_count"`
	LastFailure    time.Time `json:"last_failure"`
	LastStateChange time.Time `json:"last_state_change"`
	TotalRequests  int64     `json:"total_requests"`
	TotalFailures  int64     `json:"total_failures"`
	TotalSuccesses int64     `json:"total_successes"`
	CooldownEnds   time.Time `json:"cooldown_ends,omitempty"`
}

// ConnectionCooldown manages per-account cooldown
type ConnectionCooldown struct {
	mu         sync.RWMutex
	AccountID  string
	ProviderID string
	CooledUntil time.Time
	RetryAfter time.Duration
	Reason     string
}

// NewConnectionCooldown creates a new connection cooldown
func NewConnectionCooldown(accountID, providerID string) *ConnectionCooldown {
	return &ConnectionCooldown{
		AccountID:  accountID,
		ProviderID: providerID,
		RetryAfter: 30 * time.Second,
		Reason:     "rate_limit",
	}
}

// CanUse checks if the cooldown has expired
func (cc *ConnectionCooldown) CanUse() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return time.Now().After(cc.CooledUntil)
}

// SetCooldown sets a cooldown period
func (cc *ConnectionCooldown) SetCooldown(duration time.Duration, reason string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.CooledUntil = time.Now().Add(duration)
	cc.RetryAfter = duration
	cc.Reason = reason
}

// Remaining returns the remaining cooldown time
func (cc *ConnectionCooldown) Remaining() time.Duration {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	remaining := cc.CooledUntil.Sub(time.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CooldownMgr manages multiple connection cooldowns
type CooldownMgr struct {
	mu        sync.RWMutex
	cooldowns map[string]*ConnectionCooldown
}

// NewCooldownMgr creates a new cooldown manager
func NewCooldownMgr() *CooldownMgr {
	return &CooldownMgr{
		cooldowns: make(map[string]*ConnectionCooldown),
	}
}

func cooldownKey(accountID, providerID string) string {
	return accountID + ":" + providerID
}

// GetCooldown gets or creates a cooldown for an account
func (m *CooldownMgr) GetCooldown(accountID, providerID string) *ConnectionCooldown {
	m.mu.RLock()
	cc, exists := m.cooldowns[cooldownKey(accountID, providerID)]
	m.mu.RUnlock()
	
	if exists {
		return cc
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := cooldownKey(accountID, providerID)
	if cc, exists = m.cooldowns[key]; exists {
		return cc
	}

	cc = NewConnectionCooldown(accountID, providerID)
	m.cooldowns[key] = cc
	return cc
}

// RemoveCooldown removes a cooldown
func (m *CooldownMgr) RemoveCooldown(accountID, providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cooldowns, cooldownKey(accountID, providerID))
}

// ModelLockout manages provider+model level lockouts
type ModelLockout struct {
	mu         sync.RWMutex
	ProviderID string
	ModelID    string
	LockedUntil time.Time
	Reason     string
	LockedBy   string
}

// NewModelLockout creates a new model lockout
func NewModelLockout(providerID, modelID string) *ModelLockout {
	return &ModelLockout{
		ProviderID: providerID,
		ModelID:   modelID,
		Reason:    "manual",
	}
}

// IsLocked checks if the model is currently locked
func (ml *ModelLockout) IsLocked() bool {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return time.Now().Before(ml.LockedUntil)
}

// Lock locks the model for a duration
func (ml *ModelLockout) Lock(duration time.Duration, reason, lockedBy string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.LockedUntil = time.Now().Add(duration)
	ml.Reason = reason
	ml.LockedBy = lockedBy
}

// Unlock releases the lock
func (ml *ModelLockout) Unlock() {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.LockedUntil = time.Time{}
}

// Remaining returns remaining lock duration
func (ml *ModelLockout) Remaining() time.Duration {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	remaining := ml.LockedUntil.Sub(time.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// LockoutMgr manages multiple model lockouts
type LockoutMgr struct {
	mu       sync.RWMutex
	lockouts map[string]*ModelLockout
}

// NewLockoutMgr creates a new lockout manager
func NewLockoutMgr() *LockoutMgr {
	return &LockoutMgr{
		lockouts: make(map[string]*ModelLockout),
	}
}

func lockoutKey(providerID, modelID string) string {
	return providerID + ":" + modelID
}

// GetLockout gets or creates a lockout for a model
func (m *LockoutMgr) GetLockout(providerID, modelID string) *ModelLockout {
	m.mu.RLock()
	ml, exists := m.lockouts[lockoutKey(providerID, modelID)]
	m.mu.RUnlock()

	if exists {
		return ml
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := lockoutKey(providerID, modelID)
	if ml, exists = m.lockouts[key]; exists {
		return ml
	}

	ml = NewModelLockout(providerID, modelID)
	m.lockouts[key] = ml
	return ml
}

// IsLocked checks if a specific model is locked
func (m *LockoutMgr) IsLocked(providerID, modelID string) bool {
	ml := m.GetLockout(providerID, modelID)
	return ml.IsLocked()
}

// RemoveLockout removes a lockout
func (m *LockoutMgr) RemoveLockout(providerID, modelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lockouts, lockoutKey(providerID, modelID))
}

// ThreeLayerResilience provides the complete 3-layer resilience system
type ThreeLayerResilience struct {
	providerCBs map[string]*ProviderCB
	cooldownMgr *CooldownMgr
	lockoutMgr  *LockoutMgr
	mu          sync.RWMutex
}

// NewThreeLayerResilience creates a new 3-layer resilience system
func NewThreeLayerResilience() *ThreeLayerResilience {
	return &ThreeLayerResilience{
		providerCBs: make(map[string]*ProviderCB),
		cooldownMgr: NewCooldownMgr(),
		lockoutMgr:  NewLockoutMgr(),
	}
}

// GetProviderCB gets or creates a provider circuit breaker
func (t *ThreeLayerResilience) GetProviderCB(providerID string) *ProviderCB {
	t.mu.RLock()
	cb, exists := t.providerCBs[providerID]
	t.mu.RUnlock()

	if exists {
		return cb
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if cb, exists = t.providerCBs[providerID]; exists {
		return cb
	}

	cb = NewProviderCB(providerID)
	t.providerCBs[providerID] = cb
	return cb
}

// GetCooldownMgr returns the cooldown manager
func (t *ThreeLayerResilience) GetCooldownMgr() *CooldownMgr {
	return t.cooldownMgr
}

// GetLockoutMgr returns the lockout manager
func (t *ThreeLayerResilience) GetLockoutMgr() *LockoutMgr {
	return t.lockoutMgr
}

// CanRoute checks if routing is allowed for a provider/model
func (t *ThreeLayerResilience) CanRoute(providerID, modelID, accountID string) bool {
	cb := t.GetProviderCB(providerID)
	if !cb.Allow() {
		return false
	}

	cc := t.cooldownMgr.GetCooldown(accountID, providerID)
	if !cc.CanUse() {
		return false
	}

	if modelID != "" {
		if t.lockoutMgr.IsLocked(providerID, modelID) {
			return false
		}
	}

	return true
}

// RecordSuccess records a successful request
func (t *ThreeLayerResilience) RecordSuccess(providerID, modelID, accountID string) {
	cb := t.GetProviderCB(providerID)
	cb.RecordSuccess()

	t.cooldownMgr.RemoveCooldown(accountID, providerID)
}

// RecordFailure records a failed request
func (t *ThreeLayerResilience) RecordFailure(providerID, modelID, accountID string) {
	cb := t.GetProviderCB(providerID)
	cb.RecordFailure()
}

// SetCooldown sets a cooldown for an account
func (t *ThreeLayerResilience) SetCooldown(accountID, providerID string, duration time.Duration, reason string) {
	cc := t.cooldownMgr.GetCooldown(accountID, providerID)
	cc.SetCooldown(duration, reason)
}

// LockModel locks a model
func (t *ThreeLayerResilience) LockModel(providerID, modelID string, duration time.Duration, reason, lockedBy string) {
	ml := t.lockoutMgr.GetLockout(providerID, modelID)
	ml.Lock(duration, reason, lockedBy)
}

// UnlockModel unlocks a model
func (t *ThreeLayerResilience) UnlockModel(providerID, modelID string) {
	t.lockoutMgr.RemoveLockout(providerID, modelID)
}

// GetAllStats returns stats for all circuit breakers
func (t *ThreeLayerResilience) GetAllStats() []*CBStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make([]*CBStats, 0, len(t.providerCBs))
	for _, cb := range t.providerCBs {
		stats = append(stats, cb.Stats())
	}
	return stats
}

// Errors
var (
	ErrAllProvidersExhausted = errors.New("all providers exhausted")
	ErrNoProviderInTier     = errors.New("no available provider in tier")
)
