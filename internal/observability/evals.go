package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AssertionType represents the type of assertion
type AssertionType int

const (
	AssertOutput AssertionType = iota
	AssertCost
	AssertLatency
	AssertQuality
	AssertFormat
	AssertContent
	AssertSchema
)

// String returns the string representation of an assertion type
func (a AssertionType) String() string {
	switch a {
	case AssertOutput:
		return "output"
	case AssertCost:
		return "cost"
	case AssertLatency:
		return "latency"
	case AssertQuality:
		return "quality"
	case AssertFormat:
		return "format"
	case AssertContent:
		return "content"
	case AssertSchema:
		return "schema"
	default:
		return "unknown"
	}
}

// Operator constants for assertions
const (
	OpEq        = "eq"
	OpNe        = "ne"
	OpGt        = "gt"
	OpGte       = "gte"
	OpLt        = "lt"
	OpLte       = "lte"
	OpContains  = "contains"
	OpNotContains = "not_contains"
	OpMatches   = "matches"
	OpIn        = "in"
	OpNotIn     = "not_in"
	OpStartsWith = "starts_with"
	OpEndsWith  = "ends_with"
)

// Assertion represents a single assertion in an eval suite
type Assertion struct {
	ID        string
	Type      AssertionType
	Target    string  // JSONPath to target field
	Expected  interface{}
	Operator  string
	Weight    float64 // Weight for scoring (0-1)
	Message   string  // Custom failure message
}

// NewAssertion creates a new assertion with defaults
func NewAssertion(assertionType AssertionType, target, operator string, expected interface{}) *Assertion {
	return &Assertion{
		ID:       uuid.New().String(),
		Type:     assertionType,
		Target:   target,
		Operator: operator,
		Expected: expected,
		Weight:   1.0,
	}
}

// SetWeight sets the weight of an assertion
func (a *Assertion) SetWeight(weight float64) *Assertion {
	a.Weight = weight
	return a
}

// SetMessage sets the custom failure message
func (a *Assertion) SetMessage(msg string) *Assertion {
	a.Message = msg
	return a
}

// EvalSuite represents a collection of assertions for evaluation
type EvalSuite struct {
	ID          string
	Name        string
	Description string
	Assertions  []*Assertion
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]interface{}
}

// NewEvalSuite creates a new eval suite
func NewEvalSuite(name, description string) *EvalSuite {
	return &EvalSuite{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Assertions:  make([]*Assertion, 0),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Metadata:    make(map[string]interface{}),
	}
}

// AddAssertion adds an assertion to the suite
func (s *EvalSuite) AddAssertion(assertion *Assertion) {
	s.Assertions = append(s.Assertions, assertion)
	s.UpdatedAt = time.Now().UTC()
}

// RemoveAssertion removes an assertion by ID
func (s *EvalSuite) RemoveAssertion(id string) {
	assertions := make([]*Assertion, 0)
	for _, a := range s.Assertions {
		if a.ID != id {
			assertions = append(assertions, a)
		}
	}
	s.Assertions = assertions
	s.UpdatedAt = time.Now().UTC()
}

// AssertionResult represents the result of an assertion evaluation
type AssertionResult struct {
	AssertionID string
	Type        AssertionType
	Passed      bool
	Actual      interface{}
	Expected    interface{}
	Operator    string
	Weight      float64
	Message     string
}

// EvalResult represents the result of an eval suite run
type EvalResult struct {
	SuiteID     string
	SuiteName   string
	RunID       string
	Assertions  []*AssertionResult
	Score       float64
	Passed      bool
	TotalWeight float64
	PassedWeight float64
	PassedCount int
	FailedCount int
	RunAt       time.Time
	Duration    time.Duration
	Metadata    map[string]interface{}
}

