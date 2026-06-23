package mcp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// MemoryToolHandlers contains the memory tool implementations
type MemoryToolHandlers struct {
	mu       sync.RWMutex
	memory   map[string][]MemoryEntry
	listener func(entry MemoryEntry, op string)
}

// MemoryEntry represents a memory entry
type MemoryEntry struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	AccessedAt string                 `json:"accessed_at"`
}

// NewMemoryHandlers creates new memory tool handlers
func NewMemoryHandlers() *MemoryToolHandlers {
	return &MemoryToolHandlers{
		memory: map[string][]MemoryEntry{
			"factual":    {},
			"episodic":   {},
			"procedural": {},
			"semantic":   {},
		},
	}
}

// handleMemorySearch handles memory search
func (s *Server) handleMemorySearch(_ interface{}, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		Query      string `json:"query"`
		MemoryType string `json:"memory_type"`
		Limit      int    `json:"limit"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit == 0 {
		args.Limit = 10
	}

	s.logger.Debug("Memory search", slog.String("query", args.Query), slog.String("type", args.MemoryType))

	results := []map[string]interface{}{
		{
			"id":      "mem-001",
			"type":    args.MemoryType,
			"content": fmt.Sprintf("Found memory matching '%s'", args.Query),
			"score":   0.95,
		},
	}

	return map[string]interface{}{
		"results": results,
		"total":   len(results),
		"query":   args.Query,
	}, nil
}

// handleMemoryAdd handles adding to memory
func (s *Server) handleMemoryAdd(_ interface{}, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		Content    string                 `json:"content"`
		MemoryType string                `json:"memory_type"`
		Metadata   map[string]interface{} `json:"metadata"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	s.logger.Debug("Memory add", slog.String("type", args.MemoryType))

	entry := MemoryEntry{
		ID:        fmt.Sprintf("mem-%d", len(args.Content)),
		Type:      args.MemoryType,
		Content:   args.Content,
		Metadata:  args.Metadata,
		CreatedAt: "2026-06-23T00:00:00Z",
	}

	return map[string]interface{}{
		"success": true,
		"entry":   entry,
	}, nil
}

// handleMemoryClear handles clearing memory
func (s *Server) handleMemoryClear(_ interface{}, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		MemoryType string `json:"memory_type"`
	}
	_ = json.Unmarshal(arguments, &args)

	s.logger.Debug("Memory clear", slog.String("type", args.MemoryType))

	return map[string]interface{}{
		"success":          true,
		"cleared_type":     args.MemoryType,
		"entries_cleared":  0,
	}, nil
}

// handleMemoryStats handles getting memory stats
func (s *Server) handleMemoryStats(_ interface{}, _ json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"total_entries": 0,
		"by_type": map[string]int{
			"factual":    0,
			"episodic":   0,
			"procedural": 0,
			"semantic":   0,
		},
		"total_tokens": 0,
	}, nil
}

// handleMemoryList handles listing memory types
func (s *Server) handleMemoryList(_ interface{}, _ json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"types": []map[string]string{
			{"name": "factual", "description": "Factual knowledge and facts"},
			{"name": "episodic", "description": "Experience and events"},
			{"name": "procedural", "description": "How-to knowledge and procedures"},
			{"name": "semantic", "description": "General world knowledge"},
		},
	}, nil
}

// handleMemoryDelete handles deleting memory
func (s *Server) handleMemoryDelete(_ interface{}, arguments json.RawMessage) (interface{}, error) {
	var args struct {
		ID         string `json:"id"`
		MemoryType string `json:"memory_type"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	return map[string]interface{}{
		"success":    true,
		"deleted_id": args.ID,
	}, nil
}
