package learning

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrorPattern represents a discovered error pattern
type ErrorPattern struct {
	ID          string    `json:"id"`
	Signature   string    `json:"signature"` // Hash of error signature
	Type        string    `json:"type"`      // "compilation", "runtime", "network", "auth", etc.
	Pattern     string    `json:"pattern"`   // Regex pattern
	Description string    `json:"description"`
	Frequency   int       `json:"frequency"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Suggestions []string  `json:"suggestions"` // Suggested fixes
	FixedCount  int       `json:"fixed_count"` // Times this pattern was fixed
}

// ErrorEvent represents a single error occurrence
type ErrorEvent struct {
	ID         string            `json:"id"`
	Command    string            `json:"command"`
	ErrorType  string            `json:"error_type"`
	ErrorMsg   string            `json:"error_msg"`
	ExitCode   int               `json:"exit_code"`
	StackTrace string            `json:"stack_trace"`
	Context    map[string]string `json:"context"`
	Timestamp  time.Time         `json:"timestamp"`
}

// TrendPoint represents a trend data point
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Type      string    `json:"type"`
}

// PatternExtractor extracts patterns from error events
type PatternExtractor struct {
	minFrequency int
	signatures   map[string]*ErrorPattern
}

// NewPatternExtractor creates a new pattern extractor
func NewPatternExtractor(minFrequency int) *PatternExtractor {
	return &PatternExtractor{
		minFrequency: minFrequency,
		signatures:   make(map[string]*ErrorPattern),
	}
}

// commonErrorPatterns contains known error patterns with suggestions
var commonErrorPatterns = []struct {
	typePattern  *regexp.Regexp
	msgPattern   *regexp.Regexp
	errorType    string
	description  string
	suggestions  []string
}{
	{
		typePattern: regexp.MustCompile(`(?i)(compilation|compile|syntax)`),
		msgPattern:  regexp.MustCompile(`(?i)(unexpected token|expected|undefined|cannot find)`),
		errorType:   "compilation",
		description: "Code has syntax or type errors",
		suggestions: []string{
			"Check for missing imports",
			"Verify type declarations",
			"Run linter to find syntax errors",
		},
	},
	{
		typePattern: regexp.MustCompile(`(?i)(null|nil|undefined|undefined)`),
		msgPattern:  regexp.MustCompile(`(?i)(cannot read|is not a|of undefined)`),
		errorType:   "runtime",
		description: "Runtime error - null/undefined reference",
		suggestions: []string{
			"Add null checks before access",
			"Initialize variables properly",
			"Check API responses for null values",
		},
	},
	{
		typePattern: regexp.MustCompile(`(?i)(connection|network|tcp|timeout)`),
		msgPattern:  regexp.MustCompile(`(?i)(refused|timeout|unreachable|reset)`),
		errorType:   "network",
		description: "Network connectivity issue",
		suggestions: []string{
			"Check network connectivity",
			"Verify service is running",
			"Check firewall rules",
			"Retry with exponential backoff",
		},
	},
	{
		typePattern: regexp.MustCompile(`(?i)(auth|permission|denied|unauthorized|forbidden)`),
		msgPattern:  regexp.MustCompile(`(?i)(invalid|expired|forbidden|access denied)`),
		errorType:   "auth",
		description: "Authentication or authorization error",
		suggestions: []string{
			"Check credentials are valid",
			"Verify token hasn't expired",
			"Ensure proper permissions are set",
		},
	},
	{
		typePattern: regexp.MustCompile(`(?i)(memory|oom|out of|allocation)`),
		msgPattern:  regexp.MustCompile(`(?i)(out of memory|cannot allocate)`),
		errorType:   "resource",
		description: "Resource exhaustion error",
		suggestions: []string{
			"Reduce batch sizes",
			"Add streaming/chunking",
			"Increase memory limits",
		},
	},
}

// Extract finds patterns in error events
func (e *PatternExtractor) Extract(events []ErrorEvent) ([]*ErrorPattern, error) {
	patterns := make(map[string]*ErrorPattern)

	for _, event := range events {
		signature := e.ExtractSignature(event.ErrorMsg, event.ExitCode)

		if _, exists := patterns[signature]; !exists {
			pattern := e.createPattern(event, signature)
			patterns[signature] = pattern
		}

		patterns[signature].Frequency++
		if event.Timestamp.After(patterns[signature].LastSeen) {
			patterns[signature].LastSeen = event.Timestamp
		}
	}

	var result []*ErrorPattern
	for _, pattern := range patterns {
		if pattern.Frequency >= e.minFrequency {
			result = append(result, pattern)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})

	return result, nil
}

func (e *PatternExtractor) createPattern(event ErrorEvent, signature string) *ErrorPattern {
	var matchedType string
	var description string
	var suggestions []string

	for _, cp := range commonErrorPatterns {
		if cp.typePattern.MatchString(event.ErrorType) || cp.msgPattern.MatchString(event.ErrorMsg) {
			matchedType = cp.errorType
			description = cp.description
			suggestions = cp.suggestions
			break
		}
	}

	if matchedType == "" {
		matchedType = "unknown"
		description = "Unknown error pattern"
		suggestions = []string{
			"Review error message for clues",
			"Check system logs for context",
			"Search error message online",
		}
	}

	return &ErrorPattern{
		ID:          uuid.New().String(),
		Signature:   signature,
		Type:        matchedType,
		Pattern:     e.extractPattern(event.ErrorMsg),
		Description: description,
		Frequency:   1,
		FirstSeen:   event.Timestamp,
		LastSeen:    event.Timestamp,
		Suggestions: suggestions,
		FixedCount:  0,
	}
}

// extractPattern creates a regex-like pattern from error message
func (e *PatternExtractor) extractPattern(msg string) string {
	// Replace specific values with placeholders
	pattern := regexp.MustCompile(`["'][\w\-\./:]+["']`)
	result := pattern.ReplaceAllString(msg, "?")

	// Replace numbers
	numberPattern := regexp.MustCompile(`\d+`)
	result = numberPattern.ReplaceAllString(result, "*")

	return result
}

// ExtractSignature creates a signature for an error
func (e *PatternExtractor) ExtractSignature(errMsg string, exitCode int) string {
	normalized := normalizeErrorMessage(errMsg)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", normalized, exitCode)))
	return hex.EncodeToString(hash[:8])
}

// normalizeErrorMessage removes variable parts of error messages
func normalizeErrorMessage(msg string) string {
	// Remove variable paths and line numbers
	msg = regexp.MustCompile(`[/\\][\w\-\.]+:\d+`).ReplaceAllString(msg, "FILE:LINE")
	msg = regexp.MustCompile(`line \d+`).ReplaceAllString(msg, "line *")
	msg = regexp.MustCompile(`at [\w\.]+ \(.*?\)`).ReplaceAllString(msg, "at *")
	msg = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(msg, "0x*")
	msg = regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).ReplaceAllString(msg, "*")
	return strings.ToLower(msg)
}