// NewEvalResult creates a new eval result
func NewEvalResult(suite *EvalSuite) *EvalResult {
	return &EvalResult{
		SuiteID:     suite.ID,
		SuiteName:   suite.Name,
		RunID:       uuid.New().String(),
		Assertions:  make([]*AssertionResult, 0),
		Score:       0.0,
		Passed:      false,
		TotalWeight: 0.0,
		PassedWeight: 0.0,
		PassedCount: 0,
		FailedCount: 0,
		RunAt:       time.Now().UTC(),
		Metadata:    make(map[string]interface{}),
	}
}

// CalculateScore calculates the weighted score based on assertion results
func (r *EvalResult) CalculateScore() {
	var totalWeight, passedWeight float64

	for _, result := range r.Assertions {
		totalWeight += result.Weight
		if result.Passed {
			passedWeight += result.Weight
			r.PassedCount++
		} else {
			r.FailedCount++
		}
	}

	r.TotalWeight = totalWeight
	r.PassedWeight = passedWeight

	if totalWeight > 0 {
		r.Score = passedWeight / totalWeight
	}

	r.Passed = r.PassedCount == len(r.Assertions) && len(r.Assertions) > 0
}

// EvalRunner runs evaluations against test inputs
type EvalRunner struct {
	suites     map[string]*EvalSuite
	results    []*EvalResult
	mu         sync.RWMutex
	logger     *slog.Logger
}

// NewEvalRunner creates a new eval runner
func NewEvalRunner() *EvalRunner {
	return &EvalRunner{
		suites:  make(map[string]*EvalSuite),
		results: make([]*EvalResult, 0),
		logger:  slog.Default(),
	}
}

// RegisterSuite registers an eval suite
func (r *EvalRunner) RegisterSuite(suite *EvalSuite) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.suites[suite.ID] = suite
	r.logger.Info("Registered eval suite",
		slog.String("suite_id", suite.ID),
		slog.String("suite_name", suite.Name))
}

// UnregisterSuite removes an eval suite
func (r *EvalRunner) UnregisterSuite(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.suites, id)
}

// GetSuite retrieves a suite by ID
func (r *EvalRunner) GetSuite(id string) (*EvalSuite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	suite, exists := r.suites[id]
	if !exists {
		return nil, fmt.Errorf("suite not found: %s", id)
	}

	return suite, nil
}

// GetSuiteByName retrieves a suite by name
func (r *EvalRunner) GetSuiteByName(name string) (*EvalSuite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, suite := range r.suites {
		if suite.Name == name {
			return suite, nil
		}
	}

	return nil, fmt.Errorf("suite not found: %s", name)
}

// ListSuites returns all registered suites
func (r *EvalRunner) ListSuites() []*EvalSuite {
	r.mu.RLock()
	defer r.mu.RUnlock()

	suites := make([]*EvalSuite, 0, len(r.suites))
	for _, suite := range r.suites {
		suites = append(suites, suite)
	}

	return suites
}

