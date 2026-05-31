package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store implements an in-memory store for all memory types
type Store struct {
	mu             sync.RWMutex
	episodic       map[string]*Memory
	semantic       map[string]*Memory
	procedural     map[string]*Memory
	embeddingIndex map[string][]float32
	sessionEvents  map[string][]Event
	maxMemoryAge   time.Duration
	gcInterval     time.Duration
}

// Event represents an episodic event
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewStore creates a new in-memory memory store
func NewStore(maxMemoryAge time.Duration) *Store {
	s := &Store{
		episodic:       make(map[string]*Memory),
		semantic:       make(map[string]*Memory),
		procedural:     make(map[string]*Memory),
		embeddingIndex: make(map[string][]float32),
		sessionEvents:  make(map[string][]Event),
		maxMemoryAge:   maxMemoryAge,
		gcInterval:     5 * time.Minute,
	}
	return s
}

// Store stores a memory entry
func (s *Store) Store(ctx context.Context, memory Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if memory.ID == "" {
		memory.ID = generateID()
	}
	memory.CreatedAt = time.Now()
	memory.UpdatedAt = time.Now()

	memoryCopy := memory

	switch memoryCopy.Type {
	case MemoryTypeEpisodic:
		s.episodic[memoryCopy.ID] = &memoryCopy
		if sessionID, ok := memoryCopy.Metadata["session_id"].(string); ok {
			s.addEvent(sessionID, Event{
				ID:          memoryCopy.ID,
				Type:        "memory_stored",
				Description: memoryCopy.Content,
				Timestamp:   memoryCopy.CreatedAt,
				Metadata:    memoryCopy.Metadata,
			})
		}
	case MemoryTypeSemantic:
		s.semantic[memoryCopy.ID] = &memoryCopy
		if len(memoryCopy.Embedding) > 0 {
			s.embeddingIndex[memoryCopy.ID] = memoryCopy.Embedding
		}
	case MemoryTypeProcedural:
		s.procedural[memoryCopy.ID] = &memoryCopy
	default:
		return ErrInvalidMemoryType
	}

	return nil
}

// Retrieve retrieves memories based on a query
func (s *Store) Retrieve(ctx context.Context, query MemoryQuery) ([]MemoryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query.Limit <= 0 {
		query.Limit = 10
	}

	var results []MemoryResult

	switch {
	case len(query.Embedding) > 0:
		results = s.retrieveByEmbedding(query)
	case query.SessionID != "":
		results = s.retrieveBySession(query)
	default:
		results = s.retrieveByContent(query)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

func (s *Store) retrieveByEmbedding(query MemoryQuery) []MemoryResult {
	var results []MemoryResult

	for id, embedding := range s.embeddingIndex {
		var memory *Memory
		switch {
		case s.episodic[id] != nil:
			memory = s.episodic[id]
		case s.semantic[id] != nil:
			memory = s.semantic[id]
		case s.procedural[id] != nil:
			memory = s.procedural[id]
		}

		if memory == nil || memory.IsExpired() {
			continue
		}

		if query.Type != "" && memory.Type != query.Type {
			continue
		}

		similarity := cosineSimilarity(query.Embedding, embedding)
		if similarity >= query.MinSimilarity {
			results = append(results, MemoryResult{
				Memory:     *memory,
				Similarity: similarity,
				Score:      similarity,
			})
		}
	}

	return results
}

func (s *Store) retrieveBySession(query MemoryQuery) []MemoryResult {
	var results []MemoryResult

	for _, memory := range s.episodic {
		if memory.IsExpired() {
			continue
		}
		if sessionID, ok := memory.Metadata["session_id"].(string); ok && sessionID == query.SessionID {
			results = append(results, MemoryResult{
				Memory:     *memory,
				Similarity: 1.0,
				Score:      1.0,
			})
		}
	}

	return results
}

func (s *Store) retrieveByContent(query MemoryQuery) []MemoryResult {
	var results []MemoryResult
	queryLower := strings.ToLower(query.Query)

	memoryMap := map[string]*Memory{
		"episodic":   nil,
		"semantic":   nil,
		"procedural": nil,
	}

	for id, m := range s.episodic {
		memoryMap[id] = m
	}
	for id, m := range s.semantic {
		memoryMap[id] = m
	}
	for id, m := range s.procedural {
		memoryMap[id] = m
	}

	for _, memory := range memoryMap {
		if memory == nil || memory.IsExpired() {
			continue
		}

		if query.Type != "" && memory.Type != query.Type {
			continue
		}

		contentLower := strings.ToLower(memory.Content)
		if strings.Contains(contentLower, queryLower) {
			score := float32(len(queryLower)) / float32(len(contentLower))
			results = append(results, MemoryResult{
				Memory:     *memory,
				Similarity: score,
				Score:      score,
			})
		}
	}

	return results
}

// Delete deletes a memory entry by ID
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.episodic[id]; ok {
		delete(s.episodic, id)
		delete(s.sessionEvents, id)
		return nil
	}
	if _, ok := s.semantic[id]; ok {
		delete(s.semantic, id)
		delete(s.embeddingIndex, id)
		return nil
	}
	if _, ok := s.procedural[id]; ok {
		delete(s.procedural, id)
		return nil
	}

	return ErrMemoryNotFound
}

// List lists memories by type
func (s *Store) List(ctx context.Context, memoryType MemoryType, limit int) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var memories []Memory
	var source map[string]*Memory

	switch memoryType {
	case MemoryTypeEpisodic:
		source = s.episodic
	case MemoryTypeSemantic:
		source = s.semantic
	case MemoryTypeProcedural:
		source = s.procedural
	default:
		return nil, ErrInvalidMemoryType
	}

	for _, m := range source {
		if !m.IsExpired() {
			memories = append(memories, *m)
			if len(memories) >= limit {
				break
			}
		}
	}

	return memories, nil
}

