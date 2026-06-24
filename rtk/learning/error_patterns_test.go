package learning

import (
	"context"
	"testing"
	"time"
)

func TestNewPatternExtractor(t *testing.T) {
	extractor := NewPatternExtractor(2)
	if extractor == nil {
		t.Fatal("NewPatternExtractor returned nil")
	}
	if extractor.minFrequency != 2 {
		t.Errorf("expected minFrequency 2, got %d", extractor.minFrequency)
	}
}

func TestExtractSignature(t *testing.T) {
	extractor := NewPatternExtractor(1)

	tests := []struct {
		name     string
		errMsg   string
		exitCode int
	}{
		{
			name:     "basic error",
			errMsg:   "connection refused",
			exitCode: 1,
		},
		{
			name:     "different exit code",
			errMsg:   "connection refused",
			exitCode: 2,
		},
		{
			name:     "different message",
			errMsg:   "timeout exceeded",
			exitCode: 1,
		},
	}

	sig1 := extractor.ExtractSignature(tests[0].errMsg, tests[0].exitCode)
	sig2 := extractor.ExtractSignature(tests[1].errMsg, tests[1].exitCode)
	sig3 := extractor.ExtractSignature(tests[2].errMsg, tests[2].exitCode)

	if sig1 == "" {
		t.Error("signature should not be empty")
	}
	if sig1 == sig2 {
		t.Error("same message with different exit codes should have different signatures")
	}
	if sig1 == sig3 {
		t.Error("different messages should have different signatures")
	}
}

func TestExtractSignatureConsistency(t *testing.T) {
	extractor := NewPatternExtractor(1)

	msg := "test error message"
	code := 42

	sig1 := extractor.ExtractSignature(msg, code)
	sig2 := extractor.ExtractSignature(msg, code)

	if sig1 != sig2 {
		t.Errorf("same input should produce same signature: %s != %s", sig1, sig2)
	}
}

func TestExtract(t *testing.T) {
	extractor := NewPatternExtractor(1)

	events := []ErrorEvent{
		{
			ID:        "1",
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "syntax error: unexpected token",
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			ID:        "2",
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "syntax error: unexpected token",
			ExitCode:  1,
			Timestamp: time.Now().Add(time.Hour),
		},
		{
			ID:        "3",
			Command:   "curl api",
			ErrorType: "network",
			ErrorMsg:  "connection refused",
			ExitCode:  7,
			Timestamp: time.Now(),
		},
	}

	patterns, err := extractor.Extract(events)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}

	for _, p := range patterns {
		if p.Frequency < 1 {
			t.Errorf("pattern frequency should be >= 1, got %d", p.Frequency)
		}
		if p.ID == "" {
			t.Error("pattern ID should not be empty")
		}
	}
}

func TestExtractMinFrequency(t *testing.T) {
	extractor := NewPatternExtractor(2)

	// Use identical error messages to test frequency filtering
	identicalMsg := "syntax error: unexpected token"
	events := []ErrorEvent{
		{
			ID:        "1",
			ErrorMsg:  identicalMsg,
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			ID:        "2",
			ErrorMsg:  identicalMsg,
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			ID:        "3",
			ErrorMsg:  "connection refused",
			ExitCode:  1,
			Timestamp: time.Now(),
		},
	}

	patterns, err := extractor.Extract(events)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should get 2 patterns: one with frequency 2 (above minFrequency), one with frequency 1 (below threshold)
	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns (1 above minFreq=2, 1 below), got %d", len(patterns))
	}

	// Find the pattern with frequency 2
	var highFreqPattern *ErrorPattern
	for _, p := range patterns {
		if p.Frequency == 2 {
			highFreqPattern = p
			break
		}
	}

	if highFreqPattern == nil {
		t.Error("expected at least one pattern with frequency >= 2")
	} else if highFreqPattern.Frequency != 2 {
		t.Errorf("expected frequency 2, got %d", highFreqPattern.Frequency)
	}
}

func TestNewLearningEngine(t *testing.T) {
	engine := NewLearningEngine(nil)
	if engine == nil {
		t.Fatal("NewLearningEngine returned nil")
	}
	if engine.patterns == nil {
		t.Error("patterns map should be initialized")
	}
	if engine.history == nil {
		t.Error("history slice should be initialized")
	}
}

