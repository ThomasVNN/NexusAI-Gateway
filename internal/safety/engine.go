package safety

import (
	"context"
	"regexp"
	"strings"
	"sync"
)

// SafetyLevel defines the strictness of safety checks
type SafetyLevel int

const (
	SafetyLevelOff SafetyLevel = iota
	SafetyLevelLow
	SafetyLevelMedium
	SafetyLevelHigh
	SafetyLevelStrict
)

// SafetyResult represents the result of a safety evaluation
type SafetyResult struct {
	Passed   bool     `json:"passed"`
	Score    float64  `json:"score"`
	Category string   `json:"category"`
	Reason   string   `json:"reason"`
	Flags    []string `json:"flags,omitempty"`
}

// EvaluationMetrics holds aggregated safety metrics
type EvaluationMetrics struct {
	TotalEvaluations  int64            `json:"total_evaluations"`
	PassedEvaluations int64            `json:"passed_evaluations"`
	FailedEvaluations int64            `json:"failed_evaluations"`
	ByCategory        map[string]int64 `json:"by_category"`
	AverageScore      float64          `json:"average_score"`
	mu                sync.RWMutex
}

// GlobalMetrics is the global safety metrics
var GlobalMetrics = &EvaluationMetrics{
	ByCategory: make(map[string]int64),
}

// HarmCategory represents different categories of potential harm
type HarmCategory string

const (
	HarmCategoryToxicity         HarmCategory = "toxicity"
	HarmCategoryHateSpeech       HarmCategory = "hate_speech"
	HarmCategoryViolence         HarmCategory = "violence"
	HarmCategorySexualContent    HarmCategory = "sexual_content"
	HarmCategorySelfHarm         HarmCategory = "self_harm"
	HarmCategoryDangerousContent HarmCategory = "dangerous_content"
	HarmCategoryPII              HarmCategory = "pii"
	HarmCategoryPromptInjection  HarmCategory = "prompt_injection"
	HarmCategoryDataExfiltration HarmCategory = "data_exfiltration"
)

// SafetyEngine performs AI safety evaluations
type SafetyEngine struct {
	level     SafetyLevel
	patterns  map[HarmCategory]*regexp.Regexp
	metrics   *EvaluationMetrics
	threshold float64
}

// NewSafetyEngine creates a new safety evaluation engine
func NewSafetyEngine(level SafetyLevel) *SafetyEngine {
	engine := &SafetyEngine{
		level:     level,
		patterns:  make(map[HarmCategory]*regexp.Regexp),
		metrics:   GlobalMetrics,
		threshold: 0.7,
	}

	// Initialize patterns for each harm category
	engine.initPatterns()

	return engine
}

// initPatterns initializes regex patterns for harm detection
func (e *SafetyEngine) initPatterns() {
	// Toxicity patterns
	e.patterns[HarmCategoryToxicity] = regexp.MustCompile(`(?i)(?:hate|disgust|angry|furious|rage|fuming)`)

	// Hate speech patterns
	e.patterns[HarmCategoryHateSpeech] = regexp.MustCompile(`(?i)(?:(?:kill|murder|attack|harm|violence|exterminate)(?:\s+(?:all|every|those|the))?\s+(?:jews|blacks|muslims|immigrants|women|gays)?)`)

	// Violence patterns
	e.patterns[HarmCategoryViolence] = regexp.MustCompile(`(?i)(?:(?:kill|murder|attack|harm|beat|torture|abuse|destroy)(?:\s+(?:someone|people|a\s+person|them))?)`)

	// Sexual content patterns
	e.patterns[HarmCategorySexualContent] = regexp.MustCompile(`(?i)(?:explicit sexual|nude|naked|erotic|porn|xxx)`)

	// Self-harm patterns
	e.patterns[HarmCategorySelfHarm] = regexp.MustCompile(`(?i)(?:suicide|self.?harm|cut.?myself|kill.?myself|end.?my.?life)`)

	// Dangerous content patterns
	e.patterns[HarmCategoryDangerousContent] = regexp.MustCompile(`(?i)(?:how\s+to|instructions?\s+for|guide\s+to)\s+(?:make\s+(?:bomb|weapon|explosive|poison|drug)|hack\s+(?:computer|account|system)|kill|attack)`)

	// PII patterns (enhanced from privacy filter)
	e.patterns[HarmCategoryPII] = regexp.MustCompile(`(?i)(?:ssn|social\s+security|credit\s+card|password|secret\s+key|api\s+key)\s*[:=]\s*\S+`)

	// Prompt injection patterns
	e.patterns[HarmCategoryPromptInjection] = regexp.MustCompile(`(?i)(?:ignore\s+(?:previous|all|above)|disregard\s+(?:instructions?|previous)|system\s*:\s*|you\s+are\s+now\s+|new\s+instructions?:)`)

	// Data exfiltration patterns
	e.patterns[HarmCategoryDataExfiltration] = regexp.MustCompile(`(?i)(?:dump\s+(?:all|entire|complete)\s+(?:database|table|data)|extract\s+all\s+(?:records|users|passwords|secrets))`)
}

