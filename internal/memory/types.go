package memory

import (
	"time"
)

// MemoryType represents the type/category of a memory
type MemoryType string

const (
	MemoryFactual    MemoryType = "factual"    // Facts, statements
	MemoryEpisodic   MemoryType = "episodic"   // Events, experiences
	MemoryProcedural MemoryType = "procedural" // How-to, steps
	MemorySemantic   MemoryType = "semantic"   // Concepts, meanings
)

// Memory represents a single memory entry
type Memory struct {
	ID        string                 `json:"id"`
	Type      MemoryType             `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float32              `json:"embedding,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
}

// MemoryMatch represents a memory with its relevance score
type MemoryMatch struct {
	Memory *Memory `json:"memory"`
	Score  float64 `json:"score"`
	Rank   int     `json:"rank"`
	Source string  `json:"source"` // "fts5", "vec", "qdrant", "hybrid"
}

// DateRange represents a time range for filtering
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// SearchOptions contains options for memory search
type SearchOptions struct {
	Limit      int        `json:"limit"`
	MemoryType MemoryType `json:"memory_type,omitempty"`
	SessionID  string     `json:"session_id,omitempty"`
	UserID     string     `json:"user_id,omitempty"`
	DateRange  *DateRange `json:"date_range,omitempty"`
	Hybrid     bool       `json:"hybrid"`      // Use FTS5 + Vector combined
	RRF_K      int        `json:"rrf_k"`      // Reciprocal Rank Fusion constant
	Threshold  float64    `json:"threshold"`   // Minimum relevance score
	Sources    []string   `json:"sources"`     // ["fts5", "vec", "qdrant"]
}

// DefaultSearchOptions returns sensible defaults
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		Limit: 10,
		RRF_K: 60,
		Hybrid: true,
		Sources: []string{"fts5", "vec", "qdrant"},
	}
}

// MemoryStore is the unified interface for memory operations
type MemoryStore interface {
	// Search finds memories matching a query
	Search(ctx interface{}, query string, opts *SearchOptions) ([]*MemoryMatch, error)

	// Add stores a new memory
	Add(ctx interface{}, memory *Memory) error

	// Delete removes a memory by ID
	Delete(ctx interface{}, id string) error

	// Get retrieves a memory by ID
	Get(ctx interface{}, id string) (*Memory, error)

	// List returns all memories for a session
	List(ctx interface{}, sessionID string, opts *SearchOptions) ([]*Memory, error)

	// Name returns the tier name
	Name() string
}
