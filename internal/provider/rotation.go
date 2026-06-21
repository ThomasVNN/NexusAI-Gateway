package provider

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RotationStrategy defines how providers are selected when rotating
type RotationStrategy int

const (
	// PriorityBased selects providers by priority (higher = better)
	PriorityBased RotationStrategy = iota
	// RoundRobin cycles through providers in order
	RoundRobin
	// LeastLoaded selects provider with lowest latency
	LeastLoaded
	// WeightedRandom selects based on weighted random distribution
	WeightedRandom
)

func (s RotationStrategy) String() string {
	switch s {
	case PriorityBased:
		return "priority"
	case RoundRobin:
		return "round_robin"
	case LeastLoaded:
		return "least_loaded"
	case WeightedRandom:
		return "weighted_random"
	default:
		return "unknown"
	}
}

// ProviderSelector selects the best available provider based on strategy
type ProviderSelector struct {
	healthChecker *HealthChecker
	strategy      RotationStrategy
	providers     []*Provider
	index         int
	mu            sync.Mutex
}

// NewProviderSelector creates a new provider selector with the given strategy
func NewProviderSelector(healthChecker *HealthChecker, strategy RotationStrategy, providers []*Provider) *ProviderSelector {
	return &ProviderSelector{
		healthChecker: healthChecker,
		strategy:      strategy,
		providers:     providers,
		index:         0,
	}
}

// SelectProvider selects the best available provider based on the configured strategy
func (ps *ProviderSelector) SelectProvider(ctx context.Context) (*Provider, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var candidates []*Provider

	// Filter to only healthy providers
	for _, p := range ps.providers {
		if !p.Enabled {
			continue
		}

		// Check if provider is healthy (circuit breaker not open)
		if ps.healthChecker != nil {
			if !ps.healthChecker.IsProviderHealthy(p.ID) {
				slog.Debug("Skipping unhealthy provider", slog.String("provider_id", p.ID))
				continue
			}
		}

		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return nil, ErrNoHealthyProviders
	}

	// Apply selection strategy
	switch ps.strategy {
	case PriorityBased:
		return ps.selectByPriority(candidates), nil
	case RoundRobin:
		return ps.selectRoundRobin(candidates), nil
	case LeastLoaded:
		return ps.selectByLatency(candidates), nil
	case WeightedRandom:
		return ps.selectWeightedRandom(candidates), nil
	default:
		return ps.selectByPriority(candidates), nil
	}
}

// selectByPriority selects the provider with the highest priority
func (ps *ProviderSelector) selectByPriority(candidates []*Provider) *Provider {
	var best *Provider
	bestPriority := -1

	for _, p := range candidates {
		if p.Priority > bestPriority {
			best = p
			bestPriority = p.Priority
		}
	}

	return best
}

// selectRoundRobin cycles through providers in order
func (ps *ProviderSelector) selectRoundRobin(candidates []*Provider) *Provider {
	if len(candidates) == 0 {
		return nil
	}

	// Find starting point
	startIdx := ps.index % len(candidates)
	selected := candidates[startIdx]
	ps.index++

	return selected
}

// selectByLatency selects the provider with the lowest latency
func (ps *ProviderSelector) selectByLatency(candidates []*Provider) *Provider {
	var best *Provider
	bestLatency := int(^uint(0) >> 1) // Max int

	for _, p := range candidates {
		status, exists := ps.healthChecker.GetHealthStatus(p.ID)
		if !exists || status == nil {
			// No health data, use as fallback
			if best == nil {
				best = p
			}
			continue
		}

		if status.LatencyMs < bestLatency {
			best = p
			bestLatency = status.LatencyMs
		}
	}

	return best
}

// selectWeightedRandom selects a provider based on weighted random distribution
func (ps *ProviderSelector) selectWeightedRandom(candidates []*Provider) *Provider {
	if len(candidates) == 0 {
		return nil
	}

	// Calculate weights based on inverse latency and success rate
	type weightedProvider struct {
		provider *Provider
		weight   float64
	}

	weighted := make([]weightedProvider, 0, len(candidates))
	var totalWeight float64

	for _, p := range candidates {
		weight := float64(p.Priority + 1) // Base weight from priority

		// Adjust weight by health metrics
		if ps.healthChecker != nil {
			metrics, exists := ps.healthChecker.GetMetrics(p.ID)
			if exists && metrics != nil {
				// Higher success rate = higher weight
				weight *= metrics.SuccessRate
				// Lower latency = higher weight (inverse)
				if metrics.AvgLatencyMs > 0 {
					weight /= (metrics.AvgLatencyMs / 100.0)
				}
			}
		}

		if weight <= 0 {
			weight = 0.01 // Minimum weight
		}

		weighted = append(weighted, weightedProvider{p, weight})
		totalWeight += weight
	}

	if totalWeight <= 0 {
		return candidates[0]
	}

	// Simple weighted random selection
	// In production, use a more sophisticated algorithm
	target := float64(time.Now().UnixNano()%int64(totalWeight*1000000)) / 1000000

	var cumulative float64
	for _, wp := range weighted {
		cumulative += wp.weight
		if cumulative >= target {
			return wp.provider
		}
	}

	return candidates[0]
}

// GetHealthyProviders returns all healthy providers
func (ps *ProviderSelector) GetHealthyProviders() []*Provider {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var healthy []*Provider
	for _, p := range ps.providers {
		if !p.Enabled {
			continue
		}

		if ps.healthChecker != nil {
			if !ps.healthChecker.IsProviderHealthy(p.ID) {
				continue
			}
		}

		healthy = append(healthy, p)
	}

	return healthy
}

// GetProviderStatus returns detailed status for all providers
func (ps *ProviderSelector) GetProviderStatus() []ProviderStatus {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	statuses := make([]ProviderStatus, 0, len(ps.providers))
	for _, p := range ps.providers {
		status := ProviderStatus{
			Provider:     p,
			IsEnabled:    p.Enabled,
			IsHealthy:    true,
			CircuitState: "closed",
		}

		if ps.healthChecker != nil {
			hs, exists := ps.healthChecker.GetHealthStatus(p.ID)
			if exists && hs != nil {
				status.IsHealthy = hs.IsHealthy
				status.LatencyMs = hs.LatencyMs
				status.SuccessRate = hs.SuccessRate
				status.ErrorRate = hs.ErrorRate
				status.CircuitState = hs.CircuitState
				status.LastCheck = hs.LastCheck
				status.NextCheck = hs.NextCheck
			}

			metrics, exists := ps.healthChecker.GetMetrics(p.ID)
			if exists && metrics != nil {
				status.TotalChecks = metrics.TotalChecks
				status.SuccessfulChecks = metrics.SuccessfulChecks
				status.FailedChecks = metrics.FailedChecks
				status.AvgLatencyMs = int(metrics.AvgLatencyMs)
				status.P99LatencyMs = metrics.P99LatencyMs
			}
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// ProviderStatus represents the status of a provider
type ProviderStatus struct {
	Provider         *Provider
	IsEnabled        bool
	IsHealthy        bool
	CircuitState     string
	LatencyMs        int
	SuccessRate      float64
	ErrorRate        float64
	LastCheck        time.Time
	NextCheck        time.Time
	TotalChecks      int
	SuccessfulChecks int
	FailedChecks     int
	AvgLatencyMs     int
	P99LatencyMs     int
}

// Custom errors
var ErrNoHealthyProviders = &NoHealthyProvidersError{}

type NoHealthyProvidersError struct{}

func (e *NoHealthyProvidersError) Error() string {
	return "no healthy providers available"
}
