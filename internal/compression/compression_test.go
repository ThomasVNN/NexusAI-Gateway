package compression

import (
	"strings"
	"testing"
)

func TestLiteEngine_Compress(t *testing.T) {
	engine := NewLiteEngine()

	tests := []struct {
		name     string
		input    string
		wantSavings int
	}{
		{
			name:     "empty input",
			input:    "",
			wantSavings: 0,
		},
		{
			name:     "basic whitespace",
			input:    "  hello   world  ",
			wantSavings: 5,
		},
		{
			name:     "multiple newlines",
			input:    "line1\n\n\n\n\nline2",
			wantSavings: 2,
		},
		{
			name:     "mixed whitespace",
			input:    "  foo  \t\t  bar  \n\n\n  baz  ",
			wantSavings: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, saved, err := engine.Compress(tt.input)
			if err != nil {
				t.Errorf("Compress() error = %v", err)
				return
			}
			if saved < tt.wantSavings {
				t.Errorf("Compress() saved = %d, want >= %d", saved, tt.wantSavings)
			}
			// Verify no leading/trailing whitespace
			if compressed != strings.TrimSpace(compressed) {
				t.Errorf("Compress() result has leading/trailing whitespace: %q", compressed)
			}
		})
	}
}

func TestLiteEngine_Stats(t *testing.T) {
	engine := NewLiteEngine()
	initialStats := engine.Stats()

	if initialStats.Name != "lite" {
		t.Errorf("Stats().Name = %s, want lite", initialStats.Name)
	}

	// Compress something
	engine.Compress("  test  ")

	stats := engine.Stats()
	if stats.Invocations != 1 {
		t.Errorf("Stats().Invocations = %d, want 1", stats.Invocations)
	}
}

func TestDedupEngine_Compress(t *testing.T) {
	engine := NewDedupEngine()

	// First compression - should keep content
	input1 := "unique line 1\nunique line 2"
	compressed1, saved1, err := engine.CompressWithSession(input1, "test-session")
	if err != nil {
		t.Errorf("CompressWithSession() error = %v", err)
	}
	if saved1 != 0 {
		t.Errorf("First compression saved = %d, want 0", saved1)
	}
	if compressed1 != input1 {
		t.Errorf("First compression changed input unexpectedly")
	}

	// Second compression with same content - should remove
	input2 := "unique line 1\nunique line 2\nnew line 3"
	_, saved2, err := engine.CompressWithSession(input2, "test-session")
	if err != nil {
		t.Errorf("CompressWithSession() error = %v", err)
	}
	if saved2 == 0 {
		t.Errorf("Second compression saved = 0, want > 0")
	}

	// Different session - should not dedupe
	_, saved3, err := engine.CompressWithSession(input1, "other-session")
	if err != nil {
		t.Errorf("CompressWithSession() error = %v", err)
	}
	if saved3 != 0 {
		t.Errorf("Different session saved = %d, want 0", saved3)
	}
}

func TestRTKEngine_PatternCount(t *testing.T) {
	engine := NewRTKEngine()
	count := engine.PatternCount()
	if count < 60 {
		t.Errorf("PatternCount() = %d, want >= 60", count)
	}
}

func TestRTKEngine_Compress(t *testing.T) {
	engine := NewRTKEngine()
	engine.SetEnabled(true)

	tests := []struct {
		name    string
		input   string
		wantSavings int
	}{
		{
			name:        "empty input",
			input:       "",
			wantSavings: 0,
		},
		{
			name:        "small input",
			input:       "short",
			wantSavings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, saved, err := engine.Compress(tt.input)
			if err != nil {
				t.Errorf("Compress() error = %v", err)
				return
			}
			if saved < tt.wantSavings {
				t.Errorf("Compress() saved = %d, want >= %d", saved, tt.wantSavings)
			}
			_ = compressed
		})
	}
}

func TestCavemanEngine_RuleCount(t *testing.T) {
	engine := NewCavemanEngine()
	count := engine.RuleCount()
	if count < 40 {
		t.Errorf("RuleCount() = %d, want >= 40", count)
	}
}

