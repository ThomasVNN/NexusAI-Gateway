package memory

import (
	"context"
	"errors"
	"time"
)

// MemoryType represents the type of memory
type MemoryType string

const (
	MemoryTypeEpisodic   MemoryType = "episodic"
	MemoryTypeSemantic   MemoryType = "semantic"
	MemoryTypeProcedural MemoryType = "procedural"
)

// Memory represents a generic memory entry
type Memory struct {
	ID        string                 `json:"id"`
	Type      MemoryType             `json:"type"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

// MemoryQuery represents a query for memory retrieval
type MemoryQuery struct {
	Query         string     `json:"query"`
	Embedding     []float32  `json:"embedding,omitempty"`
	Type          MemoryType `json:"type,omitempty"`
	SessionID     string     `json:"session_id,omitempty"`
	Limit         int        `json:"limit"`
	MinSimilarity float32    `json:"min_similarity"`
	TimeRange     *TimeRange `json:"time_range,omitempty"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MemoryResult represents a memory retrieval result
type MemoryResult struct {
	Memory     Memory  `json:"memory"`
	Similarity float32 `json:"similarity"`
	Score      float32 `json:"score"`
}

// MemoryStore defines the interface for memory storage
type MemoryStore interface {
	Store(ctx context.Context, memory Memory) error
	Retrieve(ctx context.Context, query MemoryQuery) ([]MemoryResult, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, memoryType MemoryType, limit int) ([]Memory, error)
	GetByID(ctx context.Context, id string) (*Memory, error)
	Update(ctx context.Context, memory Memory) error
	Clear(ctx context.Context, memoryType MemoryType) error
}

// ErrMemoryNotFound is returned when a memory entry is not found
var ErrMemoryNotFound = errors.New("memory not found")

// ErrInvalidMemoryType is returned when an invalid memory type is provided
var ErrInvalidMemoryType = errors.New("invalid memory type")

// NewMemory creates a new memory entry
func NewMemory(memoryType MemoryType, content string, embedding []float32) *Memory {
	now := time.Now()
	return &Memory{
		ID:        generateID(),
		Type:      memoryType,
		Content:   content,
		Embedding: embedding,
		Metadata:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// WithMetadata adds metadata to the memory
func (m *Memory) WithMetadata(key string, value interface{}) *Memory {
	m.Metadata[key] = value
	return m
}

// WithSessionID adds a session ID to the metadata
func (m *Memory) WithSessionID(sessionID string) *Memory {
	m.Metadata["session_id"] = sessionID
	return m
}

// WithTTL sets a time-to-live for the memory
func (m *Memory) WithTTL(ttl time.Duration) *Memory {
	expiresAt := time.Now().Add(ttl)
	m.ExpiresAt = &expiresAt
	return m
}

// IsExpired checks if the memory has expired
func (m *Memory) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*m.ExpiresAt)
}

// generateID generates a unique ID for a memory entry
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}
