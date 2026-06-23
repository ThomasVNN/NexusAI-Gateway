package strategies

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

// RoutingStrategy defines the interface for routing strategies
type RoutingStrategy interface {
	Name() string
	SelectProvider(ctx *RoutingContext) (*Provider, error)
	Configure(config map[string]interface{}) error
}

// Provider represents a routing-enabled provider
type Provider struct {
	ID             string
	Name           string
	Tier           int // 0=subscription, 1=api_key, 2=cheap, 3=free
	OAuthSupport   bool
	Health         float64 // 0.0-1.0
	QuotaUsed      int64
	QuotaLimit     int64
	CostPer1KIn    float64
	CostPer1KOut   float64
	LatencyMs      int64
	StabilityScore float64 // 0.0-1.0
	TaskFitScores  map[string]float64 // task_type -> score
	ModelIDs       []string
	IsHealthy      bool
	LastFailureTime time.Time
	LastSuccessTime time.Time
}

// RoutingContext contains all information needed for routing decisions
type RoutingContext struct {
	TaskType         string  // "coding", "review", "planning", "general"
	CostBudget       float64
	LatencyBudget    time.Duration
	RequiredCaps     []string
	ContextSize      int
	SessionID        string
	Providers        []*Provider
	
	// 9-factor scores (pre-calculated)
	HealthScores      map[string]float64
	QuotaScores       map[string]float64
	CostScores        map[string]float64
	LatencyScores     map[string]float64
	TaskFitScores     map[string]float64
	SpecificityScores map[string]float64
	StabilityScores   map[string]float64
	TierPriorities    map[string]int
	TierAffinities    map[string]float64
}

// NewRoutingContext creates a new routing context
func NewRoutingContext(providers []*Provider) *RoutingContext {
	ctx := &RoutingContext{
		Providers:        providers,
		HealthScores:      make(map[string]float64),
		QuotaScores:       make(map[string]float64),
		CostScores:        make(map[string]float64),
		LatencyScores:     make(map[string]float64),
		TaskFitScores:     make(map[string]float64),
		SpecificityScores: make(map[string]float64),
		StabilityScores:   make(map[string]float64),
		TierPriorities:   make(map[string]int),
		TierAffinities:   make(map[string]float64),
		SessionID:        generateSessionID(),
	}
	
	// Calculate 9-factor scores for all providers
	ctx.calculateScores()
	return ctx
}

func generateSessionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// calculateScores computes the 9-factor scores for all providers
func (ctx *RoutingContext) calculateScores() {
	for _, p := range ctx.Providers {
		// 1. Health score (0-1)
		ctx.HealthScores[p.ID] = p.Health
		
		// 2. Quota score (remaining quota as percentage)
		if p.QuotaLimit > 0 {
			remaining := float64(p.QuotaLimit - p.QuotaUsed) / float64(p.QuotaLimit)
			if remaining < 0 {
				remaining = 0
			}
			ctx.QuotaScores[p.ID] = remaining
		} else {
			ctx.QuotaScores[p.ID] = 1.0 // Unlimited
		}
		
		// 3. Cost score (inverse - lower cost = higher score)
		totalCost := p.CostPer1KIn + p.CostPer1KOut
		if totalCost == 0 {
			ctx.CostScores[p.ID] = 1.0 // Free = highest score
		} else {
			// Normalize: typical range 0.001 to 0.1 per 1K tokens
			ctx.CostScores[p.ID] = 1.0 / (1.0 + totalCost*100)
		}
		
		// 4. Latency score (inverse - lower latency = higher score)
		if p.LatencyMs == 0 {
			ctx.LatencyScores[p.ID] = 1.0 // Unknown latency
		} else {
			// Normalize: typical range 100ms to 5000ms
			ctx.LatencyScores[p.ID] = 1.0 / (1.0 + float64(p.LatencyMs)/1000)
		}
		
		// 5. Task fit score
		if p.TaskFitScores != nil {
			if score, ok := p.TaskFitScores[ctx.TaskType]; ok {
				ctx.TaskFitScores[p.ID] = score
			} else {
				ctx.TaskFitScores[p.ID] = 0.5 // Default score
			}
		} else {
			ctx.TaskFitScores[p.ID] = 0.5
		}
		
		// 6. Specificity match score (based on required capabilities)
		matchCount := 0
		for _, cap := range ctx.RequiredCaps {
			for _, pCap := range p.Capabilities() {
				if cap == pCap {
					matchCount++
					break
				}
			}
		}
		if len(ctx.RequiredCaps) > 0 {
			ctx.SpecificityScores[p.ID] = float64(matchCount) / float64(len(ctx.RequiredCaps))
		} else {
			ctx.SpecificityScores[p.ID] = 1.0
		}
		
		// 7. Stability score
		ctx.StabilityScores[p.ID] = p.StabilityScore
		
		// 8. Tier priority (lower tier number = higher priority)
		ctx.TierPriorities[p.ID] = p.Tier
		
		// 9. Tier affinity (based on context and budget)
		affinity := 1.0
		if ctx.CostBudget > 0 && (p.CostPer1KIn+p.CostPer1KOut)*1000 > ctx.CostBudget {
			affinity *= 0.5 // Penalize if over budget
		}
		if ctx.LatencyBudget > 0 && time.Duration(p.LatencyMs)*time.Millisecond > ctx.LatencyBudget {
			affinity *= 0.5 // Penalize if over latency budget
		}
		ctx.TierAffinities[p.ID] = affinity
	}
}