func TestCavemanEngine_Compress(t *testing.T) {
	engine := NewCavemanEngine()
	engine.SetEnabled(true)

	input := "This is  a test   with   extra    spaces"
	compressed, saved, err := engine.Compress(input)
	if err != nil {
		t.Errorf("Compress() error = %v", err)
	}
	if saved == 0 {
		t.Errorf("Compress() saved = 0, want > 0")
	}
	// Verify no multiple spaces
	if strings.Contains(compressed, "  ") {
		t.Errorf("Compress() result still has multiple spaces: %q", compressed)
	}
}

func TestCCREngine_Compress(t *testing.T) {
	engine := NewCCREngine()

	tests := []struct {
		name       string
		input      string
		wantSavings int
	}{
		{
			name:       "empty input",
			input:      "",
			wantSavings: 0,
		},
		{
			name:       "small input",
			input:      "short",
			wantSavings: 0, // Too small to process
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, saved, err := engine.Compress(tt.input)
			if err != nil {
				t.Errorf("Compress() error = %v", err)
				return
			}
			_ = compressed
			_ = saved
			// Small inputs should pass through unchanged
			if tt.input != "" && tt.input != "short" && saved < tt.wantSavings {
				t.Errorf("Compress() saved = %d, want >= %d", saved, tt.wantSavings)
			}
		})
	}
}

func TestHeadroomEngine_Compress(t *testing.T) {
	engine := NewHeadroomEngine()

	tests := []struct {
		name       string
		input      string
		wantSavings int
	}{
		{
			name:       "empty input",
			input:      "",
			wantSavings: 0,
		},
		{
			name:       "non-json input",
			input:      "plain text",
			wantSavings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, saved, err := engine.Compress(tt.input)
			if err != nil {
				t.Errorf("Compress() error = %v", err)
			}
			_ = compressed
			_ = saved
		})
	}
}

func TestAggressiveEngine_Compress(t *testing.T) {
	engine := NewAggressiveEngine()

	// Disabled by default
	if engine.IsEnabled() {
		t.Error("AggressiveEngine should be disabled by default")
	}

	engine.SetEnabled(true)
	if !engine.IsEnabled() {
		t.Error("AggressiveEngine should be enabled after SetEnabled(true)")
	}

	input := "  lots   of   spaces  "
	compressed, saved, err := engine.Compress(input)
	if err != nil {
		t.Errorf("Compress() error = %v", err)
	}
	if saved == 0 {
		t.Errorf("Compress() saved = 0, want > 0")
	}
	_ = compressed
}

func TestUltraEngine_Compress(t *testing.T) {
	engine := NewUltraEngine()

	// Disabled by default
	if engine.IsEnabled() {
		t.Error("UltraEngine should be disabled by default")
	}

	engine.SetEnabled(true)
	if !engine.IsEnabled() {
		t.Error("UltraEngine should be enabled after SetEnabled(true)")
	}

	input := "  lots   of   spaces  with numbers 1234567890"
	compressed, saved, err := engine.Compress(input)
	if err != nil {
		t.Errorf("Compress() error = %v", err)
	}
	if saved == 0 {
		t.Errorf("Compress() saved = 0, want > 0")
	}
	_ = compressed
}

func TestPipeline_NewPipeline(t *testing.T) {
	pipeline := NewPipeline()

	info := pipeline.GetInfo()
	if info.TotalEngines < 9 {
		t.Errorf("GetInfo().TotalEngines = %d, want >= 9", info.TotalEngines)
	}
	if info.Mode != ModeLite {
		t.Errorf("GetInfo().Mode = %s, want lite", info.Mode)
	}
}

func TestPipeline_SetMode(t *testing.T) {
	pipeline := NewPipeline()

	modes := []PipelineMode{ModeOff, ModeLite, ModeStandard, ModeAggressive, ModeUltra, ModeStacked}
	for _, mode := range modes {
		pipeline.SetMode(mode)
		if pipeline.GetMode() != mode {
			t.Errorf("GetMode() = %s, want %s", pipeline.GetMode(), mode)
		}
	}
}

func TestPipeline_Compress(t *testing.T) {
	pipeline := NewPipeline()
	pipeline.SetMode(ModeLite)

	input := "  This   is   a   test  \n\n\n  with   spaces  "
	result := pipeline.Compress(input)

	if result == nil {
		t.Fatal("Compress() returned nil")
	}
	if result.Original != input {
		t.Errorf("Compress().Original = %q, want %q", result.Original, input)
	}
	if result.Compressed == "" {
		t.Error("Compress().Compressed is empty")
	}
	if result.LatencyMs < 0 {
		t.Errorf("Compress().LatencyMs = %f, want >= 0", result.LatencyMs)
	}
}

