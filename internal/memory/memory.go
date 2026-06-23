package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// MemoryConfig holds configuration for all memory tiers
type MemoryConfig struct {
	Tier0 FTS5Config `json:"tier0"`
	Tier1 VecConfig  `json:"tier1"`
	Tier2 QdrantConfig `json:"tier2"`
	
	// Global settings
	DefaultLimit int  `json:"default_limit"`
	RRF_K       int  `json:"rrf_k"`
}

// MemorySystem is the unified memory system that coordinates all tiers
type MemorySystem struct {
	tiers     []MemoryStore
	retriever *RetrievalEngine
	extractor *ExtractionEngine
	config    *MemoryConfig
	logger    *slog.Logger
	mu        sync.RWMutex
}

// NewMemorySystem creates a new unified memory system
func NewMemorySystem(config *MemoryConfig) (*MemorySystem, error) {
	if config == nil {
		config = &MemoryConfig{
			DefaultLimit: 10,
			RRF_K:       60,
		}
	}

	ms := &MemorySystem{
		config:    config,
		retriever: NewRetrievalEngine(config.RRF_K),
		extractor: NewExtractionEngine(),
		logger:    slog.Default(),
	}

	// Initialize tiers
	if config.Tier0.Enabled {
		fts5, err := NewFTS5Memory(&config.Tier0)
		if err != nil {
			ms.logger.Warn("FTS5 tier init failed", "error", err)
		} else {
			ms.tiers = append(ms.tiers, fts5)
			ms.retriever.RegisterTier(fts5)
			ms.logger.Info("FTS5 tier initialized")
		}
	}

	if config.Tier1.Enabled {
		vec, err := NewVecMemory(&config.Tier1)
		if err != nil {
			ms.logger.Warn("Vec tier init failed", "error", err)
		} else {
			ms.tiers = append(ms.tiers, vec)
			ms.retriever.RegisterTier(vec)
			ms.logger.Info("Vec tier initialized")
		}
	}

	if config.Tier2.Enabled {
		qdrant, err := NewQdrantMemory(&config.Tier2)
		if err != nil {
			ms.logger.Warn("Qdrant tier init failed", "error", err)
		} else {
			ms.tiers = append(ms.tiers, qdrant)
			ms.retriever.RegisterTier(qdrant)
			ms.logger.Info("Qdrant tier initialized")
		}
	}

	if len(ms.tiers) == 0 {
		// Default to FTS5 if nothing configured
		fts5, err := NewFTS5Memory(&config.Tier0)
		if err != nil {
			return nil, err
		}
		ms.tiers = append(ms.tiers, fts5)
		ms.retriever.RegisterTier(fts5)
		ms.logger.Info("Default FTS5 tier initialized")
	}

	return ms, nil
}

// Search performs hybrid search across all tiers
func (ms *MemorySystem) Search(ctx context.Context, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	return ms.retriever.HybridSearch(query, opts)
}

// Add stores a new memory in all tiers
func (ms *MemorySystem) Add(ctx context.Context, memory *Memory) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now().UTC()
	}
	if memory.UpdatedAt.IsZero() {
		memory.UpdatedAt = time.Now().UTC()
	}

	var lastErr error
	for _, tier := range ms.tiers {
		if err := tier.Add(ctx, memory); err != nil {
			lastErr = err
			ms.logger.Warn("failed to add to tier", "tier", tier.Name(), "error", err)
		}
	}

	return lastErr
}

// AddAsync stores a memory asynchronously
func (ms *MemorySystem) AddAsync(ctx context.Context, memory *Memory, callback func(error)) {
	go func() {
		err := ms.Add(ctx, memory)
		if callback != nil {
			callback(err)
		}
	}()
}

// Delete removes a memory from all tiers
func (ms *MemorySystem) Delete(ctx context.Context, id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var lastErr error
	for _, tier := range ms.tiers {
		if err := tier.Delete(ctx, id); err != nil {
			lastErr = err
			ms.logger.Warn("failed to delete from tier", "tier", tier.Name(), "error", err)
		}
	}

	return lastErr
}