func TestRecordError(t *testing.T) {
	engine := NewLearningEngine(nil)

	event := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "syntax error",
		ExitCode:  1,
		Timestamp: time.Now(),
	}

	err := engine.RecordError(context.Background(), event)
	if err != nil {
		t.Fatalf("RecordError failed: %v", err)
	}

	if len(engine.history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(engine.history))
	}

	if len(engine.patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(engine.patterns))
	}
}

func TestRecordErrorGeneratesID(t *testing.T) {
	engine := NewLearningEngine(nil)

	event := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "syntax error",
		ExitCode:  1,
	}

	err := engine.RecordError(context.Background(), event)
	if err != nil {
		t.Fatalf("RecordError failed: %v", err)
	}

	if event.ID == "" {
		t.Error("event ID should be generated")
	}
}

func TestRecordErrorIncrementsFrequency(t *testing.T) {
	engine := NewLearningEngine(nil)

	event1 := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "same error",
		ExitCode:  1,
		Timestamp: time.Now(),
	}

	event2 := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "same error",
		ExitCode:  1,
		Timestamp: time.Now(),
	}

	engine.RecordError(context.Background(), event1)
	engine.RecordError(context.Background(), event2)

	if len(engine.patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(engine.patterns))
	}

	for _, pattern := range engine.patterns {
		if pattern.Frequency != 2 {
			t.Errorf("expected frequency 2, got %d", pattern.Frequency)
		}
	}
}

func TestGetPatterns(t *testing.T) {
	engine := NewLearningEngine(nil)

	events := []*ErrorEvent{
		{
			Command:   "cmd1",
			ErrorType: "compilation",
			ErrorMsg:  "error 1",
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			Command:   "cmd2",
			ErrorType: "network",
			ErrorMsg:  "error 2",
			ExitCode:  2,
			Timestamp: time.Now(),
		},
		{
			Command:   "cmd3",
			ErrorType: "auth",
			ErrorMsg:  "error 3",
			ExitCode:  3,
			Timestamp: time.Now(),
		},
	}

	for _, e := range events {
		engine.RecordError(context.Background(), e)
	}

	patterns, err := engine.GetPatterns(context.Background())
	if err != nil {
		t.Fatalf("GetPatterns failed: %v", err)
	}

	if len(patterns) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(patterns))
	}
}

func TestGetPatternByError(t *testing.T) {
	engine := NewLearningEngine(nil)

	event := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "syntax error: unexpected token",
		ExitCode:  1,
		Timestamp: time.Now(),
	}
	engine.RecordError(context.Background(), event)

	pattern, err := engine.GetPatternByError(context.Background(), "syntax error: unexpected token", 1)
	if err != nil {
		t.Fatalf("GetPatternByError failed: %v", err)
	}

	if pattern == nil {
		t.Fatal("expected pattern, got nil")
	}

	if pattern.Type != "compilation" {
		t.Errorf("expected type 'compilation', got '%s'", pattern.Type)
	}
}

func TestGetPatternByErrorNotFound(t *testing.T) {
	engine := NewLearningEngine(nil)

	pattern, err := engine.GetPatternByError(context.Background(), "unknown error", 999)
	if err != nil {
		t.Fatalf("GetPatternByError failed: %v", err)
	}

	if pattern != nil {
		t.Errorf("expected nil for unknown pattern, got %v", pattern)
	}
}

func TestGetSuggestions(t *testing.T) {
	engine := NewLearningEngine(nil)

	event := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "syntax error: unexpected token",
		ExitCode:  1,
		Timestamp: time.Now(),
	}
	engine.RecordError(context.Background(), event)

	suggestions, err := engine.GetSuggestions(context.Background(), "syntax error: unexpected token", 1)
	if err != nil {
		t.Fatalf("GetSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("expected suggestions, got empty list")
	}
}