// RunEval runs an eval suite against test data
func (r *EvalRunner) RunEval(suiteID string, testData map[string]interface{}) (*EvalResult, error) {
	suite, err := r.GetSuite(suiteID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	result := NewEvalResult(suite)

	for _, assertion := range suite.Assertions {
		assertionResult := r.evaluateAssertion(assertion, testData)
		result.Assertions = append(result.Assertions, assertionResult)
	}

	result.CalculateScore()
	result.Duration = time.Since(startTime)

	r.mu.Lock()
	r.results = append(r.results, result)
	r.mu.Unlock()

	// Publish event
	PublishEvent(context.Background(), NewEventWithData(EventTypeEvalSuiteRun, map[string]interface{}{
		"suite_id":     result.SuiteID,
		"suite_name":   result.SuiteName,
		"run_id":       result.RunID,
		"passed":       result.Passed,
		"score":        result.Score,
		"passed_count": result.PassedCount,
		"failed_count": result.FailedCount,
		"duration_ms":  result.Duration.Milliseconds(),
	}))

	return result, nil
}

// evaluateAssertion evaluates a single assertion against test data
func (r *EvalRunner) evaluateAssertion(assertion *Assertion, testData map[string]interface{}) *AssertionResult {
	result := &AssertionResult{
		AssertionID: assertion.ID,
		Type:        assertion.Type,
		Expected:    assertion.Expected,
		Operator:    assertion.Operator,
		Weight:      assertion.Weight,
	}

	// Extract actual value from test data using the target path
	actual := r.extractValue(testData, assertion.Target)
	result.Actual = actual

	// Evaluate the assertion
	passed, message := r.evaluateComparison(actual, assertion.Operator, assertion.Expected)
	result.Passed = passed

	if !passed && assertion.Message != "" {
		result.Message = assertion.Message
	} else if !passed {
		result.Message = message
	}

	// Publish assertion event
	eventType := EventTypeEvalAssertionPass
	if !passed {
		eventType = EventTypeEvalAssertionFail
	}

	PublishEvent(context.Background(), NewEventWithData(eventType, map[string]interface{}{
		"assertion_id": assertion.ID,
		"suite_id":     assertion.ID,
		"passed":       passed,
		"actual":        fmt.Sprintf("%v", actual),
		"expected":      fmt.Sprintf("%v", assertion.Expected),
		"operator":      assertion.Operator,
	}))

	return result
}

// extractValue extracts a value from test data using a simple path notation
func (r *EvalRunner) extractValue(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	// Remove leading $. if present
	if strings.HasPrefix(path, "$.") {
		path = path[2:]
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Handle array indexing: field[index]
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			fieldPart := strings.Split(part, "[")
			field := fieldPart[0]

			if m, ok := current.(map[string]interface{}); ok {
				current = m[field]
			} else {
				return nil
			}

			idxStr := strings.TrimSuffix(fieldPart[1], "]")
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil
			}

			if arr, ok := current.([]interface{}); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			if m, ok := current.(map[string]interface{}); ok {
				current = m[part]
			} else {
				return nil
			}
		}
	}

	return current
}

// evaluateComparison evaluates a comparison operation
func (r *EvalRunner) evaluateComparison(actual interface{}, operator string, expected interface{}) (bool, string) {
	switch operator {
	case OpEq:
		return r.equals(actual, expected), fmt.Sprintf("expected %v to equal %v", actual, expected)
	case OpNe:
		return !r.equals(actual, expected), fmt.Sprintf("expected %v to not equal %v", actual, expected)
	case OpGt:
		return r.greaterThan(actual, expected), fmt.Sprintf("expected %v to be greater than %v", actual, expected)
	case OpGte:
		return r.greaterOrEqual(actual, expected), fmt.Sprintf("expected %v to be greater than or equal to %v", actual, expected)
	case OpLt:
		return r.lessThan(actual, expected), fmt.Sprintf("expected %v to be less than %v", actual, expected)
	case OpLte:
		return r.lessOrEqual(actual, expected), fmt.Sprintf("expected %v to be less than or equal to %v", actual, expected)
	case OpContains:
		return r.contains(actual, expected), fmt.Sprintf("expected %v to contain %v", actual, expected)
	case OpNotContains:
		return !r.contains(actual, expected), fmt.Sprintf("expected %v to not contain %v", actual, expected)
	case OpMatches:
		return r.matches(actual, expected), fmt.Sprintf("expected %v to match pattern %v", actual, expected)
	case OpStartsWith:
		return r.startsWith(actual, expected), fmt.Sprintf("expected %v to start with %v", actual, expected)
	case OpEndsWith:
		return r.endsWith(actual, expected), fmt.Sprintf("expected %v to end with %v", actual, expected)
	case OpIn:
		return r.in(actual, expected), fmt.Sprintf("expected %v to be in %v", actual, expected)
	case OpNotIn:
		return !r.in(actual, expected), fmt.Sprintf("expected %v to not be in %v", actual, expected)
	default:
		return false, fmt.Sprintf("unknown operator: %s", operator)
	}
}

