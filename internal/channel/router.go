package channel

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"log/slog"
)

// Router handles channel selection with weighted random distribution and circuit breaker
type Router struct {
	repo           *Repository
	circuitBreaker *CircuitBreaker
	mu             sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

const (
	circuitFailureThreshold = 5
	circuitOpenDuration     = 30 * time.Second
	circuitResetInterval    = 10 * time.Second
)

// CircuitBreaker implements the circuit breaker pattern for channel health
type CircuitBreaker struct {
	states        map[int64]CircuitBreakerState
	failureCounts map[int64]int
	lastFailure   map[int64]time.Time
	mu            sync.RWMutex
	halfOpenMax   int // Max requests to allow in half-open state
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		states:        make(map[int64]CircuitBreakerState),
		failureCounts: make(map[int64]int),
		lastFailure:   make(map[int64]time.Time),
		halfOpenMax:   3,
	}
}

// NewRouter creates a new channel router
func NewRouter(repo *Repository) *Router {
	return &Router{
		repo:           repo,
		circuitBreaker: NewCircuitBreaker(),
	}
}

// SelectChannel selects a channel using weighted random distribution
func (r *Router) SelectChannel(ctx context.Context, model string) (*Channel, error) {
	channels, err := r.repo.ListActive(ctx, model)
	if err != nil {
		return nil, err
	}

	if len(channels) == 0 {
		slog.Warn("No active channels available for model", slog.String("model", model))
		return nil, ErrChannelNotFound
	}

	// Filter by model support and circuit breaker state
	r.circuitBreaker.mu.RLock()
	var eligible []*Channel
	for _, ch := range channels {
		if !ch.IsModelSupported(model) {
			continue
		}
		if r.circuitBreaker.states[ch.ID] == CircuitOpen {
			// Check if circuit should transition from open to half-open
			if time.Since(r.circuitBreaker.lastFailure[ch.ID]) > circuitOpenDuration {
				r.circuitBreaker.mu.RUnlock()
				r.circuitBreaker.mu.Lock()
				r.circuitBreaker.states[ch.ID] = CircuitHalfOpen
				r.circuitBreaker.failureCounts[ch.ID] = 0
				r.circuitBreaker.mu.Unlock()
				r.circuitBreaker.mu.RLock()
				eligible = append(eligible, ch)
				slog.Info("Circuit breaker transitioning to half-open",
					slog.Int64("channel_id", ch.ID))
			}
			continue
		}
		eligible = append(eligible, ch)
	}
	r.circuitBreaker.mu.RUnlock()

	if len(eligible) == 0 {
		slog.Warn("No eligible channels after circuit breaker filtering")
		return nil, ErrChannelInactive
	}

	// Weighted random selection
	selected := weightedRandomSelect(eligible)
	slog.Debug("Selected channel",
		slog.Int64("channel_id", selected.ID),
		slog.String("name", selected.Name),
		slog.Int("priority", selected.Priority),
		slog.Int("ratio", selected.Ratio))

	return selected, nil
}

// RecordSuccess records a successful request for circuit breaker
func (r *Router) RecordSuccess(channelID int64) {
	r.circuitBreaker.mu.Lock()
	defer r.circuitBreaker.mu.Unlock()

	delete(r.circuitBreaker.failureCounts, channelID)
	if r.circuitBreaker.states[channelID] == CircuitHalfOpen {
		r.circuitBreaker.states[channelID] = CircuitClosed
		slog.Info("Circuit breaker closed after successful request",
			slog.Int64("channel_id", channelID))
	}
}

// RecordFailure records a failed request for circuit breaker
func (r *Router) RecordFailure(channelID int64) {
	r.circuitBreaker.mu.Lock()
	defer r.circuitBreaker.mu.Unlock()

	r.circuitBreaker.failureCounts[channelID]++
	r.circuitBreaker.lastFailure[channelID] = time.Now()

	if r.circuitBreaker.failureCounts[channelID] >= circuitFailureThreshold {
		r.circuitBreaker.states[channelID] = CircuitOpen
		slog.Warn("Circuit breaker opened due to failures",
			slog.Int64("channel_id", channelID),
			slog.Int("failure_count", r.circuitBreaker.failureCounts[channelID]))
	}
}

