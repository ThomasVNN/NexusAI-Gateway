package compression

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// LiteEngine provides baseline compression with always-on optimizations.
// This engine is the default for all compression modes and provides
// 3-10% savings with <1ms latency.
//
// ENG-9201: Lite compression engine
type LiteEngine struct {
	enabled bool
	stats   EngineStats
	mu      sync.RWMutex

	// Pre-compiled patterns for performance
	multiSpace   *regexp.Regexp
	multiNewline *regexp.Regexp
	trimmed      *regexp.Regexp
}

// NewLiteEngine creates a new Lite compression engine
func NewLiteEngine() *LiteEngine {
	return &LiteEngine{
		enabled: true,
		stats: EngineStats{
			Name: "lite",
		},
		multiSpace:   regexp.MustCompile(`[ \t]+`),
		multiNewline: regexp.MustCompile(`\n{3,}`),
		trimmed:      regexp.MustCompile(`^\s+|\s+$`),
	}
}

// Name returns the engine name
func (e *LiteEngine) Name() string {
	return "lite"
}

// Priority returns the execution priority (0 = first)
func (e *LiteEngine) Priority() int {
	return 0
}

// IsEnabled returns whether the engine is active
func (e *LiteEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *LiteEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Compress applies lite compression optimizations:
// - Trims leading/trailing whitespace
// - Collapses multiple spaces/tabs to single space
// - Collapses 3+ consecutive newlines to 2
// - Removes empty lines at start/end
func (e *LiteEngine) Compress(input string) (string, int, error) {
	if input == "" {
		return "", 0, nil
	}

	originalLen := len(input)

	// Step 1: Trim leading/trailing whitespace
	compressed := strings.TrimSpace(input)

	// Step 2: Collapse multiple spaces/tabs
	compressed = e.multiSpace.ReplaceAllString(compressed, " ")

	// Step 3: Collapse multiple newlines (but preserve paragraph breaks)
	compressed = e.multiNewline.ReplaceAllString(compressed, "\n\n")

	// Step 4: Remove trailing whitespace from each line
	lines := strings.Split(compressed, "\n")
	for i, line := range lines {
		lines[i] = e.trimmed.ReplaceAllString(line, "")
	}
	compressed = strings.Join(lines, "\n")

	// Step 5: Final trim
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
func (e *LiteEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *LiteEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "lite"}
}

// CompactJSON performs basic JSON compaction for lite mode
func (e *LiteEngine) CompactJSON(input string) (string, int, error) {
	if input == "" {
		return "", 0, nil
	}

	originalLen := len(input)

	// Check if it looks like JSON
	if !strings.HasPrefix(strings.TrimSpace(input), "{") && !strings.HasPrefix(strings.TrimSpace(input), "[") {
		return input, 0, nil
	}

	// Only compact if it's reasonably formatted (has newlines or extra spaces)
	hasFormatting := strings.Contains(input, "\n") || strings.Contains(input, "  ")

	if !hasFormatting {
		return input, 0, nil
	}

	// Use the same lite optimizations for JSON
	compressed := strings.TrimSpace(input)
	compressed = e.multiSpace.ReplaceAllString(compressed, " ")
	compressed = e.multiNewline.ReplaceAllString(compressed, "")

	saved := originalLen - len(compressed)
	return compressed, saved, nil
}

// Ensure LiteEngine implements CompressionEngine
var _ CompressionEngine = (*LiteEngine)(nil)