// Helper methods for comparison
func (r *EvalRunner) equals(actual, expected interface{}) bool {
	// Try numeric comparison first
	actualNum, actualIsNum := toFloat64(actual)
	expectedNum, expectedIsNum := toFloat64(expected)

	if actualIsNum && expectedIsNum {
		return actualNum == expectedNum
	}

	// Fall back to string comparison
	actualStr := strings.ToLower(fmt.Sprintf("%v", actual))
	expectedStr := strings.ToLower(fmt.Sprintf("%v", expected))
	return actualStr == expectedStr
}

func (r *EvalRunner) greaterThan(actual, expected interface{}) bool {
	actualNum, ok1 := toFloat64(actual)
	expectedNum, ok2 := toFloat64(expected)
	if ok1 && ok2 {
		return actualNum > expectedNum
	}
	return false
}

func (r *EvalRunner) greaterOrEqual(actual, expected interface{}) bool {
	actualNum, ok1 := toFloat64(actual)
	expectedNum, ok2 := toFloat64(expected)
	if ok1 && ok2 {
		return actualNum >= expectedNum
	}
	return false
}

func (r *EvalRunner) lessThan(actual, expected interface{}) bool {
	actualNum, ok1 := toFloat64(actual)
	expectedNum, ok2 := toFloat64(expected)
	if ok1 && ok2 {
		return actualNum < expectedNum
	}
	return false
}

func (r *EvalRunner) lessOrEqual(actual, expected interface{}) bool {
	actualNum, ok1 := toFloat64(actual)
	expectedNum, ok2 := toFloat64(expected)
	if ok1 && ok2 {
		return actualNum <= expectedNum
	}
	return false
}

func (r *EvalRunner) contains(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.Contains(actualStr, expectedStr)
}

func (r *EvalRunner) matches(actual, expected interface{}) bool {
	pattern, ok := expected.(string)
	if !ok {
		return false
	}
	actualStr := fmt.Sprintf("%v", actual)
	matched, err := regexp.MatchString(pattern, actualStr)
	if err != nil {
		return false
	}
	return matched
}

func (r *EvalRunner) startsWith(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.HasPrefix(actualStr, expectedStr)
}

func (r *EvalRunner) endsWith(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.HasSuffix(actualStr, expectedStr)
}

func (r *EvalRunner) in(actual, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)

	switch exp := expected.(type) {
	case []interface{}:
		for _, v := range exp {
			if strings.ToLower(actualStr) == strings.ToLower(fmt.Sprintf("%v", v)) {
				return true
			}
		}
		return false
	case []string:
		for _, v := range exp {
			if strings.EqualFold(actualStr, v) {
				return true
			}
		}
		return false
	default:
		return r.equals(actual, expected)
	}
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// GetResults returns all eval results
func (r *EvalRunner) GetResults(limit int) []*EvalResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := r.results
	if limit > 0 && limit < len(results) {
		return results[len(results)-limit:]
	}

	return results
}

// GetResultsBySuite returns results for a specific suite
func (r *EvalRunner) GetResultsBySuite(suiteID string, limit int) []*EvalResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]*EvalResult, 0)
	for _, result := range r.results {
		if result.SuiteID == suiteID {
			results = append(results, result)
		}
	}

	if limit > 0 && limit < len(results) {
		return results[len(results)-limit:]
	}

	return results
}

// Golden Sets - Pre-defined eval suites for common use cases

// CodingGoldenSet is a golden set for evaluating code generation
var CodingGoldenSet = NewEvalSuite("coding-golden-set", "Golden set for evaluating code generation quality")

