-- Eval & Quality Framework Schema
-- Version: 1.0
-- Description: Database schema for evaluation tests, benchmarks, and regression tracking

-- Evaluation Tests
CREATE TABLE IF NOT EXISTS eval_tests (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    prompt TEXT NOT NULL,
    expected_type VARCHAR(50) NOT NULL, -- 'exact', 'contains', 'regex', 'semantic'
    expected_value TEXT,
    expected_min_score DECIMAL(5, 4) DEFAULT 0.0,
    expected_max_latency BIGINT DEFAULT 0, -- milliseconds
    weight DECIMAL(5, 4) DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Evaluation Results
CREATE TABLE IF NOT EXISTS eval_results (
    id SERIAL PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL REFERENCES eval_tests(id),
    passed BOOLEAN NOT NULL,
    score DECIMAL(5, 4) NOT NULL,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    output TEXT,
    error TEXT,
    evaluated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Benchmarks
CREATE TABLE IF NOT EXISTS benchmarks (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    target VARCHAR(255) NOT NULL,
    threshold DECIMAL(10, 4) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Benchmark Results
CREATE TABLE IF NOT EXISTS benchmark_results (
    id SERIAL PRIMARY KEY,
    benchmark_id VARCHAR(255) NOT NULL REFERENCES benchmarks(id),
    value DECIMAL(10, 4) NOT NULL,
    passed BOOLEAN NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Regressions
CREATE TABLE IF NOT EXISTS regressions (
    id VARCHAR(255) PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL REFERENCES eval_tests(id),
    previous_score DECIMAL(5, 4) NOT NULL,
    current_score DECIMAL(5, 4) NOT NULL,
    delta DECIMAL(6, 4) NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'open' -- 'open', 'acknowledged', 'resolved'
);

-- Eval Suites
CREATE TABLE IF NOT EXISTS eval_suites (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Eval Suite to Test Mapping
CREATE TABLE IF NOT EXISTS eval_suite_tests (
    suite_id VARCHAR(255) NOT NULL REFERENCES eval_suites(id) ON DELETE CASCADE,
    test_id VARCHAR(255) NOT NULL REFERENCES eval_tests(id) ON DELETE CASCADE,
    PRIMARY KEY (suite_id, test_id)
);

-- Generated Tests (for tracking AI-generated tests)
CREATE TABLE IF NOT EXISTS generated_tests (
    id SERIAL PRIMARY KEY,
    prompt TEXT NOT NULL,
    count INTEGER NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trend Data (pre-aggregated for performance)
CREATE TABLE IF NOT EXISTS trend_data (
    id SERIAL PRIMARY KEY,
    test_id VARCHAR(255) NOT NULL REFERENCES eval_tests(id),
    bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    avg_score DECIMAL(5, 4) NOT NULL,
    count INTEGER NOT NULL,
    UNIQUE(test_id, bucket)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_eval_tests_category ON eval_tests(category);
CREATE INDEX IF NOT EXISTS idx_eval_results_test_id ON eval_results(test_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_evaluated_at ON eval_results(evaluated_at);
CREATE INDEX IF NOT EXISTS idx_benchmark_results_benchmark_id ON benchmark_results(benchmark_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_results_timestamp ON benchmark_results(timestamp);
CREATE INDEX IF NOT EXISTS idx_regressions_test_id ON regressions(test_id);
CREATE INDEX IF NOT EXISTS idx_regressions_status ON regressions(status);
CREATE INDEX IF NOT EXISTS idx_regressions_detected_at ON regressions(detected_at);
CREATE INDEX IF NOT EXISTS idx_trend_data_test_id ON trend_data(test_id);
CREATE INDEX IF NOT EXISTS idx_trend_data_bucket ON trend_data(bucket);
