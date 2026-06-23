package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ExtractionPattern represents a regex pattern for memory extraction
type ExtractionPattern struct {
	Type    MemoryType
	Pattern *regexp.Regexp
	Weight  float64
}

// ExtractionEngine handles non-blocking regex-based memory extraction
type ExtractionEngine struct {
	patterns []*ExtractionPattern
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewExtractionEngine creates a new extraction engine with default patterns
func NewExtractionEngine() *ExtractionEngine {
	patterns := []*ExtractionPattern{
		// Factual patterns - facts and statements
		{Type: MemoryFactual, Pattern: regexp.MustCompile(`(?i)(\w+(?:\s+\w+){0,5}\s+is\s+(?:a\s+)?(?:\w+(?:\s+\w+){0,5}))`), Weight: 1.0},
		{Type: MemoryFactual, Pattern: regexp.MustCompile(`(?i)(\w+(?:\s+\w+){0,3}\s+was\s+(?:a\s+)?(?:\w+(?:\s+\w+){0,3}))`), Weight: 1.0},
		{Type: MemoryFactual, Pattern: regexp.MustCompile(`(?i)(the\s+\w+(?:\s+\w+){0,5}\s+(?:is|are|was|were)\s+(?:\w+(?:\s+\w+){0,5}))`), Weight: 0.9},
		{Type: MemoryFactual, Pattern: regexp.MustCompile(`(?i)(fact:\s*[^\n]+)`), Weight: 1.0},
		{Type: MemoryFactual, Pattern: regexp.MustCompile(`(?i)(remember\s+that\s+[^\n]+)`), Weight: 0.8},
		
		// Procedural patterns - how-to, steps
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(to\s+\w+(?:\s+\w+){0,3},?\s+(?:first|you\s+need\s+to)\s+\w+(?:\s+\w+){0,5})`), Weight: 1.0},
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(step\s+\d+[:\.]\s*[^\n]+)`), Weight: 1.0},
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(first[,\s]+then[,\s]+finally[^\n]+)`), Weight: 0.9},
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(how\s+to\s+\w+(?:\s+\w+){0,10})`), Weight: 0.8},
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(you\s+can\s+(?:do|use|make|create)\s+\w+(?:\s+\w+){0,5})`), Weight: 0.7},
		{Type: MemoryProcedural, Pattern: regexp.MustCompile(`(?i)(here(?:\'s| is)\s+how[^\n]+)`), Weight: 0.8},
		
		// Episodic patterns - events, experiences
		{Type: MemoryEpisodic, Pattern: regexp.MustCompile(`(?i)(yesterday|today|earlier|previously|lately)[^\n]+`), Weight: 0.8},
		{Type: MemoryEpisodic, Pattern: regexp.MustCompile(`(?i)(i\s+(?:did|made|created|tried|worked\s+on)\s+\w+(?:\s+\w+){0,5})`), Weight: 0.9},
		{Type: MemoryEpisodic, Pattern: regexp.MustCompile(`(?i)(we\s+(?:discussed|talked\s+about|worked\s+on)\s+\w+(?:\s+\w+){0,5})`), Weight: 0.9},
		{Type: MemoryEpisodic, Pattern: regexp.MustCompile(`(?i)(in\s+the\s+last\s+(?:session|meeting|conversation)[^\n]+)`), Weight: 0.8},
		
		// Semantic patterns - concepts, meanings
		{Type: MemorySemantic, Pattern: regexp.MustCompile(`(?i)(the\s+concept\s+of\s+\w+(?:\s+\w+){0,5})`), Weight: 1.0},
		{Type: MemorySemantic, Pattern: regexp.MustCompile(`(?i)(essentially[,\s]+(?:it\s+)?(?:is|means|refers\s+to)[^\n]+)`), Weight: 0.9},
		{Type: MemorySemantic, Pattern: regexp.MustCompile(`(?i)(in\s+other\s+words[,\s]+[^\n]+)`), Weight: 0.8},
		{Type: MemorySemantic, Pattern: regexp.MustCompile(`(?i)(basically[,\s]+(?:it\s+)?(?:is|means)[^\n]+)`), Weight: 0.8},
		{Type: MemorySemantic, Pattern: regexp.MustCompile(`(?i)(the\s+(?:meaning|definition)\s+of\s+\w+(?:\s+\w+){0,3})`), Weight: 0.9},
	}

	return &ExtractionEngine{
		patterns: patterns,
		logger:   slog.Default(),
	}
}

// AddPattern adds a custom extraction pattern
func (e *ExtractionEngine) AddPattern(memType MemoryType, pattern string, weight float64) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.patterns = append(e.patterns, &ExtractionPattern{
		Type:    memType,
		Pattern: re,
		Weight:  weight,
	})

	return nil
}

// ExtractMemories extracts memories from content using non-blocking goroutines
func (e *ExtractionEngine) ExtractMemories(content string, sessionID, userID string) []*Memory {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return e.extractMemoriesWithContext(ctx, content, sessionID, userID)
}

// extractMemoriesWithContext extracts memories with context for cancellation
func (e *ExtractionEngine) extractMemoriesWithContext(ctx context.Context, content string, sessionID, userID string) []*Memory {
	memories := make(chan *Memory, 100)
	var wg sync.WaitGroup

	// Run pattern matching in parallel goroutines
	for _, p := range e.patterns {
		wg.Add(1)
		go func(pattern *ExtractionPattern) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			matches := pattern.Pattern.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}

				// Skip very short matches
				if len(match[1]) < 10 {
					continue
				}

				mem := &Memory{
					ID:        generateMemoryID(),
					Type:      pattern.Type,
					Content:   strings.TrimSpace(match[1]),
					Metadata: map[string]interface{}{
						"weight":    pattern.Weight,
						"extracted": time.Now().UTC(),
					},
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
					SessionID: sessionID,
					UserID:    userID,
				}

				select {
				case <-ctx.Done():
					return
				case memories <- mem:
				}
			}
		}(p)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(memories)
	}()

	// Collect results (non-blocking)
	result := make([]*Memory, 0)
	for mem := range memories {
		result = append(result, mem)
	}

	return result
}

// ExtractAndScore extracts memories and assigns semantic scores
func (e *ExtractionEngine) ExtractAndScore(content string, sessionID, userID string, scoreFunc func(string) float64) []*Memory {
	memories := e.ExtractMemories(content, sessionID, userID)

	for _, mem := range memories {
		if scoreFunc != nil {
			mem.Metadata["semantic_score"] = scoreFunc(mem.Content)
		}
	}

	return memories
}

// ExtractStructured extracts structured information from content
func (e *ExtractionEngine) ExtractStructured(content string, sessionID, userID string) ([]*Memory, map[string]interface{}) {
	memories := e.ExtractMemories(content, sessionID, userID)
	structured := make(map[string]interface{})

	// Group by type
	typeCounts := make(map[MemoryType]int)
	for _, mem := range memories {
		typeCounts[mem.Type]++
	}

	structured["counts_by_type"] = typeCounts
	structured["total_extracted"] = len(memories)
	structured["extraction_time"] = time.Now().UTC()

	return memories, structured
}

// generateMemoryID generates a unique memory ID
func generateMemoryID() string {
	return "mem_" + time.Now().UTC().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Prevent duplicate generation
	}
	return string(b)
}

// MemoryToJSON converts a memory to JSON string
func MemoryToJSON(mem *Memory) string {
	data, err := json.Marshal(mem)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// JSONToMemory converts JSON string to memory
func JSONToMemory(data string) (*Memory, error) {
	var mem Memory
	err := json.Unmarshal([]byte(data), &mem)
	return &mem, err
}