// Evaluate checks a prompt or response for safety violations
func (e *SafetyEngine) Evaluate(ctx context.Context, content string, contentType string) *SafetyResult {
	if e.level == SafetyLevelOff {
		return &SafetyResult{Passed: true, Score: 1.0, Category: "none"}
	}

	result := &SafetyResult{
		Passed: true,
		Score:  1.0,
		Flags:  []string{},
	}

	contentLower := strings.ToLower(content)

	// Check each harm category
	for category, pattern := range e.patterns {
		if matches := pattern.FindString(contentLower); matches != "" {
			severity := e.calculateSeverity(category, matches)
			result.Passed = false
			result.Category = string(category)
			result.Score -= severity
			result.Flags = append(result.Flags, string(category))

			if len(result.Reason) > 0 {
				result.Reason += "; "
			}
			result.Reason += "Detected " + string(category) + " pattern"

			// Update metrics
			e.metrics.mu.Lock()
			e.metrics.ByCategory[string(category)]++
			e.metrics.mu.Unlock()

			// In strict mode, fail immediately on certain categories
			if e.level == SafetyLevelStrict {
				if category == HarmCategoryDangerousContent || category == HarmCategoryPromptInjection {
					break
				}
			}
		}
	}

	// Check content length anomalies (potential injection)
	if len(content) > 100000 {
		result.Passed = false
		result.Score -= 0.5
		result.Reason += "; Content length exceeds safe limit"
	}

	// Update global metrics
	e.metrics.mu.Lock()
	e.metrics.TotalEvaluations++
	if result.Passed {
		e.metrics.PassedEvaluations++
	} else {
		e.metrics.FailedEvaluations++
	}
	e.metrics.mu.Unlock()

	return result
}

// calculateSeverity determines the severity of a match
func (e *SafetyEngine) calculateSeverity(category HarmCategory, match string) float64 {
	// High severity categories
	highSeverity := []HarmCategory{
		HarmCategoryDangerousContent,
		HarmCategoryPromptInjection,
		HarmCategorySelfHarm,
		HarmCategoryDataExfiltration,
	}

	for _, cat := range highSeverity {
		if cat == category {
			return 0.8
		}
	}

	// Medium severity categories
	mediumSeverity := []HarmCategory{
		HarmCategoryViolence,
		HarmCategoryHateSpeech,
		HarmCategorySexualContent,
	}

	for _, cat := range mediumSeverity {
		if cat == category {
			return 0.5
		}
	}

	// Low severity categories
	return 0.3
}

// EvaluatePrompt evaluates a user prompt for safety
func (e *SafetyEngine) EvaluatePrompt(ctx context.Context, prompt string) *SafetyResult {
	return e.Evaluate(ctx, prompt, "prompt")
}

// EvaluateResponse evaluates an AI response for safety
func (e *SafetyEngine) EvaluateResponse(ctx context.Context, response string) *SafetyResult {
	return e.Evaluate(ctx, response, "response")
}

// GetMetrics returns current safety metrics
func (e *SafetyEngine) GetMetrics() map[string]interface{} {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	total := e.metrics.TotalEvaluations
	var avgScore float64
	if total > 0 {
		avgScore = float64(e.metrics.PassedEvaluations) / float64(total)
	}

	return map[string]interface{}{
		"total_evaluations": e.metrics.TotalEvaluations,
		"passed":            e.metrics.PassedEvaluations,
		"failed":            e.metrics.FailedEvaluations,
		"pass_rate":         avgScore,
		"by_category":       e.metrics.ByCategory,
	}
}

// SafetyPolicy defines acceptable content policies
type SafetyPolicy struct {
	AllowedCategories []HarmCategory
	BlockedPatterns   []string
	MaxContentLength  int
}

// DefaultPolicy returns the default safety policy
func DefaultPolicy() *SafetyPolicy {
	return &SafetyPolicy{
		AllowedCategories: []HarmCategory{},
		BlockedPatterns:   []string{},
		MaxContentLength:  100000,
	}
}

// StrictPolicy returns a strict safety policy
func StrictPolicy() *SafetyPolicy {
	return &SafetyPolicy{
		AllowedCategories: []HarmCategory{},
		BlockedPatterns: []string{
			`(?i)violence`,
			`(?i)kill`,
			`(?i)weapon`,
			`(?i)drug`,
		},
		MaxContentLength: 50000,
	}
}