// LearningEngine learns from error patterns
type LearningEngine struct {
	patterns map[string]*ErrorPattern
	history  []ErrorEvent
	db       interface{}
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine(db interface{}) *LearningEngine {
	return &LearningEngine{
		patterns: make(map[string]*ErrorPattern),
		history:  make([]ErrorEvent, 0),
		db:       db,
	}
}

// RecordError records an error event and updates patterns
func (e *LearningEngine) RecordError(ctx context.Context, event *ErrorEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	extractor := NewPatternExtractor(1)
	signature := extractor.ExtractSignature(event.ErrorMsg, event.ExitCode)

	e.history = append(e.history, *event)

	if existing, exists := e.patterns[signature]; exists {
		existing.Frequency++
		if event.Timestamp.After(existing.LastSeen) {
			existing.LastSeen = event.Timestamp
		}
	} else {
		pattern := extractor.createPattern(*event, signature)
		e.patterns[signature] = pattern
	}

	if db, ok := e.db.(*sql.DB); ok {
		return e.persistToDB(ctx, db, event, signature)
	}

	return nil
}

func (e *LearningEngine) persistToDB(ctx context.Context, db *sql.DB, event *ErrorEvent, signature string) error {
	contextJSON, _ := json.Marshal(event.Context)

	_, err := db.ExecContext(ctx, `
		INSERT INTO error_events (id, command, error_type, error_msg, exit_code, stack_trace, context, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			error_msg = EXCLUDED.error_msg,
			timestamp = EXCLUDED.timestamp
	`, event.ID, event.Command, event.ErrorType, event.ErrorMsg, event.ExitCode, event.StackTrace, contextJSON, event.Timestamp)

	return err
}

// GetPatterns returns all known patterns
func (e *LearningEngine) GetPatterns(ctx context.Context) ([]*ErrorPattern, error) {
	result := make([]*ErrorPattern, 0, len(e.patterns))
	for _, p := range e.patterns {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})
	return result, nil
}

