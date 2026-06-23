package observability

import (
	"testing"
)

func TestEvalSuite_AddAssertion(t *testing.T) {
	suite := NewEvalSuite("test-suite", "Test eval suite")

	t.Run("adds assertion to suite", func(t *testing.T) {
		initialCount := len(suite.Assertions)

		assertion := NewAssertion(AssertOutput, "$.code", OpContains, "func ")
		suite.AddAssertion(assertion)

		if len(suite.Assertions) != initialCount+1 {
			t.Errorf("Expected %d assertions, got %d", initialCount+1, len(suite.Assertions))
		}
	})

	t.Run("updates timestamp on add", func(t *testing.T) {
		original := suite.UpdatedAt
		assertion := NewAssertion(AssertQuality, "$.score", OpGt, 0.8)
		suite.AddAssertion(assertion)

		if !suite.UpdatedAt.After(original) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})
}

func TestEvalSuite_RemoveAssertion(t *testing.T) {
	suite := NewEvalSuite("test-suite", "Test eval suite")

	assertion := NewAssertion(AssertOutput, "$.code", OpContains, "func ")
	suite.AddAssertion(assertion)

	if len(suite.Assertions) != 1 {
		t.Fatalf("Expected 1 assertion, got %d", len(suite.Assertions))
	}

	suite.RemoveAssertion(assertion.ID)

	if len(suite.Assertions) != 0 {
		t.Errorf("Expected 0 assertions, got %d", len(suite.Assertions))
	}
}

func TestEvalRunner_RegisterSuite(t *testing.T) {
	runner := NewEvalRunner()

	t.Run("registers suite", func(t *testing.T) {
		suite := NewEvalSuite("test-suite", "Test eval suite")
		runner.RegisterSuite(suite)

		retrieved, err := runner.GetSuite(suite.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if retrieved.ID != suite.ID {
			t.Errorf("Expected suite ID %s, got %s", suite.ID, retrieved.ID)
		}
	})

	t.Run("returns error for non-existent suite", func(t *testing.T) {
		_, err := runner.GetSuite("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent suite")
		}
	})

	t.Run("lists all suites", func(t *testing.T) {
		initialCount := len(runner.ListSuites())

		runner.RegisterSuite(NewEvalSuite("suite-1", ""))
		runner.RegisterSuite(NewEvalSuite("suite-2", ""))

		suites := runner.ListSuites()
		if len(suites) != initialCount+2 {
			t.Errorf("Expected %d suites, got %d", initialCount+2, len(suites))
		}
	})

	t.Run("gets suite by name", func(t *testing.T) {
		suite := NewEvalSuite("unique-name-suite", "")
		runner.RegisterSuite(suite)

		retrieved, err := runner.GetSuiteByName("unique-name-suite")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if retrieved.Name != "unique-name-suite" {
			t.Errorf("Expected name 'unique-name-suite', got %s", retrieved.Name)
		}
	})
}

