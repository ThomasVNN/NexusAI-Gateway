package compression

import (
	"time"
)

// CompressionEngine is the base interface all compression engines must implement.
// Each engine provides a specific compression strategy with a priority order.
type CompressionEngine interface {
	// Name returns the unique identifier for this engine
	Name() string

	// Compress applies the engine's compression strategy to the input.
	// Returns the compressed string and the number of characters saved.
	Compress(input string) (string, int, error)

	// Priority determines the execution order (lower = earlier)
	Priority() int

	// IsEnabled returns whether this engine is active
	IsEnabled() bool

	// SetEnabled enables or disables the engine
	SetEnabled(enabled bool)

	// Stats returns the engine statistics
	Stats() EngineStats
}

// EngineStats tracks compression statistics for an engine
type EngineStats struct {
	Name           string        `json:"name"`
	Invocations    int64         `json:"invocations"`
	TotalInputLen  int64         `json:"total_input_len"`
	TotalOutputLen int64         `json:"total_output_len"`
	TotalSaved    int64         `json:"total_saved"`
	AvgLatency    time.Duration `json:"avg_latency_ns"`
	LastUsed      time.Time     `json:"last_used"`
}

// SavingsPercent returns the compression ratio as a percentage
func (s *EngineStats) SavingsPercent() float64 {
	if s.TotalInputLen == 0 {
		return 0
	}
	return float64(s.TotalSaved) / float64(s.TotalInputLen) * 100
}

// PipelineConfig defines the configuration for the compression pipeline
type PipelineConfig struct {
	Mode      PipelineMode `json:"mode"`
	MaxTokens int          `json:"max_tokens"` // Target max tokens
	EnabledEngines []string `json:"enabled_engines"`
}

// PipelineMode defines the preset compression modes
type PipelineMode string

const (
	ModeOff        PipelineMode = "off"
	ModeLite       PipelineMode = "lite"        // Always-on baseline
	ModeStandard   PipelineMode = "standard"    // Default optimization
	ModeAggressive PipelineMode = "aggressive"  // High compression
	ModeUltra      PipelineMode = "ultra"       // Maximum compression
	ModeStacked    PipelineMode = "stacked"    // All engines enabled
)

// CompressionResult holds the result of pipeline compression
type CompressionResult struct {
	Original    string            `json:"original"`
	Compressed  string            `json:"compressed"`
	EngineStats map[string]EngineStats `json:"engine_stats"`
	TotalSaved  int               `json:"total_saved"`
	SavingsPct  float64           `json:"savings_pct"`
	Mode        PipelineMode      `json:"mode"`
	LatencyMs   float64           `json:"latency_ms"`
}

// CompressedChunk represents a content-addressed storage unit
type CompressedChunk struct {
	SHA256     string    `json:"sha256"`
	Content    string    `json:"content"`
	Length     int       `json:"length"`
	Compressed string    `json:"compressed"`
	CreatedAt  time.Time `json:"created_at"`
	Hits       int64     `json:"hits"`
}

// SessionDeduplication tracks content seen across conversation turns
type SessionDedup struct {
	SessionID  string            `json:"session_id"`
	SeenHashes map[string]string // SHA256 hash -> normalized content
	SeenLines  map[string]bool   // Cache for quick lookups
}

// DefaultEngines returns the default engine set for each mode
func DefaultEngines(mode PipelineMode) []string {
	switch mode {
	case ModeOff:
		return []string{}
	case ModeLite:
		return []string{"lite"}
	case ModeStandard:
		return []string{"lite", "dedup", "rtk", "caveman"}
	case ModeAggressive:
		return []string{"lite", "dedup", "rtk", "caveman", "headroom", "ccr", "aggressive"}
	case ModeUltra:
		return []string{"lite", "dedup", "rtk", "caveman", "headroom", "ccr", "aggressive", "ultra"}
	case ModeStacked:
		return []string{"lite", "dedup", "rtk", "caveman", "headroom", "ccr", "llmlingua", "aggressive", "ultra"}
	default:
		return []string{"lite"}
	}
}
