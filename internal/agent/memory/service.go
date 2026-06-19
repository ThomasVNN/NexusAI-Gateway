package memory

import (
	"context"
	"sync"
)

// EmbeddingService defines the interface for embedding generation
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingConfig holds configuration for the embedding service
type EmbeddingConfig struct {
	Model    string
	Endpoint string
	APIKey   string
}

// Service provides high-level memory operations
type Service struct {
	store            MemoryStore
	embeddingService EmbeddingService
	mu               sync.RWMutex
}

// NewService creates a new memory service
func NewService(store MemoryStore, embeddingService EmbeddingService) *Service {
	return &Service{
		store:            store,
		embeddingService: embeddingService,
	}
}

// Store stores a memory entry
func (s *Service) Store(ctx context.Context, memory *Memory) error {
	if s.embeddingService != nil && len(memory.Embedding) == 0 && memory.Type == MemoryTypeSemantic {
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, memory.Content)
		if err != nil {
			return err
		}
		memory.Embedding = embedding
	}
	return s.store.Store(ctx, *memory)
}

// Retrieve retrieves memories based on a query
func (s *Service) Retrieve(ctx context.Context, query MemoryQuery) ([]MemoryResult, error) {
	if s.embeddingService != nil && len(query.Embedding) == 0 && query.Query != "" {
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, query.Query)
		if err != nil {
			return nil, err
		}
		query.Embedding = embedding
	}
	return s.store.Retrieve(ctx, query)
}

// Delete deletes a memory entry
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List lists memories by type
func (s *Service) List(ctx context.Context, memoryType MemoryType, limit int) ([]Memory, error) {
	return s.store.List(ctx, memoryType, limit)
}

// GetByID retrieves a memory entry by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Memory, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates an existing memory entry
func (s *Service) Update(ctx context.Context, memory Memory) error {
	if s.embeddingService != nil && len(memory.Embedding) == 0 && memory.Type == MemoryTypeSemantic {
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, memory.Content)
		if err != nil {
			return err
		}
		memory.Embedding = embedding
	}
	return s.store.Update(ctx, memory)
}

// Clear clears all memories of a given type
func (s *Service) Clear(ctx context.Context, memoryType MemoryType) error {
	return s.store.Clear(ctx, memoryType)
}

// StoreEpisodic stores an episodic memory
func (s *Service) StoreEpisodic(ctx context.Context, sessionID, content string, metadata map[string]interface{}) error {
	memory := NewMemory(MemoryTypeEpisodic, content, nil)
	memory.Metadata = metadata
	if sessionID != "" {
		memory.WithSessionID(sessionID)
	}
	return s.store.Store(ctx, *memory)
}

// StoreSemantic stores a semantic memory with embedding
func (s *Service) StoreSemantic(ctx context.Context, content string, embedding []float32, metadata map[string]interface{}) error {
	memory := NewMemory(MemoryTypeSemantic, content, embedding)
	memory.Metadata = metadata
	return s.store.Store(ctx, *memory)
}

// StoreProcedural stores a procedural memory
func (s *Service) StoreProcedural(ctx context.Context, content string, metadata map[string]interface{}) error {
	memory := NewMemory(MemoryTypeProcedural, content, nil)
	memory.Metadata = metadata
	return s.store.Store(ctx, *memory)
}

// RetrieveContext retrieves relevant context from all memory types
func (s *Service) RetrieveContext(ctx context.Context, query string, limit int) ([]MemoryResult, error) {
	memoryQuery := MemoryQuery{
		Query:         query,
		Limit:         limit,
		MinSimilarity: 0.5,
	}
	return s.store.Retrieve(ctx, memoryQuery)
}

// GetStats returns statistics about the memory store
func (s *Service) GetStats(ctx context.Context) (StoreStats, error) {
	if store, ok := s.store.(*Store); ok {
		return store.GetStats(ctx)
	}
	return StoreStats{}, nil
}
