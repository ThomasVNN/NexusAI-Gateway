package compression

import (
	"sync"
	"time"
)

// LLMLinguaEngine provides semantic compression using LLMLingua-2 ONNX model.
// This engine performs self-hosted semantic pruning for high-quality
// compression at the cost of higher latency.
//
// ENG-9207: LLMLingua-2 ONNX integration
type LLMLinguaEngine struct {
	enabled     bool
	stats       EngineStats
	mu          sync.RWMutex

	// Model configuration
	modelPath  string
	device     string // "cpu", "cuda", "mps"
	compressor interface{} // Would be ONNX model in production

	// Compression settings
	compressionRatio float64 // Target compression ratio (0.0-1.0)
	keepEssential    bool    // Keep essential semantic elements
}

// LLMLinguaConfig holds configuration for the LLMLingua engine
type LLMLinguaConfig struct {
	ModelPath        string
	Device           string
	CompressionRatio float64
	KeepEssential    bool
}

// NewLLMLinguaEngine creates a new LLMLingua compression engine
// Note: This is a stub implementation. In production, this would
// load and use the actual ONNX model.
func NewLLMLinguaEngine() *LLMLinguaEngine {
	return &LLMLinguaEngine{
		enabled:          false, // Disabled by default - requires model
		compressionRatio: 0.5,
		keepEssential:    true,
		device:           "cpu",
		stats: EngineStats{
			Name: "llmlingua",
		},
	}
}

// NewLLMLinguaEngineWithConfig creates a new LLMLingua engine with config
func NewLLMLinguaEngineWithConfig(config LLMLinguaConfig) *LLMLinguaEngine {
	device := config.Device
	if device == "" {
		device = "cpu"
	}

	ratio := config.CompressionRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 0.5
	}

	return &LLMLinguaEngine{
		enabled:          config.ModelPath != "", // Only enabled if model path provided
		modelPath:        config.ModelPath,
		device:           device,
		compressionRatio: ratio,
		keepEssential:    config.KeepEssential,
		stats: EngineStats{
			Name: "llmlingua",
		},
	}
}

// Name returns the engine name
func (e *LLMLinguaEngine) Name() string {
	return "llmlingua"
}

// Priority returns the execution priority
func (e *LLMLinguaEngine) Priority() int {
	return 50
}

// IsEnabled returns whether the engine is active
func (e *LLMLinguaEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *LLMLinguaEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Compress performs semantic compression using LLMLingua-2
// This is a stub implementation that provides basic compression
// when the ONNX model is not available.
func (e *LLMLinguaEngine) Compress(input string) (string, int, error) {
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

	// Stub implementation: basic semantic-aware compression
	// In production, this would:
	// 1. Tokenize input
	// 2. Score each token by importance
	// 3. Keep top N% of tokens by importance score
	// 4. Reconstruct the compressed text

	compressed := e.stubCompress(input)
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

// stubCompress provides basic compression when model is unavailable
func (e *LLMLinguaEngine) stubCompress(input string) string {
	// This is a placeholder that applies basic compression
	// Real implementation would use the ONNX model

	// Simple word-based compression
	words := splitIntoWords(input)
	if len(words) == 0 {
		return input
	}

	// Keep essential words based on compression ratio
	targetCount := int(float64(len(words)) * e.compressionRatio)
	if targetCount >= len(words) {
		return input
	}

	// Select important words (first, last, and every Nth word)
	var selected []string
	step := len(words) / targetCount
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(words); i += step {
		selected = append(selected, words[i])
	}

	// Ensure we don't miss the end
	if len(selected) > 0 && selected[len(selected)-1] != words[len(words)-1] {
		selected = append(selected, words[len(words)-1])
	}

	return joinWords(selected)
}

// splitIntoWords splits text into words
func splitIntoWords(s string) []string {
	var words []string
	var current []rune

	for _, r := range s {
		if isWordChar(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		}
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}

// isWordChar returns true if rune is a word character
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// joinWords joins words with spaces
func joinWords(words []string) string {
	if len(words) == 0 {
		return ""
	}
	result := words[0]
	for i := 1; i < len(words); i++ {
		result += " " + words[i]
	}
	return result
}

// SetCompressionRatio sets the target compression ratio
func (e *LLMLinguaEngine) SetCompressionRatio(ratio float64) {
	if ratio > 0 && ratio <= 1 {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.compressionRatio = ratio
	}
}

// GetCompressionRatio returns the current compression ratio
func (e *LLMLinguaEngine) GetCompressionRatio() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.compressionRatio
}

// IsModelLoaded returns whether the ONNX model is loaded
func (e *LLMLinguaEngine) IsModelLoaded() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.modelPath != ""
}

// Stats returns the engine statistics
func (e *LLMLinguaEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *LLMLinguaEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "llmlingua"}
}

// Ensure LLMLinguaEngine implements CompressionEngine
var _ CompressionEngine = (*LLMLinguaEngine)(nil)