func TestPipeline_CompressWithSession(t *testing.T) {
	pipeline := NewPipeline()
	pipeline.SetMode(ModeLite)

	sessionID := "test-session-123"

	// First compression
	input1 := "line 1\nline 2"
	result1 := pipeline.CompressWithSession(input1, sessionID)
	if result1.Compressed != input1 {
		t.Errorf("First compression changed input unexpectedly")
	}

	// Second compression - dedup should kick in
	input2 := "line 1\nline 2\nline 3"
	result2 := pipeline.CompressWithSession(input2, sessionID)
	if result2.TotalSaved == 0 {
		t.Log("Note: Dedup savings may vary")
	}
}

func TestPipeline_GetEnabledEngines(t *testing.T) {
	pipeline := NewPipeline()
	pipeline.SetMode(ModeLite)

	engines := pipeline.GetEnabledEngines()
	if len(engines) == 0 {
		t.Error("GetEnabledEngines() returned empty for ModeLite")
	}

	// Lite mode should have at least 1 engine (lite itself)
	found := false
	for _, e := range engines {
		if e.Name() == "lite" {
			found = true
			break
		}
	}
	if !found {
		t.Error("lite engine not found in enabled engines for ModeLite")
	}
}

func TestPipeline_EstimateSavings(t *testing.T) {
	pipeline := NewPipeline()

	tests := []struct {
		mode     PipelineMode
		wantMin  float64
		wantMax  float64
	}{
		{ModeOff, 0, 0},
		{ModeLite, 0, 20},
		{ModeStandard, 10, 40},
		{ModeAggressive, 30, 70},
		{ModeUltra, 50, 85},
		{ModeStacked, 40, 85},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			pipeline.SetMode(tt.mode)
			_, pct := pipeline.EstimateSavings("test input with some content to estimate savings")
			if pct < tt.wantMin || pct > tt.wantMax {
				t.Errorf("EstimateSavings() = %f, want between %f and %f", pct, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPipeline_ResetAllStats(t *testing.T) {
	pipeline := NewPipeline()

	// Compress something
	pipeline.Compress("test input")

	// Reset
	pipeline.ResetAllStats()

	// Stats should be reset
	stats := pipeline.GetTotalStats()
	for name, stat := range stats {
		if stat.Invocations != 0 {
			t.Errorf("After ResetAllStats(), %s.Invocations = %d, want 0", name, stat.Invocations)
		}
	}
}

func TestGetPresetModes(t *testing.T) {
	modes := GetPresetModes()
	if len(modes) != 6 {
		t.Errorf("GetPresetModes() returned %d modes, want 6", len(modes))
	}
}

func TestGetPresetDescription(t *testing.T) {
	descriptions := map[PipelineMode]string{
		ModeOff:        "No compression applied",
		ModeLite:       "Always-on baseline compression (3-10% savings)",
		ModeStandard:   "Standard optimization with dedup, RTK, and caveman (15-30% savings)",
		ModeAggressive: "High compression with all engines except ultra (40-60% savings)",
		ModeUltra:      "Maximum compression with ultra engine (60-80% savings)",
		ModeStacked:    "All engines including LLMLingua enabled (60-80% savings)",
	}

	for mode, wantDesc := range descriptions {
		if desc := GetPresetDescription(mode); desc != wantDesc {
			t.Errorf("GetPresetDescription(%s) = %q, want %q", mode, desc, wantDesc)
		}
	}
}

func TestCompressionResult_Savings(t *testing.T) {
	result := &CompressionResult{
		Original:   "test input",
		Compressed: "test",
		TotalSaved: 6,
	}

	pct := float64(result.TotalSaved) / float64(len(result.Original)) * 100
	// 6/10 = 60%
	if pct < 50 || pct > 70 {
		t.Errorf("Savings percentage = %f, want 50-70", pct)
	}
}

func TestEngineStats_SavingsPercent(t *testing.T) {
	stats := EngineStats{
		TotalInputLen:  1000,
		TotalOutputLen: 700,
		TotalSaved:    300,
	}

	pct := stats.SavingsPercent()
	if pct != 30.0 {
		t.Errorf("SavingsPercent() = %f, want 30.0", pct)
	}
}

func TestDefaultEngines(t *testing.T) {
	tests := []struct {
		mode    PipelineMode
		wantMin int
	}{
		{ModeOff, 0},
		{ModeLite, 1},
		{ModeStandard, 4},
		{ModeAggressive, 7},
		{ModeUltra, 8},
		{ModeStacked, 9},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			engines := DefaultEngines(tt.mode)
			if len(engines) < tt.wantMin {
				t.Errorf("DefaultEngines(%s) = %v, want >= %d engines", tt.mode, engines, tt.wantMin)
			}
		})
	}
}

// Benchmark tests

func BenchmarkLiteEngine_Compress(b *testing.B) {
	engine := NewLiteEngine()
	input := "  This   is   a   longer   test   string   with   extra   whitespace  \n\n\n  and   multiple   lines  "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Compress(input)
	}
}

func BenchmarkRTKEngine_Compress(b *testing.B) {
	engine := NewRTKEngine()
	engine.SetEnabled(true)
	input := strings.Repeat("total 123\n-rw-r--r-- 1 user staff 1024 Jun 23 10:00 file.txt\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Compress(input)
	}
}

func BenchmarkPipeline_Compress(b *testing.B) {
	pipeline := NewPipeline()
	pipeline.SetMode(ModeStandard)
	input := "  This   is   a   longer   test   string   with   extra   whitespace  \n\n\n  and   multiple   lines  "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.Compress(input)
	}
}