// GetCircuitState returns the current circuit breaker state for a channel
func (r *Router) GetCircuitState(channelID int64) CircuitBreakerState {
	r.circuitBreaker.mu.RLock()
	defer r.circuitBreaker.mu.RUnlock()
	return r.circuitBreaker.states[channelID]
}

// ResetCircuit resets the circuit breaker for a channel
func (r *Router) ResetCircuit(channelID int64) {
	r.circuitBreaker.mu.Lock()
	defer r.circuitBreaker.mu.Unlock()

	delete(r.circuitBreaker.states, channelID)
	delete(r.circuitBreaker.failureCounts, channelID)
	delete(r.circuitBreaker.lastFailure, channelID)
	slog.Info("Circuit breaker reset", slog.Int64("channel_id", channelID))
}

// weightedRandomSelect selects a channel based on weighted random distribution
func weightedRandomSelect(channels []*Channel) *Channel {
	if len(channels) == 1 {
		return channels[0]
	}

	// Calculate total weight
	totalWeight := 0
	for _, ch := range channels {
		// Weight = priority * ratio (higher priority and ratio = more likely)
		totalWeight += ch.Priority * ch.Ratio
	}

	if totalWeight == 0 {
		return channels[0]
	}

	// Random selection based on weight
	r := rand.Intn(totalWeight)
	cumulative := 0

	for _, ch := range channels {
		cumulative += ch.Priority * ch.Ratio
		if r < cumulative {
			return ch
		}
	}

	return channels[len(channels)-1]
}

// SelectWithFailover attempts to select a channel, and if it fails, tries the next one
func (r *Router) SelectWithFailover(ctx context.Context, model string, maxAttempts int) (*Channel, error) {
	attempts := 0

	for attempts < maxAttempts {
		channel, err := r.SelectChannel(ctx, model)
		if err != nil {
			return nil, err
		}

		// Return the channel for the caller to attempt
		if attempts > 0 {
			slog.Info("Failover: trying next channel",
				slog.Int64("channel_id", channel.ID),
				slog.Int("attempt", attempts+1))
		}

		attempts++
	}

	return nil, ErrChannelInactive
}

// HealthStats represents channel health statistics
type HealthStats struct {
	ChannelID     int64     `json:"channel_id"`
	Name          string    `json:"name"`
	State         string    `json:"state"`
	FailureCount  int       `json:"failure_count"`
	LastFailure   time.Time `json:"last_failure,omitempty"`
	SuccessRate   float64   `json:"success_rate"`
	AvgLatencyMS  float64   `json:"avg_latency_ms"`
	TotalRequests int64     `json:"total_requests"`
}

// GetHealthStats returns health statistics for all channels
func (r *Router) GetHealthStats(channels []*Channel) []HealthStats {
	stats := make([]HealthStats, len(channels))

	r.circuitBreaker.mu.RLock()
	defer r.circuitBreaker.mu.RUnlock()

	for i, ch := range channels {
		state := r.circuitBreaker.states[ch.ID]
		stateStr := "closed"
		switch state {
		case CircuitOpen:
			stateStr = "open"
		case CircuitHalfOpen:
			stateStr = "half_open"
		}

		lastFailure := time.Time{}
		if t, ok := r.circuitBreaker.lastFailure[ch.ID]; ok {
			lastFailure = t
		}

		stats[i] = HealthStats{
			ChannelID:    ch.ID,
			Name:         ch.Name,
			State:        stateStr,
			FailureCount: r.circuitBreaker.failureCounts[ch.ID],
			LastFailure:  lastFailure,
		}
	}

	return stats
}