func TestEvalRunner_RunEval(t *testing.T) {
	runner := NewEvalRunner()
	suite := NewEvalSuite("coding-test", "Test coding suite")

	suite.AddAssertion(NewAssertion(AssertOutput, "$.code", OpContains, "func ").SetWeight(0.5))
	suite.AddAssertion(NewAssertion(AssertQuality, "$.score", OpGt, 0.8).SetWeight(0.3))
	suite.AddAssertion(NewAssertion(AssertLatency, "$.latency_ms", OpLt, 5000).SetWeight(0.2))

	runner.RegisterSuite(suite)

	t.Run("runs eval with passing assertions", func(t *testing.T) {
		testData := map[string]interface{}{
			"code":       "func main() {}",
			"score":      0.95,
			"latency_ms": 1000,
		}

		result, err := runner.RunEval(suite.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result to be returned")
		}

		if !result.Passed {
			t.Error("Expected all assertions to pass")
		}

		if result.Score != 1.0 {
			t.Errorf("Expected score 1.0, got %f", result.Score)
		}

		if result.PassedCount != 3 {
			t.Errorf("Expected 3 passed assertions, got %d", result.PassedCount)
		}

		if result.FailedCount != 0 {
			t.Errorf("Expected 0 failed assertions, got %d", result.FailedCount)
		}
	})

	t.Run("runs eval with failing assertions", func(t *testing.T) {
		testData := map[string]interface{}{
			"code":       "no function here",
			"score":      0.5,
			"latency_ms": 1000,
		}

		result, err := runner.RunEval(suite.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.Passed {
			t.Error("Expected some assertions to fail")
		}

		if result.Score >= 1.0 {
			t.Errorf("Expected score < 1.0, got %f", result.Score)
		}
	})

	t.Run("calculates weighted score correctly", func(t *testing.T) {
		testData := map[string]interface{}{
			"code":       "func main() {}", // passes (weight 0.5)
			"score":      0.9,              // passes (weight 0.3)
			"latency_ms": 10000,             // fails (weight 0.2)
		}

		result, err := runner.RunEval(suite.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Expected: (0.5 + 0.3) / 1.0 = 0.8
		expectedScore := 0.8
		if result.Score != expectedScore {
			t.Errorf("Expected score %f, got %f", expectedScore, result.Score)
		}

		if result.PassedCount != 2 {
			t.Errorf("Expected 2 passed, got %d", result.PassedCount)
		}

		if result.FailedCount != 1 {
			t.Errorf("Expected 1 failed, got %d", result.FailedCount)
		}
	})
}

func TestAssertionOperators(t *testing.T) {
	runner := NewEvalRunner()
	suite := NewEvalSuite("operator-test", "Test operators")

	runner.RegisterSuite(suite)

	// Test basic string operations
	t.Run("string contains works", func(t *testing.T) {
		assertion := NewAssertion(AssertOutput, "$.value", OpContains, "hello")
		suiteLocal := NewEvalSuite("contains-test", "")
		suiteLocal.AddAssertion(assertion)
		runner.RegisterSuite(suiteLocal)

		result, err := runner.RunEval(suiteLocal.ID, map[string]interface{}{"value": "hello world"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected contains to pass")
		}
	})

	t.Run("string equals works", func(t *testing.T) {
		assertion := NewAssertion(AssertOutput, "$.value", OpEq, "test")
		suiteLocal := NewEvalSuite("eq-test", "")
		suiteLocal.AddAssertion(assertion)
		runner.RegisterSuite(suiteLocal)

		result, err := runner.RunEval(suiteLocal.ID, map[string]interface{}{"value": "test"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected eq to pass")
		}
	})

	t.Run("matches regex works", func(t *testing.T) {
		assertion := NewAssertion(AssertOutput, "$.value", OpMatches, "^[a-z]+$")
		suiteLocal := NewEvalSuite("matches-test", "")
		suiteLocal.AddAssertion(assertion)
		runner.RegisterSuite(suiteLocal)

		result, err := runner.RunEval(suiteLocal.ID, map[string]interface{}{"value": "hello"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected matches to pass")
		}
	})
}

func TestExtractValue(t *testing.T) {
	runner := NewEvalRunner()

	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		expected interface{}
	}{
		{"simple field", map[string]interface{}{"key": "value"}, "key", "value"},
		{"simple field with $.", map[string]interface{}{"key": "value"}, "$.key", "value"},
		{"nested field", map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}}, "outer.inner", "value"},
		{"array index", map[string]interface{}{"arr": []interface{}{"a", "b", "c"}}, "arr[1]", "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.extractValue(tt.data, tt.path)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGoldenSets(t *testing.T) {
	t.Run("coding golden set passes with valid code", func(t *testing.T) {
		testData := map[string]interface{}{
			"code":       "func main() { fmt.Println(\"Hello\") }",
			"score":      0.9,
			"latency_ms": 1000,
		}

		result, err := GetGlobalEvalRunner().RunEval(CodingGoldenSet.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected coding golden set to pass")
		}
	})

	t.Run("math golden set passes with valid answer", func(t *testing.T) {
		testData := map[string]interface{}{
			"answer":    "42",
			"accuracy":  0.98,
			"latency_ms": 500,
		}

		result, err := GetGlobalEvalRunner().RunEval(MathGoldenSet.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected math golden set to pass")
		}
	})

	t.Run("reasoning golden set passes with valid reasoning", func(t *testing.T) {
		testData := map[string]interface{}{
			"reasoning":       "The answer is 42 because...",
			"coherence_score": 0.85,
			"steps":           "First step, second step",
		}

		result, err := GetGlobalEvalRunner().RunEval(ReasoningGoldenSet.ID, testData)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected reasoning golden set to pass")
		}
	})

	t.Run("golden set assertion works", func(t *testing.T) {
		// Test that assertions work with the golden set style
		suite := NewEvalSuite("test-golden", "")
		suite.AddAssertion(NewAssertion(
			AssertContent, "$.summary", OpContains, "summary",
		).SetWeight(1.0))

		runner := NewEvalRunner()
		runner.RegisterSuite(suite)

		result, err := runner.RunEval(suite.ID, map[string]interface{}{
			"summary": "This is a valid summary for testing",
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !result.Passed {
			t.Error("Expected assertion to pass")
		}
	})
}

func TestEvalRunner_Results(t *testing.T) {
	runner := NewEvalRunner()
	suite := NewEvalSuite("results-test", "")

	suite.AddAssertion(NewAssertion(AssertOutput, "$.value", OpEq, "test"))
	runner.RegisterSuite(suite)

	// Run multiple evals
	runner.RunEval(suite.ID, map[string]interface{}{"value": "test"})
	runner.RunEval(suite.ID, map[string]interface{}{"value": "wrong"})

	t.Run("returns all results", func(t *testing.T) {
		results := runner.GetResults(0)
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("limits results", func(t *testing.T) {
		results := runner.GetResults(1)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("returns results by suite", func(t *testing.T) {
		results := runner.GetResultsBySuite(suite.ID, 0)
		if len(results) != 2 {
			t.Errorf("Expected 2 results for suite, got %d", len(results))
		}
	})
}

func TestGlobalEvalRunner(t *testing.T) {
	t.Run("global runner is initialized", func(t *testing.T) {
		runner := GetGlobalEvalRunner()
		if runner == nil {
			t.Fatal("Expected global runner to be initialized")
		}
	})

	t.Run("global runner has default suites", func(t *testing.T) {
		runner := GetGlobalEvalRunner()
		suites := runner.ListSuites()

		if len(suites) < 4 {
			t.Errorf("Expected at least 4 default suites, got %d", len(suites))
		}
	})

	t.Run("run eval by ID", func(t *testing.T) {
		// Test with a locally registered suite
		suite := NewEvalSuite("test-eval-suite", "")
		suite.AddAssertion(NewAssertion(AssertOutput, "$.value", OpEq, "test"))

		runner := NewEvalRunner()
		runner.RegisterSuite(suite)

		result, err := runner.RunEval(suite.ID, map[string]interface{}{
			"value": "test",
		})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result to be returned")
		}
	})
}

func TestAssertionResult(t *testing.T) {
	result := &AssertionResult{
		AssertionID: "test-id",
		Type:        AssertOutput,
		Passed:      true,
		Actual:      "hello",
		Expected:    "hello",
		Operator:    OpEq,
		Weight:      1.0,
	}

	if result.Weight != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", result.Weight)
	}
}

func TestEvalResult_CalculateScore(t *testing.T) {
	t.Run("calculates score correctly with all passed", func(t *testing.T) {
		suite := NewEvalSuite("test", "")
		result := NewEvalResult(suite)

		result.Assertions = []*AssertionResult{
			{Passed: true, Weight: 0.5},
			{Passed: true, Weight: 0.5},
		}

		result.CalculateScore()

		if result.Score != 1.0 {
			t.Errorf("Expected score 1.0, got %f", result.Score)
		}

		if result.PassedCount != 2 {
			t.Errorf("Expected 2 passed, got %d", result.PassedCount)
		}
	})

	t.Run("calculates score correctly with partial pass", func(t *testing.T) {
		suite := NewEvalSuite("test", "")
		result := NewEvalResult(suite)

		result.Assertions = []*AssertionResult{
			{Passed: true, Weight: 0.7},
			{Passed: false, Weight: 0.3},
		}

		result.CalculateScore()

		if result.Score != 0.7 {
			t.Errorf("Expected score 0.7, got %f", result.Score)
		}
	})

	t.Run("marks as passed only when all pass", func(t *testing.T) {
		suite := NewEvalSuite("test", "")
		result := NewEvalResult(suite)

		result.Assertions = []*AssertionResult{
			{Passed: true, Weight: 0.5},
			{Passed: false, Weight: 0.5},
		}

		result.CalculateScore()

		if result.Passed {
			t.Error("Expected result to not be passed")
		}
	})

	t.Run("handles empty assertions", func(t *testing.T) {
		suite := NewEvalSuite("test", "")
		result := NewEvalResult(suite)

		result.CalculateScore()

		if result.Passed {
			t.Error("Expected empty result to not be passed")
		}

		if result.Score != 0 {
			t.Errorf("Expected score 0, got %f", result.Score)
		}
	})
}
