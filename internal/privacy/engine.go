package privacy

import (
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

// RedactionLevel defines how aggressively PII should be redacted
type RedactionLevel int

const (
	RedactionLevelNone RedactionLevel = iota
	RedactionLevelBasic
	RedactionLevelStandard
	RedactionLevelStrict
)

// RedactionRule defines a single redaction rule
type RedactionRule struct {
	Type     PIIType
	Marker   string
	Enabled  bool
	CountCap int // Maximum number of redactions (0 = unlimited)
}

// FilterConfig holds the configuration for the redaction engine
type FilterConfig struct {
	DefaultMarker  string
	Level          RedactionLevel
	Rules          map[PIIType]RedactionRule
	MaxRedactions  int
	PreserveFormat bool
	CaseSensitive  bool
}

// DefaultFilterConfig returns the standard filter configuration
func DefaultFilterConfig() *FilterConfig {
	return &FilterConfig{
		DefaultMarker: "[REDACTED]",
		Level:         RedactionLevelStandard,
		Rules: map[PIIType]RedactionRule{
			PIITypeEmail: {
				Type:     PIITypeEmail,
				Marker:   "[REDACTED_EMAIL]",
				Enabled:  true,
				CountCap: 50,
			},
			PIITypePhone: {
				Type:     PIITypePhone,
				Marker:   "[REDACTED_PHONE]",
				Enabled:  true,
				CountCap: 50,
			},
			PIITypeSSN: {
				Type:     PIITypeSSN,
				Marker:   "[REDACTED_SSN]",
				Enabled:  true,
				CountCap: 10,
			},
			PIITypeCreditCard: {
				Type:     PIITypeCreditCard,
				Marker:   "[REDACTED_CARD]",
				Enabled:  true,
				CountCap: 10,
			},
			PIITypeIPAddress: {
				Type:     PIITypeIPAddress,
				Marker:   "[REDACTED_IP]",
				Enabled:  true,
				CountCap: 50,
			},
		},
		MaxRedactions:  200,
		PreserveFormat: false,
		CaseSensitive:  false,
	}
}

// Engine handles PII detection and redaction with configurable rules
type Engine struct {
	detector *Detector
	config   *FilterConfig
	mu       sync.RWMutex
}

// NewEngine creates a new redaction engine with default configuration
func NewEngine() *Engine {
	return &Engine{
		detector: NewDetector(),
		config:   DefaultFilterConfig(),
	}
}

// NewEngineWithConfig creates a new redaction engine with custom configuration
func NewEngineWithConfig(config *FilterConfig) *Engine {
	return &Engine{
		detector: NewDetector(),
		config:   config,
	}
}

// Redact sanitizes text by replacing all detected PII with appropriate markers
func (e *Engine) Redact(text string) string {
	if e.config.Level == RedactionLevelNone {
		return text
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	result := text
	totalRedactions := 0

	for piiType, rule := range e.config.Rules {
		if !rule.Enabled {
			continue
		}

		pattern := e.detector.patterns[piiType]
		if pattern == nil {
			continue
		}

		marker := rule.Marker
		if marker == "" {
			marker = e.config.DefaultMarker
		}

		if e.config.CaseSensitive {
			result = pattern.Pattern.ReplaceAllStringFunc(result, func(match string) string {
				if rule.CountCap > 0 && totalRedactions >= e.config.MaxRedactions {
					return match
				}
				totalRedactions++
				return marker
			})
		} else {
			// Case-insensitive replacement
			result = pattern.Pattern.ReplaceAllStringFunc(result, func(match string) string {
				if rule.CountCap > 0 && totalRedactions >= e.config.MaxRedactions {
					return match
				}
				totalRedactions++
				return marker
			})
		}
	}

	return result
}

// RedactWithLevel applies redaction at a specific level
func (e *Engine) RedactWithLevel(text string, level RedactionLevel) string {
	if level == RedactionLevelNone {
		return text
	}

	// Temporarily adjust level
	e.mu.Lock()
	originalLevel := e.config.Level
	e.config.Level = level
	e.mu.Unlock()

	result := e.Redact(text)

	// Restore original level
	e.mu.Lock()
	e.config.Level = originalLevel
	e.mu.Unlock()

	return result
}

// RedactPartial redacts only specific PII types
func (e *Engine) RedactPartial(text string, types []PIIType) string {
	if len(types) == 0 {
		return text
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	result := text

	for _, piiType := range types {
		rule, ok := e.config.Rules[piiType]
		if !ok || !rule.Enabled {
			continue
		}

		pattern := e.detector.patterns[piiType]
		if pattern == nil {
			continue
		}

		marker := rule.Marker
		if marker == "" {
			marker = e.config.DefaultMarker
		}

		result = pattern.Pattern.ReplaceAllString(result, marker)
	}

	return result
}

// Detect returns all detected PII in the text
func (e *Engine) Detect(text string) []DetectResult {
	return e.detector.Detect(text)
}

// DetectPartial returns only specific PII types
func (e *Engine) DetectPartial(text string, types []PIIType) []DetectResult {
	return e.detector.DetectWithFilter(text, types)
}

// GetRedactionCount returns the number of redactions that would be performed
func (e *Engine) GetRedactionCount(text string) map[PIIType]int {
	counts := make(map[PIIType]int)
	results := e.Detect(text)

	for _, result := range results {
		counts[result.Type]++
	}

	return counts
}

// GetTotalRedactionCount returns the total number of PII instances detected
func (e *Engine) GetTotalRedactionCount(text string) int {
	return len(e.Detect(text))
}

// UpdateConfig updates the engine configuration
func (e *Engine) UpdateConfig(config *FilterConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config
}

// GetConfig returns a copy of the current configuration
func (e *Engine) GetConfig() *FilterConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent concurrent access issues
	configCopy := *e.config
	rulesCopy := make(map[PIIType]RedactionRule)
	for k, v := range e.config.Rules {
		rulesCopy[k] = v
	}
	configCopy.Rules = rulesCopy

	return &configCopy
}

// EnableType enables redaction for a specific PII type
func (e *Engine) EnableType(piiType PIIType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule, ok := e.config.Rules[piiType]; ok {
		rule.Enabled = true
		e.config.Rules[piiType] = rule
	}
}

// DisableType disables redaction for a specific PII type
func (e *Engine) DisableType(piiType PIIType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule, ok := e.config.Rules[piiType]; ok {
		rule.Enabled = false
		e.config.Rules[piiType] = rule
	}
}

// SetMarker sets a custom marker for a specific PII type
func (e *Engine) SetMarker(piiType PIIType, marker string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule, ok := e.config.Rules[piiType]; ok {
		rule.Marker = marker
		e.config.Rules[piiType] = rule
	}
}

// GetMarker returns the current marker for a specific PII type
func (e *Engine) GetMarker(piiType PIIType) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if rule, ok := e.config.Rules[piiType]; ok {
		return rule.Marker
	}
	return e.config.DefaultMarker
}

// SetLevel sets the redaction level
func (e *Engine) SetLevel(level RedactionLevel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Level = level
}

// SetMaxRedactions sets the maximum number of total redactions
func (e *Engine) SetMaxRedactions(max int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.MaxRedactions = max
}

// RedactionResult holds the result of a redaction operation
type RedactionResult struct {
	Original      string
	Redacted      string
	RedactionType []PIIType
	Counts        map[PIIType]int
	TotalCount    int
	Level         RedactionLevel
}

// RedactWithResult performs redaction and returns detailed results
func (e *Engine) RedactWithResult(text string) *RedactionResult {
	detections := e.Detect(text)

	counts := make(map[PIIType]int)
	var types []PIIType
	typeSet := make(map[PIIType]bool)

	for _, d := range detections {
		counts[d.Type]++
		if !typeSet[d.Type] {
			types = append(types, d.Type)
			typeSet[d.Type] = true
		}
	}

	return &RedactionResult{
		Original:      text,
		Redacted:      e.Redact(text),
		RedactionType: types,
		Counts:        counts,
		TotalCount:    len(detections),
		Level:         e.config.Level,
	}
}

// ReplaceMarker replaces a specific marker in redacted text with another
func (e *Engine) ReplaceMarker(text string, oldMarker string, newMarker string) string {
	return strings.ReplaceAll(text, oldMarker, newMarker)
}

// RestoreMarkers attempts to restore redacted content with original values
// This is only possible if the original values were stored during redaction
func (e *Engine) RestoreMarkers(text string, originals map[string]string) string {
	result := text
	for original, marker := range originals {
		result = strings.ReplaceAll(result, marker, original)
	}
	return result
}

// ScanAndLog scans text for PII and logs findings
func (e *Engine) ScanAndLog(text string, logger *slog.Logger) {
	results := e.Detect(text)
	if len(results) == 0 {
		logger.Debug("No PII detected in text")
		return
	}

	counts := make(map[PIIType]int)
	for _, r := range results {
		counts[r.Type]++
	}

	for piiType, count := range counts {
		logger.Info("PII detected",
			slog.String("type", string(piiType)),
			slog.Int("count", count),
		)
	}
}

// ValidatePattern checks if a regex pattern is valid
func ValidatePattern(pattern string) bool {
	_, err := regexp.Compile(pattern)
	return err == nil
}