// Get retrieves a memory from the first available tier
func (ms *MemorySystem) Get(ctx context.Context, id string) (*Memory, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, tier := range ms.tiers {
		mem, err := tier.Get(ctx, id)
		if err == nil && mem != nil {
			return mem, nil
		}
	}

	return nil, nil
}

// List returns all memories for a session
func (ms *MemorySystem) List(ctx context.Context, sessionID string, opts *SearchOptions) ([]*Memory, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	seen := make(map[string]bool)
	var memories []*Memory

	for _, tier := range ms.tiers {
		tierMemories, err := tier.List(ctx, sessionID, opts)
		if err != nil {
			ms.logger.Warn("failed to list from tier", "tier", tier.Name(), "error", err)
			continue
		}

		for _, mem := range tierMemories {
			if !seen[mem.ID] {
				seen[mem.ID] = true
				memories = append(memories, mem)
			}
		}
	}

	return memories, nil
}

// ExtractAndStore extracts memories from content and stores them
func (ms *MemorySystem) ExtractAndStore(ctx context.Context, content, sessionID, userID string) ([]*Memory, error) {
	// Extract memories
	extracted := ms.extractor.ExtractMemories(content, sessionID, userID)
	if len(extracted) == 0 {
		return nil, nil
	}

	// Store all extracted memories
	var stored []*Memory
	for _, mem := range extracted {
		if err := ms.Add(ctx, mem); err != nil {
			ms.logger.Warn("failed to store extracted memory", "id", mem.ID, "error", err)
		} else {
			stored = append(stored, mem)
		}
	}

	return stored, nil
}

// ExtractAndStoreAsync extracts and stores memories asynchronously
func (ms *MemorySystem) ExtractAndStoreAsync(ctx context.Context, content, sessionID, userID string, callback func([]*Memory, error)) {
	go func() {
		memories, err := ms.ExtractAndStore(ctx, content, sessionID, userID)
		if callback != nil {
			callback(memories, err)
		}
	}()
}

// SearchSingleTier searches a specific tier
func (ms *MemorySystem) SearchSingleTier(tierName string, ctx context.Context, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	return ms.retriever.SearchSingleTier(tierName, query, opts)
}

// Tiers returns all registered tier names
func (ms *MemorySystem) Tiers() []string {
	return ms.retriever.Tiers()
}

// Tier returns a specific tier by name
func (ms *MemorySystem) Tier(name string) MemoryStore {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, tier := range ms.tiers {
		if tier.Name() == name {
			return tier
		}
	}

	return nil
}

// Close closes all tiers
func (ms *MemorySystem) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var lastErr error
	for _, tier := range ms.tiers {
		if closer, ok := tier.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				lastErr = err
				ms.logger.Warn("failed to close tier", "tier", tier.Name(), "error", err)
			}
		}
	}

	return lastErr
}

// Stats returns statistics about the memory system
func (ms *MemorySystem) Stats() *SystemStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	stats := &SystemStats{
		Tiers: make(map[string]*TierStats),
	}

	for _, tier := range ms.tiers {
		stats.Tiers[tier.Name()] = &TierStats{
			Name: tier.Name(),
		}
	}

	return stats
}

// SystemStats holds statistics about the memory system
type SystemStats struct {
	Tiers map[string]*TierStats
}

// TierStats holds statistics about a tier
type TierStats struct {
	Name  string
	Count int64
}

// serializeMetadata converts metadata map to JSON string
func serializeMetadata(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// parseMetadata parses metadata JSON String into a map
func parseMetadata(data string, mem *Memory) {
	if data == "" || data == "{}" {
		mem.Metadata = make(map[string]interface{})
		return
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		mem.Metadata = make(map[string]interface{})
		return
	}

	mem.Metadata = m
}
