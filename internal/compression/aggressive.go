package compression

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// AggressiveEngine provides high-compression using heuristics and summarization.
// This engine combines multiple aggressive techniques for maximum compression,
// suitable for contexts near the token limit.
//
// ENG-9208: Aggressive + Ultra engines
type AggressiveEngine struct {
	enabled bool
	stats   EngineStats
	mu      sync.RWMutex

	// Pre-compiled patterns
	patterns []*AggressivePattern
}

// AggressivePattern defines a compression pattern
type AggressivePattern struct {
	Name      string
	Pattern   *regexp.Regexp
	Replace   string
	Priority  int
}

// NewAggressiveEngine creates a new aggressive compression engine
func NewAggressiveEngine() *AggressiveEngine {
	patterns := []*AggressivePattern{
		// Very aggressive whitespace removal
		{Name: "all_whitespace", Pattern: regexp.MustCompile(`\s+`), Replace: " ", Priority: 5},

		// Code-specific optimizations
		{Name: "arrow_function", Pattern: regexp.MustCompile(`function\s+\w+\s*\(\)\s*`), Replace: "fn()", Priority: 10},
		{Name: "async_await", Pattern: regexp.MustCompile(`\basync\s+\b`), Replace: "", Priority: 15},
		{Name: "const_let", Pattern: regexp.MustCompile(`\bconst\s+`), Replace: "c ", Priority: 15},
		{Name: "let_var", Pattern: regexp.MustCompile(`\blet\s+`), Replace: "l ", Priority: 15},
		{Name: "function_keyword", Pattern: regexp.MustCompile(`\bfunction\s+`), Replace: "fn ", Priority: 15},

		// Comment removal
		{Name: "single_comment", Pattern: regexp.MustCompile(`//.*$`), Replace: "", Priority: 20},
		{Name: "multi_comment", Pattern: regexp.MustCompile(`/\*[\s\S]*?\*/`), Replace: "", Priority: 20},
		{Name: "python_comment", Pattern: regexp.MustCompile(`#.*$`), Replace: "", Priority: 20},

		// String optimization
		{Name: "long_string_trunc", Pattern: regexp.MustCompile(`"(.{100,})"`), Replace: `"..."`, Priority: 25},

		// URL shortening (in text context)
		{Name: "url_strip", Pattern: regexp.MustCompile(`https?://[^\s]+`), Replace: "[URL]", Priority: 25},

		// Number optimization
		{Name: "large_numbers", Pattern: regexp.MustCompile(`\b\d{6,}\b`), Replace: "XXXXXX", Priority: 30},
	}

	return &AggressiveEngine{
		enabled: false, // Disabled by default - use sparingly
		stats: EngineStats{
			Name: "aggressive",
		},
		patterns: patterns,
	}
}

// Name returns the engine name
func (e *AggressiveEngine) Name() string {
	return "aggressive"
}

// Priority returns the execution priority
func (e *AggressiveEngine) Priority() int {
	return 60
}

// IsEnabled returns whether the engine is active
func (e *AggressiveEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *AggressiveEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Compress applies aggressive compression
func (e *AggressiveEngine) Compress(input string) (string, int, error) {
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

	// Apply patterns
	for _, pattern := range e.patterns {
		compressed = pattern.Pattern.ReplaceAllString(compressed, pattern.Replace)
	}

	// Final cleanup
	compressed = strings.TrimSpace(compressed)

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
func (e *AggressiveEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *AggressiveEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "aggressive"}
}

// PatternCount returns the number of patterns
func (e *AggressiveEngine) PatternCount() int {
	return len(e.patterns)
}

// Ensure AggressiveEngine implements CompressionEngine
var _ CompressionEngine = (*AggressiveEngine)(nil)
