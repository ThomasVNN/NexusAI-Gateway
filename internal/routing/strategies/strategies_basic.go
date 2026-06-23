package strategies

import (
	"math/rand"
	"sort"
)

// PriorityStrategy selects the provider with the highest priority (lowest number)
type PriorityStrategy struct {
	orderedList []string // Ordered list of provider IDs
}

func (s *PriorityStrategy) Name() string {
	return "priority"
}

func (s *PriorityStrategy) Configure(config map[string]interface{}) error {
	if ordered, ok := config["ordered_list"].([]interface{}); ok {
		s.orderedList = make([]string, len(ordered))
		for i, v := range ordered {
			if str, ok := v.(string); ok {
				s.orderedList[i] = str
			}
		}
	}
	return nil
}

func (s *PriorityStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Sort by priority (lower = higher priority)
	providers := make([]*Provider, len(ctx.Providers))
	copy(providers, ctx.Providers)
	
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].Tier != providers[j].Tier {
			return providers[i].Tier < providers[j].Tier
		}
		return ctx.HealthScores[providers[i].ID] > ctx.HealthScores[providers[j].ID]
	})

	// Return first healthy provider
	for _, p := range providers {
		if p.IsHealthy {
			return p, nil
		}
	}
	return providers[0], nil
}

// WeightedStrategy selects a provider based on weighted random distribution
type WeightedStrategy struct {
	weights map[string]float64
}

func (s *WeightedStrategy) Name() string {
	return "weighted"
}

func (s *WeightedStrategy) Configure(config map[string]interface{}) error {
	if weights, ok := config["weights"].(map[string]interface{}); ok {
		s.weights = make(map[string]float64)
		for id, w := range weights {
			if f, ok := w.(float64); ok {
				s.weights[id] = f
			}
		}
	}
	return nil
}

func (s *WeightedStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Calculate total weight
	var totalWeight float64
	weightMap := make(map[string]float64)
	
	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}
		var w float64 = 1.0
		if s.weights != nil {
			if customWeight, ok := s.weights[p.ID]; ok {
				w = customWeight
			}
		} else {
			// Default: weight by health and quota
			w = p.Health * ctx.QuotaScores[p.ID]
		}
		weightMap[p.ID] = w
		totalWeight += w
	}

	if totalWeight == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Weighted random selection
	r := float64(rand.Int63n(int64(totalWeight*1000))) / 1000
	var cumulative float64
	
	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}
		cumulative += weightMap[p.ID]
		if r <= cumulative {
			return p, nil
		}
	}
	
	// Fallback to last provider
	for i := len(ctx.Providers) - 1; i >= 0; i-- {
		if ctx.Providers[i].IsHealthy {
			return ctx.Providers[i], nil
		}
	}
	
	return nil, ErrNoHealthyProvider
}

// RoundRobinStrategy cycles through providers in order
type RoundRobinStrategy struct {
	mu       int
	provider string
}

func (s *RoundRobinStrategy) Name() string {
	return "round-robin"
}

func (s *RoundRobinStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *RoundRobinStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Increment and wrap
	s.mu = (s.mu + 1) % len(healthyProviders)
	return healthyProviders[s.mu], nil
}

// P2CStrategy implements Power-of-2-Choices load balancing
type P2CStrategy struct {
	mu int
}

func (s *P2CStrategy) Name() string {
	return "p2c"
}

func (s *P2CStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *P2CStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	if len(healthyProviders) == 1 {
		return healthyProviders[0], nil
	}

	// Pick two random providers
	i1 := rand.Intn(len(healthyProviders))
	i2 := (i1 + 1 + rand.Intn(len(healthyProviders)-1)) % len(healthyProviders)
	
	p1 := healthyProviders[i1]
	p2 := healthyProviders[i2]

	// Compare by load (quota usage percentage)
	load1 := float64(p1.QuotaUsed) / float64(p1.QuotaLimit+1)
	load2 := float64(p2.QuotaUsed) / float64(p2.QuotaLimit+1)

	if load1 <= load2 {
		return p1, nil
	}
	return p2, nil
}

// RandomStrategy selects a random provider
type RandomStrategy struct{}

func (s *RandomStrategy) Name() string {
	return "random"
}

func (s *RandomStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *RandomStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	return healthyProviders[rand.Intn(len(healthyProviders))], nil
}

// LeastUsedStrategy selects the provider with the lowest quota usage
type LeastUsedStrategy struct{}

func (s *LeastUsedStrategy) Name() string {
	return "least-used"
}

func (s *LeastUsedStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *LeastUsedStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	var best *Provider
	var lowestUsage float64 = -1

	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}

		var usage float64
		if p.QuotaLimit > 0 {
			usage = float64(p.QuotaUsed) / float64(p.QuotaLimit)
		} else {
			usage = 0 // Unlimited = lowest usage
		}

		if lowestUsage < 0 || usage < lowestUsage {
			lowestUsage = usage
			best = p
		}
	}

	if best == nil {
		return nil, ErrNoHealthyProvider
	}
	return best, nil
}

// CostOptimizedStrategy selects the cheapest provider
type CostOptimizedStrategy struct{}

func (s *CostOptimizedStrategy) Name() string {
	return "cost-optimized"
}

func (s *CostOptimizedStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *CostOptimizedStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	var best *Provider
	var lowestCost float64 = -1

	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}

		// Check budget constraint
		cost := p.CostPer1KIn + p.CostPer1KOut
		if ctx.CostBudget > 0 && cost*1000 > ctx.CostBudget {
			continue
		}

		if lowestCost < 0 || cost < lowestCost {
			lowestCost = cost
			best = p
		}
	}

	if best == nil {
		return nil, ErrNoHealthyProvider
	}
	return best, nil
}
