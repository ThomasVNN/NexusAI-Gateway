package compression

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// UltraEngine provides maximum compression using aggressive heuristics.
// This engine applies the most aggressive compression techniques,
// suitable only for situations where context size is critical.
//
// ENG-9208: Aggressive + Ultra engines
type UltraEngine struct {
	enabled bool
	stats   EngineStats
	mu      sync.RWMutex

	// Pre-compiled patterns
	patterns []*UltraPattern
}

// UltraPattern defines an ultra-compression pattern
type UltraPattern struct {
	Name     string
	Pattern  *regexp.Regexp
	Replace  string
	Priority int
}

// NewUltraEngine creates a new ultra compression engine
func NewUltraEngine() *UltraEngine {
	patterns := []*UltraPattern{
		// Maximum whitespace collapse
		{Name: "all_whitespace", Pattern: regexp.MustCompile(`\s+`), Replace: " ", Priority: 1},

		// Line-based truncation (keep structure)
		{Name: "truncate_long_lines", Pattern: regexp.MustCompile(`^.{200,}$`), Replace: "[LONG LINE]", Priority: 5},

		// Word reduction
		{Name: "long_words", Pattern: regexp.MustCompile(`\b\w{30,}\b`), Replace: "[LONG]", Priority: 10},

		// Number reduction
		{Name: "all_numbers", Pattern: regexp.MustCompile(`\b\d+\b`), Replace: "N", Priority: 15},

		// UUID removal
		{Name: "uuid", Pattern: regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), Replace: "[ID]", Priority: 15},

		// Hash removal
		{Name: "long_hash", Pattern: regexp.MustCompile(`\b[0-9a-f]{32,}\b`), Replace: "[HASH]", Priority: 20},

		// Email removal
		{Name: "email", Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), Replace: "[EMAIL]", Priority: 20},

		// URL removal
		{Name: "url", Pattern: regexp.MustCompile(`https?://[^\s]+`), Replace: "[URL]", Priority: 20},

		// IP address removal
		{Name: "ip", Pattern: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), Replace: "[IP]", Priority: 25},

		// Path truncation
		{Name: "long_path", Pattern: regexp.MustCompile(`[/\\][^/\\]{50,}`), Replace: "/[PATH]", Priority: 25},

		// Code cleanup
		{Name: "all_comments", Pattern: regexp.MustCompile(`(?://|#|/\*|\*/)[^\n]*`), Replace: "", Priority: 30},
		{Name: "empty_braces", Pattern: regexp.MustCompile(`\{\s*\}`), Replace: "{}", Priority: 30},
		{Name: "empty_brackets", Pattern: regexp.MustCompile(`\[\s*\]`), Replace: "[]", Priority: 30},

		// Semantic reduction
		{Name: "very_common_words", Pattern: regexp.MustCompile(`\b(the|a|an|and|or|but|in|on|at|to|for|of|with|by)\b`), Replace: "", Priority: 35},

		// Type annotations
		{Name: "typescript_types", Pattern: regexp.MustCompile(`:\s*(string|number|boolean|any|void|never|unknown)\b`), Replace: "", Priority: 40},
	}

	return &UltraEngine{
		enabled: false, // Disabled by default - extreme compression
		stats: EngineStats{
			Name: "ultra",
		},
		patterns: patterns,
	}
}

// Name returns the engine name
func (e *UltraEngine) Name() string {
	return "ultra"
}

// Priority returns the execution priority
func (e *UltraEngine) Priority() int {
	return 70
}

// IsEnabled returns whether the engine is active
func (e *UltraEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *UltraEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Compress applies ultra compression
func (e *UltraEngine) Compress(input string) (string, int, error) {
	if input == "" {
		return "", 0, nil
	}

	e.mu.RLock()
	enabled := e.enabled
	e.mu.RUnlock()

	if !enabled {
		return input, 0, nil
	}

	originalLen := len(input)
	compressed := input

	// Apply patterns in priority order
	for _, pattern := range e.patterns {
		compressed = pattern.Pattern.ReplaceAllString(compressed, pattern.Replace)
	}

	// Final aggressive cleanup
	compressed = strings.TrimSpace(compressed)
	compressed = strings.ReplaceAll(compressed, "  ", " ")

	saved := originalLen - len(compressed)

	// Update stats
	e.mu.Lock()
	e.stats.Invocations++
	e.stats.TotalInputLen += int64(originalLen)
	e.stats.TotalOutputLen += int64(len(compressed))
	e.stats.TotalSaved += int64(saved)
	e.stats.LastUsed = time.Now()
	e.mu.Unlock()

	return compressed, saved, nil
}

// Stats returns the engine statistics
func (e *UltraEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *UltraEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "ultra"}
}

// PatternCount returns the number of patterns
func (e *UltraEngine) PatternCount() int {
	return len(e.patterns)
}

// Ensure UltraEngine implements CompressionEngine
var _ CompressionEngine = (*UltraEngine)(nil)