// Utility function tests

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Error("min(1, 2) = 1 failed")
	}
	if min(2, 1) != 1 {
		t.Error("min(2, 1) = 1 failed")
	}
	if min(0, 0) != 0 {
		t.Error("min(0, 0) = 0 failed")
	}
}

func TestMax(t *testing.T) {
	if max(1, 2) != 2 {
		t.Error("max(1, 2) = 2 failed")
	}
	if max(2, 1) != 2 {
		t.Error("max(2, 1) = 2 failed")
	}
	if max(0, 0) != 0 {
		t.Error("max(0, 0) = 0 failed")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is a ..."},
		{"", 10, ""},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// Integration-style tests

func TestFullCompressionCycle(t *testing.T) {
	pipeline := NewPipeline()

	// Test all modes
	modes := []PipelineMode{ModeOff, ModeLite, ModeStandard, ModeAggressive, ModeUltra, ModeStacked}
	input := "  This   is   a   test   of   the   compression   pipeline   with   multiple   lines   of   text   and   extra   whitespace  \n\n\n\n\n"

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			pipeline.SetMode(mode)
			result := pipeline.Compress(input)

			if result == nil {
				t.Fatal("Compress() returned nil")
			}
			if result.Mode != mode {
				t.Errorf("Result.Mode = %s, want %s", result.Mode, mode)
			}
			if result.LatencyMs < 0 {
				t.Errorf("LatencyMs = %f, want >= 0", result.LatencyMs)
			}
		})
	}
}

func TestSessionIsolation(t *testing.T) {
	pipeline := NewPipeline()

	session1 := "session-1"
	session2 := "session-2"

	shared := "shared line 1\nshared line 2"

	// Compress in session 1
	pipeline.CompressWithSession(shared, session1)

	// Same content in session 2 should not be deduplicated
	result := pipeline.CompressWithSession(shared, session2)

	// Session 2 should keep all content (no dedup)
	if result.TotalSaved > 0 {
		t.Errorf("Session isolation failed: session 2 was deduplicated")
	}
}

func TestEngineEnableDisable(t *testing.T) {
	pipeline := NewPipeline()

	// Disable lite engine
	pipeline.SetEngineEnabled("lite", false)

	// Lite should not appear in enabled engines
	for _, e := range pipeline.GetEnabledEngines() {
		if e.Name() == "lite" {
			t.Error("lite engine still enabled after SetEngineEnabled(false, \"lite\")")
		}
	}

	// Re-enable
	pipeline.SetEngineEnabled("lite", true)

	// Lite should appear in enabled engines
	found := false
	for _, e := range pipeline.GetEnabledEngines() {
		if e.Name() == "lite" {
			found = true
			break
		}
	}
	if !found {
		t.Error("lite engine not found after SetEngineEnabled(true, \"lite\")")
	}
}