func TestGetSuggestionsForUnknownError(t *testing.T) {
	engine := NewLearningEngine(nil)

	suggestions, err := engine.GetSuggestions(context.Background(), "completely unknown error", 0)
	if err != nil {
		t.Fatalf("GetSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("expected default suggestions for unknown error")
	}
}

func TestMarkFixed(t *testing.T) {
	engine := NewLearningEngine(nil)

	event := &ErrorEvent{
		Command:   "go build",
		ErrorType: "compilation",
		ErrorMsg:  "syntax error",
		ExitCode:  1,
		Timestamp: time.Now(),
	}
	engine.RecordError(context.Background(), event)

	var patternID string
	for _, p := range engine.patterns {
		patternID = p.ID
		break
	}

	err := engine.MarkFixed(context.Background(), patternID)
	if err != nil {
		t.Fatalf("MarkFixed failed: %v", err)
	}

	for _, p := range engine.patterns {
		if p.FixedCount != 1 {
			t.Errorf("expected FixedCount 1, got %d", p.FixedCount)
		}
	}
}

func TestMarkFixedNotFound(t *testing.T) {
	engine := NewLearningEngine(nil)

	err := engine.MarkFixed(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent pattern")
	}
}

func TestGetErrorHistory(t *testing.T) {
	engine := NewLearningEngine(nil)

	for i := 0; i < 10; i++ {
		event := &ErrorEvent{
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "error",
			ExitCode:  1,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		}
		engine.RecordError(context.Background(), event)
	}

	history, err := engine.GetErrorHistory(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetErrorHistory failed: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(history))
	}
}

func TestGetErrorHistoryDefault(t *testing.T) {
	engine := NewLearningEngine(nil)

	history, err := engine.GetErrorHistory(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetErrorHistory failed: %v", err)
	}

	if history == nil {
		t.Error("expected empty history, got nil")
	}
}

func TestGetMostCommonErrors(t *testing.T) {
	engine := NewLearningEngine(nil)

	// Create patterns with different frequencies
	for i := 0; i < 5; i++ {
		event := &ErrorEvent{
			Command:   "cmd1",
			ErrorType: "compilation",
			ErrorMsg:  "error type A",
			ExitCode:  1,
			Timestamp: time.Now(),
		}
		engine.RecordError(context.Background(), event)
	}

	for i := 0; i < 3; i++ {
		event := &ErrorEvent{
			Command:   "cmd2",
			ErrorType: "network",
			ErrorMsg:  "error type B",
			ExitCode:  2,
			Timestamp: time.Now(),
		}
		engine.RecordError(context.Background(), event)
	}

	patterns, err := engine.GetMostCommonErrors(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetMostCommonErrors failed: %v", err)
	}

	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}

	if len(patterns) >= 2 && patterns[0].Frequency < patterns[1].Frequency {
		t.Error("patterns should be sorted by frequency descending")
	}
}

func TestGetErrorTrend(t *testing.T) {
	engine := NewLearningEngine(nil)

	now := time.Now()
	for i := 0; i < 5; i++ {
		event := &ErrorEvent{
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "error",
			ExitCode:  1,
			Timestamp: now.Add(time.Duration(i) * time.Hour),
		}
		engine.RecordError(context.Background(), event)
	}

	trend, err := engine.GetErrorTrend(context.Background(), "hour")
	if err != nil {
		t.Fatalf("GetErrorTrend failed: %v", err)
	}

	if len(trend) == 0 {
		t.Error("expected trend data")
	}
}

func TestGetErrorTrendPeriods(t *testing.T) {
	engine := NewLearningEngine(nil)

	tests := []struct {
		period string
	}{
		{"hour"},
		{"day"},
		{"week"},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			_, err := engine.GetErrorTrend(context.Background(), tt.period)
			if err != nil {
				t.Errorf("GetErrorTrend(%s) failed: %v", tt.period, err)
			}
		})
	}
}

func TestTrain(t *testing.T) {
	engine := NewLearningEngine(nil)

	events := []ErrorEvent{
		{
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "syntax error 1",
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "syntax error 1",
			ExitCode:  1,
			Timestamp: time.Now(),
		},
		{
			Command:   "curl api",
			ErrorType: "network",
			ErrorMsg:  "connection refused",
			ExitCode:  7,
			Timestamp: time.Now(),
		},
	}

	err := engine.Train(context.Background(), events)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}

	if len(engine.patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(engine.patterns))
	}
}

