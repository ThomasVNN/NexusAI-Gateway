package evals

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

// EvalTest represents a single evaluation test
type EvalTest struct {
	ID          string
	Name        string
	Description string
	Category    string
	Prompt      string
	Expected    ExpectedOutput
	Weight      float64
	CreatedAt   time.Time
}

// ExpectedOutput defines expected test output
type ExpectedOutput struct {
	Type       string // "exact", "contains", "regex", "semantic"
	Value      string
	MinScore   float64
	MaxLatency time.Duration
}

// EvalResult contains test evaluation results
type EvalResult struct {
	TestID      string
	Passed      bool
	Score       float64
	Latency     time.Duration
	Output      string
	Error       string
	EvaluatedAt time.Time
}

// Benchmark represents a performance benchmark
type Benchmark struct {
	ID        string
	Name      string
	Category  string
	Target    string
	Threshold float64
	Unit      string
	CreatedAt time.Time
}

// BenchmarkResult contains benchmark results
type BenchmarkResult struct {
	BenchmarkID string
	Value       float64
	Passed      bool
	Timestamp   time.Time
}

// Regression represents a detected regression
type Regression struct {
	ID            string
	TestID        string
	PreviousScore float64
	CurrentScore  float64
	Delta         float64
	DetectedAt    time.Time
	Status        string // "open", "acknowledged", "resolved"
}

// EvalSuite represents a collection of tests
type EvalSuite struct {
	ID          string
	Name        string
	Description string
	Tests       []*EvalTest
	CreatedAt   time.Time
}

// TrendPoint represents a point in the trend
type TrendPoint struct {
	Timestamp time.Time
	Score     float64
}

// EvalService provides evaluation operations
type EvalService struct {
	db *postgres.DB
}

// NewEvalService creates a new evaluation service
func NewEvalService(db *postgres.DB) *EvalService {
	return &EvalService{db: db}
}