func init() {
	CodingGoldenSet.AddAssertion(NewAssertion(
		AssertOutput, "$.code", OpContains, "func ",
	).SetWeight(0.3))

	CodingGoldenSet.AddAssertion(NewAssertion(
		AssertQuality, "$.score", OpGt, 0.8,
	).SetWeight(0.4))

	CodingGoldenSet.AddAssertion(NewAssertion(
		AssertLatency, "$.latency_ms", OpLt, 5000,
	).SetWeight(0.3))
}

// MathGoldenSet is a golden set for evaluating math problem solving
var MathGoldenSet = NewEvalSuite("math-golden-set", "Golden set for evaluating math problem solving")

func init() {
	MathGoldenSet.AddAssertion(NewAssertion(
		AssertOutput, "$.answer", OpMatches, `^-?\d+\.?\d*$`,
	).SetWeight(0.2))

	MathGoldenSet.AddAssertion(NewAssertion(
		AssertQuality, "$.accuracy", OpGte, 0.95,
	).SetWeight(0.5))

	MathGoldenSet.AddAssertion(NewAssertion(
		AssertLatency, "$.latency_ms", OpLt, 3000,
	).SetWeight(0.3))
}

// ReasoningGoldenSet is a golden set for evaluating reasoning capabilities
var ReasoningGoldenSet = NewEvalSuite("reasoning-golden-set", "Golden set for evaluating chain-of-thought reasoning")

func init() {
	ReasoningGoldenSet.AddAssertion(NewAssertion(
		AssertOutput, "$.reasoning", OpContains, "because",
	).SetWeight(0.3))

	ReasoningGoldenSet.AddAssertion(NewAssertion(
		AssertQuality, "$.coherence_score", OpGt, 0.7,
	).SetWeight(0.4))

	ReasoningGoldenSet.AddAssertion(NewAssertion(
		AssertFormat, "$.steps", OpContains, "step",
	).SetWeight(0.3))
}

// SummarizationGoldenSet is a golden set for evaluating summarization
var SummarizationGoldenSet = NewEvalSuite("summarization-golden-set", "Golden set for evaluating text summarization")

func init() {
	SummarizationGoldenSet.AddAssertion(NewAssertion(
		AssertQuality, "$.compression_ratio", OpGte, 0.3,
	).SetWeight(0.25))

	SummarizationGoldenSet.AddAssertion(NewAssertion(
		AssertQuality, "$.rouge_score", OpGt, 0.5,
	).SetWeight(0.35))

	SummarizationGoldenSet.AddAssertion(NewAssertion(
		AssertLatency, "$.latency_ms", OpLt, 2000,
	).SetWeight(0.2))

	SummarizationGoldenSet.AddAssertion(NewAssertion(
		AssertContent, "$.summary", OpNotContains, "Lorem ipsum",
	).SetWeight(0.2))
}

// Global eval runner instance
var globalEvalRunner *EvalRunner

// InitGlobalEvalRunner initializes the global eval runner with default suites
func InitGlobalEvalRunner() {
	globalEvalRunner = NewEvalRunner()

	// Register default golden sets
	globalEvalRunner.RegisterSuite(CodingGoldenSet)
	globalEvalRunner.RegisterSuite(MathGoldenSet)
	globalEvalRunner.RegisterSuite(ReasoningGoldenSet)
	globalEvalRunner.RegisterSuite(SummarizationGoldenSet)
}

// GetGlobalEvalRunner returns the global eval runner
func GetGlobalEvalRunner() *EvalRunner {
	if globalEvalRunner == nil {
		InitGlobalEvalRunner()
	}
	return globalEvalRunner
}

// RunGlobalEval runs an eval suite by name on the global runner
func RunGlobalEval(suiteName string, testData map[string]interface{}) (*EvalResult, error) {
	return GetGlobalEvalRunner().RunEval(suiteName, testData)
}

// RegisterGlobalSuite registers a suite on the global runner
func RegisterGlobalSuite(suite *EvalSuite) {
	GetGlobalEvalRunner().RegisterSuite(suite)
}