// GetPatternByError returns matching pattern for an error
func (e *LearningEngine) GetPatternByError(ctx context.Context, errMsg string, exitCode int) (*ErrorPattern, error) {
	extractor := NewPatternExtractor(1)
	signature := extractor.ExtractSignature(errMsg, exitCode)

	if pattern, exists := e.patterns[signature]; exists {
		return pattern, nil
	}

	normalized := normalizeErrorMessage(errMsg)
	for _, pattern := range e.patterns {
		if pattern.Type == extractErrorType(errMsg) {
			if similarity(normalized, normalizeErrorMessage(pattern.Pattern)) > 0.7 {
				return pattern, nil
			}
		}
	}

	return nil, nil
}

// extractErrorType determines the error type from message
func extractErrorType(msg string) string {
	msgLower := strings.ToLower(msg)
	if strings.Contains(msgLower, "syntax") || strings.Contains(msgLower, "compile") {
		return "compilation"
	}
	if strings.Contains(msgLower, "null") || strings.Contains(msgLower, "undefined") || strings.Contains(msgLower, "nil") {
		return "runtime"
	}
	if strings.Contains(msgLower, "connection") || strings.Contains(msgLower, "timeout") || strings.Contains(msgLower, "refused") {
		return "network"
	}
	if strings.Contains(msgLower, "auth") || strings.Contains(msgLower, "permission") || strings.Contains(msgLower, "denied") {
		return "auth"
	}
	return "unknown"
}

// similarity calculates simple string similarity
func similarity(a, b string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	aRunes := []rune(a)
	bRunes := []rune(b)

	var match int
	minLen := len(aRunes)
	if len(bRunes) < minLen {
		minLen = len(bRunes)
	}

	for i := 0; i < minLen; i++ {
		if aRunes[i] == bRunes[i] {
			match++
		}
	}

	return float64(match) / float64(max(len(aRunes), len(bRunes)))
}

// GetSuggestions returns fix suggestions for an error
func (e *LearningEngine) GetSuggestions(ctx context.Context, errMsg string, exitCode int) ([]string, error) {
	pattern, err := e.GetPatternByError(ctx, errMsg, exitCode)
	if err != nil || pattern == nil {
		return []string{
			"Review error message for clues",
			"Check documentation for this error",
			"Search for similar errors online",
		}, nil
	}

	suggestions := make([]string, len(pattern.Suggestions))
	copy(suggestions, pattern.Suggestions)

	if pattern.FixedCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("This error has been fixed %d time(s) before", pattern.FixedCount))
	}

	return suggestions, nil
}

// MarkFixed marks a pattern as fixed
func (e *LearningEngine) MarkFixed(ctx context.Context, patternID string) error {
	for _, pattern := range e.patterns {
		if pattern.ID == patternID {
			pattern.FixedCount++
			return nil
		}
	}
	return fmt.Errorf("pattern not found: %s", patternID)
}

