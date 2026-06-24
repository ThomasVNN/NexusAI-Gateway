package strategies

import (
	"math"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/routing"
)

// Strategy defines a routing strategy interface
type Strategy interface {
	Name() routing.RoutingStrategy
	Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64
}

// LeastLatencyStrategy selects the fastest responding model
type LeastLatencyStrategy struct{}

func (s *LeastLatencyStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyLeastLatency
}

func (s *LeastLatencyStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Latency
}

// CheapestStrategy selects the most cost-effective model
type CheapestStrategy struct{}

func (s *CheapestStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyCheapest
}

func (s *CheapestStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Cost
}

// QualityFirstStrategy selects the highest quality model
type QualityFirstStrategy struct{}

func (s *QualityFirstStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyQualityFirst
}

func (s *QualityFirstStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Quality
}

// BalancedStrategy uses equal weighting
type BalancedStrategy struct{}

func (s *BalancedStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyBalanced
}

func (s *BalancedStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return (factors.Latency + factors.Cost + factors.Quality + factors.Reliability) / 4.0
}

// CostAwareStrategy emphasizes cost but considers other factors
type CostAwareStrategy struct{}

func (s *CostAwareStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyCostAware
}

func (s *CostAwareStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Cost*0.5 + factors.Quality*0.2 + factors.Reliability*0.15 + factors.Latency*0.15
}

// ThroughputStrategy selects models with highest capacity
type ThroughputStrategy struct{}

func (s *ThroughputStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyThroughput
}

func (s *ThroughputStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Throughput
}

// ReliabilityStrategy selects the most reliable model
type ReliabilityStrategy struct{}

func (s *ReliabilityStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyReliability
}

func (s *ReliabilityStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Reliability
}

// CustomWeightedStrategy uses custom weights per tenant
type CustomWeightedStrategy struct {
	Weights routing.ScoringWeights
}

func (s *CustomWeightedStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyCustomWeighted
}

func (s *CustomWeightedStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Latency*s.Weights.Latency +
		factors.Cost*s.Weights.Cost +
		factors.Quality*s.Weights.Quality +
		factors.Reliability*s.Weights.Reliability +
		factors.Capabilities*s.Weights.Capabilities +
		factors.Throughput*s.Weights.Throughput +
		factors.Compliance*s.Weights.Compliance +
		factors.Security*s.Weights.Security +
		factors.Geographic*s.Weights.Geographic
}

// AdaptiveStrategy adjusts based on request characteristics
type AdaptiveStrategy struct{}

func (s *AdaptiveStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyAdaptive
}

func (s *AdaptiveStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	// Adaptive: dynamically weight based on context window usage
	// If large context needed, prioritize quality
	// If fast response needed, prioritize latency
	contextFactor := float64(model.ContextWindow) / 100000.0 // Normalize

	return factors.Latency*0.3 +
		factors.Cost*0.2 +
		factors.Quality*(0.3+contextFactor*0.1) +
		factors.Reliability*0.15 +
		factors.Capabilities*0.05
}

// LoadBalancedStrategy distributes requests evenly
type LoadBalancedStrategy struct {
	RequestCounts map[string]int64
}

func (s *LoadBalancedStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyLoadBalanced
}

func (s *LoadBalancedStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	// Inverse of request count (lower = better for distribution)
	count := s.RequestCounts[model.Name]
	loadScore := 1.0 / (float64(count) + 1.0)

	return (factors.Latency+factors.Cost+factors.Quality)*0.3 + loadScore*0.1
}

// GeographicStrategy prioritizes regional proximity
type GeographicStrategy struct {
	PreferredRegion string
}

func (s *GeographicStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyGeographic
}

func (s *GeographicStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Geographic*0.5 + factors.Latency*0.2 + factors.Reliability*0.15 + factors.Cost*0.15
}

// ComplianceStrategy prioritizes compliance requirements
type ComplianceStrategy struct{}

func (s *ComplianceStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyCompliance
}

func (s *ComplianceStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Compliance*0.5 + factors.Security*0.25 + factors.Reliability*0.15 + factors.Quality*0.1
}

// SecurityFirstStrategy prioritizes security
type SecurityFirstStrategy struct{}

func (s *SecurityFirstStrategy) Name() routing.RoutingStrategy {
	return routing.StrategySecurityFirst
}

func (s *SecurityFirstStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Security*0.5 + factors.Compliance*0.25 + factors.Reliability*0.15 + factors.Quality*0.1
}

// OfflineFriendlyStrategy prioritizes reliability and fallback options
type OfflineFriendlyStrategy struct{}

func (s *OfflineFriendlyStrategy) Name() routing.RoutingStrategy {
	return routing.StrategyOfflineFriendly
}

func (s *OfflineFriendlyStrategy) Score(model *routing.ExtendedModelInfo, factors *routing.ScoringFactors) float64 {
	return factors.Reliability*0.4 + factors.Cost*0.2 + factors.Quality*0.2 + factors.Latency*0.1 + factors.Capabilities*0.1
}

// Registry of all strategies
type Registry struct {
	strategies map[routing.RoutingStrategy]Strategy
}

func NewRegistry() *Registry {
	r := &Registry{
		strategies: make(map[routing.RoutingStrategy]Strategy),
	}

	// Register all strategies
	r.strategies[routing.StrategyLeastLatency] = &LeastLatencyStrategy{}
	r.strategies[routing.StrategyCheapest] = &CheapestStrategy{}
	r.strategies[routing.StrategyQualityFirst] = &QualityFirstStrategy{}
	r.strategies[routing.StrategyBalanced] = &BalancedStrategy{}
	r.strategies[routing.StrategyCostAware] = &CostAwareStrategy{}
	r.strategies[routing.StrategyThroughput] = &ThroughputStrategy{}
	r.strategies[routing.StrategyReliability] = &ReliabilityStrategy{}
	r.strategies[routing.StrategyAdaptive] = &AdaptiveStrategy{}
	r.strategies[routing.StrategyLoadBalanced] = &LoadBalancedStrategy{
		RequestCounts: make(map[string]int64),
	}
	r.strategies[routing.StrategyGeographic] = &GeographicStrategy{}
	r.strategies[routing.StrategyCompliance] = &ComplianceStrategy{}
	r.strategies[routing.StrategySecurityFirst] = &SecurityFirstStrategy{}
	r.strategies[routing.StrategyOfflineFriendly] = &OfflineFriendlyStrategy{}

	return r
}

func (r *Registry) Get(name routing.RoutingStrategy) Strategy {
	return r.strategies[name]
}

func (r *Registry) List() []Strategy {
	result := make([]Strategy, 0, len(r.strategies))
	for _, s := range r.strategies {
		result = append(result, s)
	}
	return result
}

// Min returns the minimum of two floats
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two floats
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Clamp clamps a value between min and max
func clamp(val, min, max float64) float64 {
	return max(min(val, max), min)
}

// Ensure math is used
var _ = math.Min