// GetByID retrieves a memory entry by ID
func (s *Store) GetByID(ctx context.Context, id string) (*Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if memory, ok := s.episodic[id]; ok {
		if !memory.IsExpired() {
			return memory, nil
		}
	}
	if memory, ok := s.semantic[id]; ok {
		if !memory.IsExpired() {
			return memory, nil
		}
	}
	if memory, ok := s.procedural[id]; ok {
		if !memory.IsExpired() {
			return memory, nil
		}
	}

	return nil, ErrMemoryNotFound
}

// Update updates an existing memory entry
func (s *Store) Update(ctx context.Context, memory Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	memory.UpdatedAt = time.Now()

	switch memory.Type {
	case MemoryTypeEpisodic:
		if _, ok := s.episodic[memory.ID]; !ok {
			return ErrMemoryNotFound
		}
		s.episodic[memory.ID] = &memory
	case MemoryTypeSemantic:
		if _, ok := s.semantic[memory.ID]; !ok {
			return ErrMemoryNotFound
		}
		s.semantic[memory.ID] = &memory
		if len(memory.Embedding) > 0 {
			s.embeddingIndex[memory.ID] = memory.Embedding
		}
	case MemoryTypeProcedural:
		if _, ok := s.procedural[memory.ID]; !ok {
			return ErrMemoryNotFound
		}
		s.procedural[memory.ID] = &memory
	default:
		return ErrInvalidMemoryType
	}

	return nil
}

// Clear clears all memories of a given type
func (s *Store) Clear(ctx context.Context, memoryType MemoryType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch memoryType {
	case MemoryTypeEpisodic:
		s.episodic = make(map[string]*Memory)
		s.sessionEvents = make(map[string][]Event)
	case MemoryTypeSemantic:
		s.semantic = make(map[string]*Memory)
		s.embeddingIndex = make(map[string][]float32)
	case MemoryTypeProcedural:
		s.procedural = make(map[string]*Memory)
	default:
		return ErrInvalidMemoryType
	}

	return nil
}

// GetSessionEvents retrieves all events for a session
func (s *Store) GetSessionEvents(ctx context.Context, sessionID string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, ok := s.sessionEvents[sessionID]
	if !ok {
		return []Event{}, nil
	}

	return events, nil
}

func (s *Store) addEvent(sessionID string, event Event) {
	s.sessionEvents[sessionID] = append(s.sessionEvents[sessionID], event)
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(sqrt(normA)) * float32(sqrt(normB)))
}

func sqrt(n float32) float32 {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// StoreStats represents memory store statistics
type StoreStats struct {
	EpisodicCount   int `json:"episodic_count"`
	SemanticCount   int `json:"semantic_count"`
	ProceduralCount int `json:"procedural_count"`
	TotalCount      int `json:"total_count"`
}

// GetStats returns statistics about the memory store
func (s *Store) GetStats(ctx context.Context) (StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StoreStats{
		EpisodicCount:   len(s.episodic),
		SemanticCount:   len(s.semantic),
		ProceduralCount: len(s.procedural),
		TotalCount:      len(s.episodic) + len(s.semantic) + len(s.procedural),
	}, nil
}

// Errors
var (
	ErrMemoryStore = errors.New("memory store error")
)