// GetErrorHistory returns error history
func (e *LearningEngine) GetErrorHistory(ctx context.Context, limit int) ([]ErrorEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	historyLen := len(e.history)
	if historyLen == 0 {
		return []ErrorEvent{}, nil
	}

	start := 0
	if historyLen > limit {
		start = historyLen - limit
	}

	result := make([]ErrorEvent, limit)
	copy(result, e.history[start:min(start+limit, historyLen)])

	return result, nil
}

// GetMostCommonErrors returns most frequent errors
func (e *LearningEngine) GetMostCommonErrors(ctx context.Context, limit int) ([]*ErrorPattern, error) {
	if limit <= 0 {
		limit = 10
	}

	patterns, err := e.GetPatterns(ctx)
	if err != nil {
		return nil, err
	}

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// GetErrorTrend returns error trends over time
func (e *LearningEngine) GetErrorTrend(ctx context.Context, period string) ([]TrendPoint, error) {
	var interval time.Duration
	switch period {
	case "hour":
		interval = time.Hour
	case "day":
		interval = 24 * time.Hour
	case "week":
		interval = 7 * 24 * time.Hour
	default:
		interval = time.Hour
	}

	typePoints := make(map[string][]TrendPoint)

	for _, event := range e.history {
		bucket := event.Timestamp.Truncate(interval)

		found := false
		for i := range typePoints[event.ErrorType] {
			if typePoints[event.ErrorType][i].Timestamp.Equal(bucket) {
				typePoints[event.ErrorType][i].Count++
				found = true
				break
			}
		}

		if !found {
			typePoints[event.ErrorType] = append(typePoints[event.ErrorType], TrendPoint{
				Timestamp: bucket,
				Count:     1,
				Type:      event.ErrorType,
			})
		}
	}

	var result []TrendPoint
	for _, points := range typePoints {
		result = append(result, points...)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// Train trains the learning engine on historical data
func (e *LearningEngine) Train(ctx context.Context, events []ErrorEvent) error {
	extractor := NewPatternExtractor(1)

	for _, event := range events {
		signature := extractor.ExtractSignature(event.ErrorMsg, event.ExitCode)

		if _, exists := e.patterns[signature]; !exists {
			pattern := extractor.createPattern(event, signature)
			e.patterns[signature] = pattern
		}

		e.patterns[signature].Frequency++
		if event.Timestamp.After(e.patterns[signature].LastSeen) {
			e.patterns[signature].LastSeen = event.Timestamp
		}
	}

	return nil
}

// Predict predicts potential errors for a command
func (e *LearningEngine) Predict(ctx context.Context, command string) ([]string, error) {
	var predictions []string

	commandLower := strings.ToLower(command)

	for _, pattern := range e.patterns {
		if pattern.Frequency >= 3 && pattern.FixedCount == 0 {
			hints := getCommandHints(commandLower, pattern)
			if len(hints) > 0 {
				predictions = append(predictions, hints...)
			}
		}
	}

	return predictions, nil
}

// getCommandHints returns hints based on command and pattern
func getCommandHints(command string, pattern *ErrorPattern) []string {
	var hints []string

	switch pattern.Type {
	case "compilation":
		if strings.Contains(command, "build") || strings.Contains(command, "compile") {
			hints = append(hints, "Run linter before building to catch syntax errors")
		}
	case "network":
		if strings.Contains(command, "curl") || strings.Contains(command, "fetch") || strings.Contains(command, "http") {
			hints = append(hints, "Add retry logic with exponential backoff")
			hints = append(hints, "Check network connectivity first")
		}
	case "auth":
		if strings.Contains(command, "login") || strings.Contains(command, "auth") || strings.Contains(command, "token") {
			hints = append(hints, "Verify credentials are valid and not expired")
			hints = append(hints, "Check token refresh logic")
		}
	}

	return hints
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