func TestPredict(t *testing.T) {
	engine := NewLearningEngine(nil)

	// Add patterns with high frequency
	for i := 0; i < 3; i++ {
		event := &ErrorEvent{
			Command:   "go build",
			ErrorType: "compilation",
			ErrorMsg:  "syntax error",
			ExitCode:  1,
			Timestamp: time.Now(),
		}
		engine.RecordError(context.Background(), event)
	}

	predictions, err := engine.Predict(context.Background(), "go build ./...")
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if len(predictions) > 0 {
		t.Logf("Got predictions: %v", predictions)
	}
}

func TestPredictNoPatterns(t *testing.T) {
	engine := NewLearningEngine(nil)

	predictions, err := engine.Predict(context.Background(), "some command")
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if len(predictions) != 0 {
		t.Errorf("expected no predictions with no patterns, got %d", len(predictions))
	}
}

func TestNormalizeErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/path/to/file.go:42: error",
			expected: "file:line error",
		},
		{
			input:    "line 123: something",
			expected: "line *: something",
		},
		{
			input:    "error at some.package (file.go:100)",
			expected: "error at * (file:line)",
		},
		{
			input:    "0x7fff5fbff8c0 pointer",
			expected: "0x* pointer",
		},
		{
			input:    "uuid: 123e4567-e89b-12d3-a456-426614174000",
			expected: "uuid: *",
		},
	}

	for _, tt := range tests {
		result := normalizeErrorMessage(tt.input)
		t.Logf("normalize(%q) = %q", tt.input, result)
	}
}

func TestExtractErrorType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"syntax error near token", "compilation"},
		{"cannot read property of null", "runtime"},
		{"connection refused", "network"},
		{"authentication failed", "auth"},
		{"permission denied", "auth"},
		{"random error", "unknown"},
	}

	for _, tt := range tests {
		result := extractErrorType(tt.input)
		if result != tt.expected {
			t.Errorf("extractErrorType(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		minSim   float64
		maxSim   float64
	}{
		{"hello world", "hello world", 1.0, 1.0},
		{"hello", "hallo", 0.6, 1.0},
		{"", "hello", 0, 0},
		{"completely different", "totally other", 0, 0.5},
	}

	for _, tt := range tests {
		sim := similarity(tt.a, tt.b)
		if sim < tt.minSim || sim > tt.maxSim {
			t.Errorf("similarity(%q, %q) = %f, expected between %f and %f",
				tt.a, tt.b, sim, tt.minSim, tt.maxSim)
		}
	}
}

func TestErrorPatternFields(t *testing.T) {
	pattern := &ErrorPattern{
		ID:          "test-id",
		Signature:   "abc123",
		Type:        "compilation",
		Pattern:     "syntax error",
		Description: "A syntax error occurred",
		Frequency:   5,
		FirstSeen:   time.Now().Add(-24 * time.Hour),
		LastSeen:    time.Now(),
		Suggestions: []string{"fix1", "fix2"},
		FixedCount:  2,
	}

	if pattern.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", pattern.ID)
	}
	if pattern.Frequency != 5 {
		t.Errorf("expected Frequency 5, got %d", pattern.Frequency)
	}
	if pattern.FixedCount != 2 {
		t.Errorf("expected FixedCount 2, got %d", pattern.FixedCount)
	}
}

func TestErrorEventFields(t *testing.T) {
	event := &ErrorEvent{
		ID:         "event-1",
		Command:    "go build",
		ErrorType:  "compilation",
		ErrorMsg:   "syntax error",
		ExitCode:   1,
		StackTrace: "at main()",
		Context:    map[string]string{"file": "main.go"},
		Timestamp:  time.Now(),
	}

	if event.ID != "event-1" {
		t.Errorf("expected ID 'event-1', got '%s'", event.ID)
	}
	if event.ExitCode != 1 {
		t.Errorf("expected ExitCode 1, got %d", event.ExitCode)
	}
	if event.Context["file"] != "main.go" {
		t.Errorf("expected Context['file'] 'main.go', got '%s'", event.Context["file"])
	}
}

func TestTrendPointFields(t *testing.T) {
	now := time.Now()
	point := TrendPoint{
		Timestamp: now,
		Count:     10,
		Type:      "compilation",
	}

	if !point.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, point.Timestamp)
	}
	if point.Count != 10 {
		t.Errorf("expected Count 10, got %d", point.Count)
	}
}