// Capabilities returns the capabilities of a provider (simplified)
func (p *Provider) Capabilities() []string {
	caps := make([]string, 0)
	if p.OAuthSupport {
		caps = append(caps, "oauth")
	}
	if p.Tier == 0 {
		caps = append(caps, "premium")
	}
	if p.Health > 0.9 {
		caps = append(caps, "high_quality")
	}
	return caps
}

// Errors
var (
	ErrNoProviderAvailable = errors.New("no available provider")
	ErrNoHealthyProvider  = errors.New("no healthy provider")
	ErrInvalidConfig       = errors.New("invalid configuration")
)

// StrategyRegistry manages routing strategies
type StrategyRegistry struct {
	mu         sync.RWMutex
	strategies map[string]RoutingStrategy
	defaultStrategy string
}

// NewStrategyRegistry creates a new strategy registry
func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make(map[string]RoutingStrategy),
	}
}

// Register registers a routing strategy
func (r *StrategyRegistry) Register(strategy RoutingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[strategy.Name()] = strategy
}

// Get returns a strategy by name
func (r *StrategyRegistry) Get(name string) (RoutingStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.strategies[name]
	if !ok {
		return nil, ErrInvalidConfig
	}
	return s, nil
}

// List returns all registered strategies
func (r *StrategyRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.strategies))
	for name := range r.strategies {
		names = append(names, name)
	}
	return names
}

// SetDefault sets the default strategy
func (r *StrategyRegistry) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultStrategy = name
}

// GetDefault returns the default strategy
func (r *StrategyRegistry) GetDefault() (RoutingStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultStrategy == "" {
		return nil, ErrInvalidConfig
	}
	s, ok := r.strategies[r.defaultStrategy]
	if !ok {
		return nil, ErrInvalidConfig
	}
	return s, nil
}

// AvailableStrategies returns a map of all available strategies
func AvailableStrategies() map[string]RoutingStrategy {
	strategies := map[string]RoutingStrategy{
		"priority":        &PriorityStrategy{},
		"weighted":        &WeightedStrategy{},
		"round-robin":     &RoundRobinStrategy{},
		"p2c":             &P2CStrategy{},
		"random":          &RandomStrategy{},
		"least-used":      &LeastUsedStrategy{},
		"cost-optimized":  &CostOptimizedStrategy{},
		"reset-aware":     &ResetAwareStrategy{},
		"strict-random":   &StrictRandomStrategy{},
		"auto":            &AutoComboStrategy{},
		"lkgp":            &LKGPStrategy{},
		"context-optimized": &ContextOptimizedStrategy{},
		"context-relay":    &ContextRelayStrategy{},
		"fill-first":      &FillFirstStrategy{},
	}
	return strategies
}

// GetAllStrategies returns all strategies with their names
func GetAllStrategies() []RoutingStrategy {
	strategies := AvailableStrategies()
	result := make([]RoutingStrategy, 0, len(strategies))
	for _, s := range strategies {
		result = append(result, s)
	}
	return result
}