// CreateTest creates a new evaluation test
func (s *EvalService) CreateTest(ctx context.Context, test *EvalTest) error {
	if test.ID == "" {
		test.ID = uuid.New().String()
	}
	if test.CreatedAt.IsZero() {
		test.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO eval_tests (id, name, description, category, prompt, expected_type, expected_value, expected_min_score, expected_max_latency, weight, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			prompt = EXCLUDED.prompt,
			expected_type = EXCLUDED.expected_type,
			expected_value = EXCLUDED.expected_value,
			expected_min_score = EXCLUDED.expected_min_score,
			expected_max_latency = EXCLUDED.expected_max_latency,
			weight = EXCLUDED.weight
	`

	_, err := s.db.ExecContext(ctx, query,
		test.ID, test.Name, test.Description, test.Category, test.Prompt,
		test.Expected.Type, test.Expected.Value, test.Expected.MinScore,
		int64(test.Expected.MaxLatency.Milliseconds()), test.Weight, test.CreatedAt,
	)
	return err
}

// GetTest retrieves a test by ID
func (s *EvalService) GetTest(ctx context.Context, testID string) (*EvalTest, error) {
	query := `
		SELECT id, name, description, category, prompt, expected_type, expected_value, 
		       expected_min_score, expected_max_latency, weight, created_at
		FROM eval_tests WHERE id = $1
	`

	test := &EvalTest{Expected: ExpectedOutput{}}
	err := s.db.QueryRowContext(ctx, query, testID).Scan(
		&test.ID, &test.Name, &test.Description, &test.Category, &test.Prompt,
		&test.Expected.Type, &test.Expected.Value, &test.Expected.MinScore,
		&test.Expected.MaxLatency, &test.Weight, &test.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	test.Expected.MaxLatency = time.Duration(test.Expected.MaxLatency) * time.Millisecond
	return test, nil
}

// RunTest runs a single evaluation test
func (s *EvalService) RunTest(ctx context.Context, testID string, response string) (*EvalResult, error) {
	test, err := s.GetTest(ctx, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	start := time.Now()
	result := &EvalResult{
		TestID:      testID,
		Output:      response,
		EvaluatedAt: time.Now(),
	}

	// Evaluate based on expected type
	passed, score := evaluateOutput(response, test.Expected)
	result.Passed = passed
	result.Score = score
	result.Latency = time.Since(start)

	// Save result
	if err := s.saveEvalResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	return result, nil
}

// evaluateOutput evaluates response against expected output
func evaluateOutput(response string, expected ExpectedOutput) (bool, float64) {
	switch expected.Type {
	case "exact":
		return response == expected.Value, boolToScore(response == expected.Value)
	case "contains":
		passed := strings.Contains(strings.ToLower(response), strings.ToLower(expected.Value))
		return passed, boolToScore(passed)
	case "regex":
		matched, _ := regexp.MatchString(expected.Value, response)
		return matched, boolToScore(matched)
	case "semantic":
		// Semantic scoring is placeholder - would integrate with embedding model
		passed := expected.MinScore >= 0
		return passed, expected.MinScore
	default:
		return false, 0
	}
}

// boolToScore converts boolean pass/fail to 1.0/0.0
func boolToScore(passed bool) float64 {
	if passed {
		return 1.0
	}
	return 0.0
}

// saveEvalResult saves an evaluation result to the database
func (s *EvalService) saveEvalResult(ctx context.Context, result *EvalResult) error {
	query := `
		INSERT INTO eval_results (test_id, passed, score, latency_ms, output, error, evaluated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		result.TestID, result.Passed, result.Score,
		int64(result.Latency.Milliseconds()), result.Output, result.Error, result.EvaluatedAt,
	)
	return err
}

// RunSuite runs all tests in a suite
func (s *EvalService) RunSuite(ctx context.Context, suiteID string) ([]*EvalResult, error) {
	// Get all tests for this suite
	query := `
		SELECT et.id, et.name, et.description, et.category, et.prompt, 
		       et.expected_type, et.expected_value, et.expected_min_score, 
		       et.expected_max_latency, et.weight, et.created_at
		FROM eval_tests et
		INNER JOIN eval_suite_tests est ON est.test_id = et.id
		WHERE est.suite_id = $1
	`

	rows, err := s.db.QueryContext(ctx, query, suiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*EvalResult
	for rows.Next() {
		test := &EvalTest{Expected: ExpectedOutput{}}
		err := rows.Scan(
			&test.ID, &test.Name, &test.Description, &test.Category, &test.Prompt,
			&test.Expected.Type, &test.Expected.Value, &test.Expected.MinScore,
			&test.Expected.MaxLatency, &test.Weight, &test.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		test.Expected.MaxLatency = time.Duration(test.Expected.MaxLatency) * time.Millisecond

		// For suite run, we'd need actual responses - placeholder
		result := &EvalResult{
			TestID:      test.ID,
			Score:       test.Expected.MinScore,
			EvaluatedAt: time.Now(),
		}
		results = append(results, result)
	}

	return results, nil
}

// CreateBenchmark creates a new benchmark
func (s *EvalService) CreateBenchmark(ctx context.Context, b *Benchmark) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO benchmarks (id, name, category, target, threshold, unit, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			target = EXCLUDED.target,
			threshold = EXCLUDED.threshold,
			unit = EXCLUDED.unit
	`

	_, err := s.db.ExecContext(ctx, query, b.ID, b.Name, b.Category, b.Target, b.Threshold, b.Unit, b.CreatedAt)
	return err
}

// RunBenchmark runs a benchmark and records result
func (s *EvalService) RunBenchmark(ctx context.Context, benchmarkID string, value float64) (*BenchmarkResult, error) {
	// Get benchmark threshold
	var threshold float64
	err := s.db.QueryRowContext(ctx, "SELECT threshold FROM benchmarks WHERE id = $1", benchmarkID).Scan(&threshold)
	if err != nil {
		return nil, err
	}

	result := &BenchmarkResult{
		BenchmarkID: benchmarkID,
		Value:      value,
		Passed:     value >= threshold,
		Timestamp:  time.Now(),
	}

	// Save result
	query := `
		INSERT INTO benchmark_results (benchmark_id, value, passed, timestamp)
		VALUES ($1, $2, $3, $4)
	`
	_, err = s.db.ExecContext(ctx, query, result.BenchmarkID, result.Value, result.Passed, result.Timestamp)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DetectRegression checks for score regressions
func (s *EvalService) DetectRegression(ctx context.Context, testID string, previousScore, currentScore float64) (*Regression, error) {
	delta := currentScore - previousScore
	threshold := -0.05 // 5% regression threshold

	regression := &Regression{
		ID:            uuid.New().String(),
		TestID:        testID,
		PreviousScore: previousScore,
		CurrentScore:  currentScore,
		Delta:         delta,
		DetectedAt:    time.Now(),
	}

	if delta < threshold {
		regression.Status = "open"
	} else {
		regression.Status = "resolved"
	}

	query := `
		INSERT INTO regressions (id, test_id, previous_score, current_score, delta, detected_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		regression.ID, regression.TestID, regression.PreviousScore,
		regression.CurrentScore, regression.Delta, regression.DetectedAt, regression.Status,
	)
	if err != nil {
		return nil, err
	}

	return regression, nil
}

// GetRegressionHistory returns regression history
func (s *EvalService) GetRegressionHistory(ctx context.Context, limit int) ([]*Regression, error) {
	query := `
		SELECT id, test_id, previous_score, current_score, delta, detected_at, status
		FROM regressions
		ORDER BY detected_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regressions []*Regression
	for rows.Next() {
		r := &Regression{}
		err := rows.Scan(&r.ID, &r.TestID, &r.PreviousScore, &r.CurrentScore, &r.Delta, &r.DetectedAt, &r.Status)
		if err != nil {
			return nil, err
		}
		regressions = append(regressions, r)
	}

	return regressions, nil
}

// GenerateTests generates tests from prompts
func (s *EvalService) GenerateTests(ctx context.Context, prompt string, count int) ([]*EvalTest, error) {
	// This is a placeholder - in production, would integrate with LLM to generate tests
	// For now, create placeholder tests
	tests := make([]*EvalTest, count)
	for i := 0; i < count; i++ {
		test := &EvalTest{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("Generated Test %d", i+1),
			Description: fmt.Sprintf("Auto-generated test based on: %s", prompt),
			Category:    "auto-generated",
			Prompt:      prompt,
			Expected: ExpectedOutput{
				Type:     "contains",
				Value:    "test",
				MinScore: 0.7,
			},
			Weight:    1.0,
			CreatedAt: time.Now(),
		}
		tests[i] = test

		if err := s.CreateTest(ctx, test); err != nil {
			return nil, err
		}
	}

	return tests, nil
}

// TrackTrend tracks score trends over time
func (s *EvalService) TrackTrend(ctx context.Context, testID string, period string) ([]TrendPoint, error) {
	var interval string
	switch period {
	case "hour":
		interval = "1 hour"
	case "day":
		interval = "1 day"
	case "week":
		interval = "1 week"
	default:
		interval = "1 day"
	}

	query := fmt.Sprintf(`
		SELECT time_bucket('%s', evaluated_at) AS bucket, AVG(score) as avg_score
		FROM eval_results
		WHERE test_id = $1 AND evaluated_at > NOW() - INTERVAL '30 days'
		GROUP BY bucket
		ORDER BY bucket DESC
		LIMIT 100
	`, interval)

	rows, err := s.db.QueryContext(ctx, query, testID)
	if err != nil {
		// Fallback to simple query if time_bucket not available
		query = `
			SELECT evaluated_at, score
			FROM eval_results
			WHERE test_id = $1
			ORDER BY evaluated_at DESC
			LIMIT 100
		`
		rows, err = s.db.QueryContext(ctx, query, testID)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var points []TrendPoint
	for rows.Next() {
		var tp TrendPoint
		if err := rows.Scan(&tp.Timestamp, &tp.Score); err != nil {
			return nil, err
		}
		points = append(points, tp)
	}

	return points, nil
}

// GetTestResults retrieves all results for a test
func (s *EvalService) GetTestResults(ctx context.Context, testID string, limit int) ([]*EvalResult, error) {
	query := `
		SELECT test_id, passed, score, latency_ms, output, error, evaluated_at
		FROM eval_results
		WHERE test_id = $1
		ORDER BY evaluated_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, testID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*EvalResult
	for rows.Next() {
		r := &EvalResult{}
		var latencyMs int64
		err := rows.Scan(&r.TestID, &r.Passed, &r.Score, &latencyMs, &r.Output, &r.Error, &r.EvaluatedAt)
		if err != nil {
			return nil, err
		}
		r.Latency = time.Duration(latencyMs) * time.Millisecond
		results = append(results, r)
	}

	return results, nil
}

// UpdateRegressionStatus updates regression status
func (s *EvalService) UpdateRegressionStatus(ctx context.Context, regressionID, status string) error {
	query := `UPDATE regressions SET status = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, regressionID)
	return err
}

// CreateSuite creates a new eval suite
func (s *EvalService) CreateSuite(ctx context.Context, suite *EvalSuite) error {
	if suite.ID == "" {
		suite.ID = uuid.New().String()
	}
	if suite.CreatedAt.IsZero() {
		suite.CreatedAt = time.Now()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create suite
	query := `
		INSERT INTO eval_suites (id, name, description, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description
	`
	_, err = tx.ExecContext(ctx, query, suite.ID, suite.Name, suite.Description, suite.CreatedAt)
	if err != nil {
		return err
	}

	// Add tests to suite
	for _, test := range suite.Tests {
		linkQuery := `
			INSERT INTO eval_suite_tests (suite_id, test_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`
		_, err = tx.ExecContext(ctx, linkQuery, suite.ID, test.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSuite retrieves a suite with its tests
func (s *EvalService) GetSuite(ctx context.Context, suiteID string) (*EvalSuite, error) {
	query := `SELECT id, name, description, created_at FROM eval_suites WHERE id = $1`
	suite := &EvalSuite{}
	err := s.db.QueryRowContext(ctx, query, suiteID).Scan(&suite.ID, &suite.Name, &suite.Description, &suite.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("suite not found: %s", suiteID)
		}
		return nil, err
	}

	// Get tests
	testQuery := `
		SELECT et.id, et.name, et.description, et.category, et.prompt,
		       et.expected_type, et.expected_value, et.expected_min_score,
		       et.expected_max_latency, et.weight, et.created_at
		FROM eval_tests et
		INNER JOIN eval_suite_tests est ON est.test_id = et.id
		WHERE est.suite_id = $1
	`
	rows, err := s.db.QueryContext(ctx, testQuery, suiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		test := &EvalTest{Expected: ExpectedOutput{}}
		err := rows.Scan(
			&test.ID, &test.Name, &test.Description, &test.Category, &test.Prompt,
			&test.Expected.Type, &test.Expected.Value, &test.Expected.MinScore,
			&test.Expected.MaxLatency, &test.Weight, &test.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		test.Expected.MaxLatency = time.Duration(test.Expected.MaxLatency) * time.Millisecond
		suite.Tests = append(suite.Tests, test)
	}

	return suite, nil
}
