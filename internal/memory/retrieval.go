package memory

import (
	"log/slog"
	"slices"
	"sync"
)

// RetrievalEngine handles hybrid retrieval across all tiers
type RetrievalEngine struct {
	tiers   []MemoryStore
	rrfK    int
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewRetrievalEngine creates a new hybrid retrieval engine
func NewRetrievalEngine(rrfK int) *RetrievalEngine {
	if rrfK <= 0 {
		rrfK = 60 // Default RRF constant
	}
	return &RetrievalEngine{
		tiers:  make([]MemoryStore, 0),
		rrfK:   rrfK,
		logger: slog.Default(),
	}
}

// RegisterTier adds a memory tier to the engine
func (e *RetrievalEngine) RegisterTier(tier MemoryStore) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tiers = append(e.tiers, tier)
	e.logger.Info("registered memory tier", "tier", tier.Name())
}

// HybridSearch performs hybrid search across all tiers using RRF
func (e *RetrievalEngine) HybridSearch(query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}
	if opts.RRF_K > 0 {
		opts.RRF_K = e.rrfK
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Collect results from all tiers
	type tierResult struct {
		tier    MemoryStore
		matches []*MemoryMatch
		err     error
	}

	resultChan := make(chan tierResult, len(e.tiers))
	var wg sync.WaitGroup

	for _, tier := range e.tiers {
		// Check if tier is in sources list
		if len(opts.Sources) > 0 && !slices.Contains(opts.Sources, tier.Name()) {
			continue
		}

		wg.Add(1)
		go func(t MemoryStore) {
			defer wg.Done()
			matches, err := t.Search(nil, query, opts)
			resultChan <- tierResult{tier: t, matches: matches, err: err}
		}(tier)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect all results
	allResults := make(map[string]*MemoryMatch)
	
	for tr := range resultChan {
		if tr.err != nil {
			e.logger.Warn("tier search failed", "tier", tr.tier.Name(), "error", tr.err)
			continue
		}
		
		for rank, match := range tr.matches {
			match.Rank = rank
			match.Source = tr.tier.Name()
			
			// Merge results, keeping highest score
			if existing, ok := allResults[match.Memory.ID]; ok {
				existing.Score += match.Score
			} else {
				allResults[match.Memory.ID] = match
			}
		}
	}

	// Apply RRF scoring
	rrfScores := e.calculateRRF(allResults, opts.RRF_K)

	// Sort by RRF score
	results := make([]*MemoryMatch, 0, len(rrfScores))
	for _, match := range allResults {
		if rrfScore, ok := rrfScores[match.Memory.ID]; ok {
			match.Score = rrfScore
			results = append(results, match)
		}
	}

	// Sort descending by score
	slices.SortFunc(results, func(a, b *MemoryMatch) int {
		if b.Score > a.Score {
			return 1
		}
		return -1
	})

	// Apply limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// calculateRRF applies Reciprocal Rank Fusion
func (e *RetrievalEngine) calculateRRF(allResults map[string]*MemoryMatch, k int) map[string]float64 {
	rrfScores := make(map[string]float64)
	
	// RRF formula: score = Σ(1/(k+rank))
	// We normalize existing scores and apply RRF weighting
	for id, match := range allResults {
		// Normalize original score to 0-1 range
		normalizedScore := match.Score
		if normalizedScore > 1.0 {
			normalizedScore = 1.0
		}
		
		// Apply RRF with rank
		rrfScore := normalizedScore / float64(k+match.Rank+1)
		rrfScores[id] = rrfScore
	}
	
	return rrfScores
}

// SearchSingleTier searches a single tier
func (e *RetrievalEngine) SearchSingleTier(tierName, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, tier := range e.tiers {
		if tier.Name() == tierName {
			return tier.Search(nil, query, opts)
		}
	}

	return nil, nil
}

// Tiers returns all registered tiers
func (e *RetrievalEngine) Tiers() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	tiers := make([]string, len(e.tiers))
	for i, t := range e.tiers {
		tiers[i] = t.Name()
	}
	return tiers
}
