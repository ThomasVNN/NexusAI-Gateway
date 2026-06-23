package compression

import (
	"sort"
	"sync"
	"time"
)

// Pipeline orchestrates multiple compression engines in priority order.
// This is the main entry point for the compression system.
//
// ENG-9209: Stacked pipeline orchestrator
type Pipeline struct {
	mu      sync.RWMutex
	engines map[string]CompressionEngine
	mode    PipelineMode
	session *DedupEngine // Shared session deduplication
}

// NewPipeline creates a new compression pipeline with default engines
func NewPipeline() *Pipeline {
	p := &Pipeline{
		engines: make(map[string]CompressionEngine),
		mode:    ModeLite,
		session: NewDedupEngine(),
	}

	// Register all engines
	p.registerEngine(NewLiteEngine())
	p.registerEngine(p.session)
	p.registerEngine(NewRTKEngine())
	p.registerEngine(NewCavemanEngine())
	p.registerEngine(NewCCREngine())
	p.registerEngine(NewHeadroomEngine())
	p.registerEngine(NewLLMLinguaEngine())
	p.registerEngine(NewAggressiveEngine())
	p.registerEngine(NewUltraEngine())

	return p
}

// NewPipelineWithConfig creates a new pipeline with specific configuration
func NewPipelineWithConfig(config PipelineConfig) *Pipeline {
	p := NewPipeline()
	p.SetMode(config.Mode)

	// Enable/disable engines based on config
	if len(config.EnabledEngines) > 0 {
		for name, engine := range p.engines {
			enabled := false
			for _, e := range config.EnabledEngines {
				if e == name {
					enabled = true
					break
				}
			}
			engine.SetEnabled(enabled)
		}
	}

	return p
}

// registerEngine adds an engine to the pipeline
func (p *Pipeline) registerEngine(engine CompressionEngine) {
	p.engines[engine.Name()] = engine
}

// SetMode sets the compression mode
func (p *Pipeline) SetMode(mode PipelineMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode

	// Enable/disable engines based on mode
	enabledEngines := DefaultEngines(mode)

	for name, engine := range p.engines {
		enabled := false
		for _, e := range enabledEngines {
			if e == name {
				enabled = true
				break
			}
		}
		engine.SetEnabled(enabled)
	}
}

// GetMode returns the current pipeline mode
func (p *Pipeline) GetMode() PipelineMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// Compress applies the compression pipeline to the input
func (p *Pipeline) Compress(input string) *CompressionResult {
	return p.CompressWithSession(input, "default")
}

// CompressWithSession applies compression with session tracking
func (p *Pipeline) CompressWithSession(input string, sessionID string) *CompressionResult {
	start := time.Now()

	if input == "" {
		return &CompressionResult{
			Original:   "",
			Compressed: "",
			EngineStats: make(map[string]EngineStats),
			TotalSaved: 0,
			SavingsPct: 0,
			Mode:       p.GetMode(),
			LatencyMs:  0,
		}
	}

	originalLen := len(input)
	engineStats := make(map[string]EngineStats)
	current := input
	totalSaved := 0

	// Get sorted engines
	engines := p.getSortedEngines()

	// Apply each engine in order
	for _, engine := range engines {
		if !engine.IsEnabled() {
			continue
		}

		compressed, saved, _ := engine.Compress(current)

		// Track stats
		engineStats[engine.Name()] = engine.Stats()

		// Apply session dedup if this is the dedup engine
		if engine.Name() == "dedup" {
			compressed, saved, _ = p.session.CompressWithSession(current, sessionID)
			engineStats[engine.Name()] = engine.Stats()
		}

		if saved > 0 {
			totalSaved += saved
			current = compressed
		}
	}

	// Calculate savings percentage
	savingsPct := 0.0
	if originalLen > 0 {
		savingsPct = float64(totalSaved) / float64(originalLen) * 100
	}

	latency := time.Since(start)

	return &CompressionResult{
		Original:    input,
		Compressed:  current,
		EngineStats: engineStats,
		TotalSaved:  totalSaved,
		SavingsPct:  savingsPct,
		Mode:        p.GetMode(),
		LatencyMs:   float64(latency.Nanoseconds()) / 1e6,
	}
}

// CompressMulti applies compression to multiple inputs
func (p *Pipeline) CompressMulti(inputs []string) []*CompressionResult {
	results := make([]*CompressionResult, len(inputs))
	for i, input := range inputs {
		results[i] = p.Compress(input)
	}
	return results
}

// getSortedEngines returns engines sorted by priority
func (p *Pipeline) getSortedEngines() []CompressionEngine {
	engines := make([]CompressionEngine, 0, len(p.engines))
	for _, engine := range p.engines {
		engines = append(engines, engine)
	}

	sort.Slice(engines, func(i, j int) bool {
		return engines[i].Priority() < engines[j].Priority()
	})

	return engines
}

// GetEngine returns an engine by name
func (p *Pipeline) GetEngine(name string) (CompressionEngine, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	engine, exists := p.engines[name]
	return engine, exists
}

// GetAllEngines returns all registered engines
func (p *Pipeline) GetAllEngines() map[string]CompressionEngine {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]CompressionEngine)
	for k, v := range p.engines {
		result[k] = v
	}
	return result
}

// GetEnabledEngines returns only enabled engines
func (p *Pipeline) GetEnabledEngines() []CompressionEngine {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var enabled []CompressionEngine
	for _, engine := range p.engines {
		if engine.IsEnabled() {
			enabled = append(enabled, engine)
		}
	}

	sort.Slice(enabled, func(i, j int) bool {
		return enabled[i].Priority() < enabled[j].Priority()
	})

	return enabled
}

// SetEngineEnabled enables or disables a specific engine
func (p *Pipeline) SetEngineEnabled(name string, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if engine, exists := p.engines[name]; exists {
		engine.SetEnabled(enabled)
	}
}

// GetTotalStats returns aggregated statistics for all engines
func (p *Pipeline) GetTotalStats() map[string]EngineStats {
	stats := make(map[string]EngineStats)
	for name, engine := range p.engines {
		stats[name] = engine.Stats()
	}
	return stats
}

// ResetAllStats clears statistics for all engines
func (p *Pipeline) ResetAllStats() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, engine := range p.engines {
		switch e := engine.(type) {
		case *LiteEngine:
			e.ResetStats()
		case *DedupEngine:
			e.ResetStats()
		case *RTKEngine:
			e.ResetStats()
		case *CCREngine:
			e.ResetStats()
		case *HeadroomEngine:
			e.ResetStats()
		case *CavemanEngine:
			e.ResetStats()
		case *LLMLinguaEngine:
			e.ResetStats()
		case *AggressiveEngine:
			e.ResetStats()
		case *UltraEngine:
			e.ResetStats()
		}
	}
}

// ClearSession clears the session deduplication data
func (p *Pipeline) ClearSession(sessionID string) {
	p.session.RemoveSession(sessionID)
}

// ClearAllSessions clears all session data
func (p *Pipeline) ClearAllSessions() {
	p.session.ClearOldSessions()
}

// EstimateSavings provides a quick estimate of potential savings
func (p *Pipeline) EstimateSavings(input string) (int, float64) {
	if input == "" {
		return 0, 0
	}

	originalLen := len(input)
	mode := p.GetMode()
	engines := DefaultEngines(mode)

	// Rough estimation based on mode
	var estimatedRatio float64
	switch mode {
	case ModeOff:
		estimatedRatio = 0
	case ModeLite:
		estimatedRatio = 0.05 // 5%
	case ModeStandard:
		estimatedRatio = 0.25 // 25%
	case ModeAggressive:
		estimatedRatio = 0.50 // 50%
	case ModeUltra:
		estimatedRatio = 0.70 // 70%
	case ModeStacked:
		estimatedRatio = 0.60 // 60%
	default:
		estimatedRatio = 0.05
	}

	// Adjust based on actual enabled engines
	enabledCount := len(engines)
	if enabledCount < 5 {
		estimatedRatio *= float64(enabledCount) / 5.0
	}

	estimatedSavings := int(float64(originalLen) * estimatedRatio)
	return estimatedSavings, estimatedRatio * 100
}

// PipelineInfo returns information about the pipeline configuration
type PipelineInfo struct {
	Mode           PipelineMode `json:"mode"`
	TotalEngines   int          `json:"total_engines"`
	EnabledEngines int          `json:"enabled_engines"`
	EngineList     []string     `json:"engine_list"`
}

// GetInfo returns pipeline information
func (p *Pipeline) GetInfo() PipelineInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var engineList []string
	enabledCount := 0

	for name, engine := range p.engines {
		engineList = append(engineList, name)
		if engine.IsEnabled() {
			enabledCount++
		}
	}

	sort.Strings(engineList)

	return PipelineInfo{
		Mode:           p.mode,
		TotalEngines:   len(p.engines),
		EnabledEngines: enabledCount,
		EngineList:     engineList,
	}
}

// Ensure Pipeline is complete
var _ interface{} = (*Pipeline)(nil)

// Pipeline presets
const (
	PresetOff        = ModeOff
	PresetLite       = ModeLite
	PresetStandard   = ModeStandard
	PresetAggressive  = ModeAggressive
	PresetUltra      = ModeUltra
	PresetStacked    = ModeStacked
)

// GetPresetModes returns all available preset modes
func GetPresetModes() []PipelineMode {
	return []PipelineMode{
		ModeOff,
		ModeLite,
		ModeStandard,
		ModeAggressive,
		ModeUltra,
		ModeStacked,
	}
}

// GetPresetDescription returns a description of a preset mode
func GetPresetDescription(mode PipelineMode) string {
	switch mode {
	case ModeOff:
		return "No compression applied"
	case ModeLite:
		return "Always-on baseline compression (3-10% savings)"
	case ModeStandard:
		return "Standard optimization with dedup, RTK, and caveman (15-30% savings)"
	case ModeAggressive:
		return "High compression with all engines except ultra (40-60% savings)"
	case ModeUltra:
		return "Maximum compression with ultra engine (60-80% savings)"
	case ModeStacked:
		return "All engines including LLMLingua enabled (60-80% savings)"
	default:
		return "Unknown mode"
	}
}

// Utility functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
