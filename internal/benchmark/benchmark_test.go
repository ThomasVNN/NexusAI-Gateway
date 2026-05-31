package benchmark

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	t.Run("Basic benchmark", func(t *testing.T) {
		fn := func(ctx context.Context) (interface{}, error) {
			time.Sleep(1 * time.Millisecond)
			return "result", nil
		}

		result := Run(ctx, "test", 100, fn)

		if result.Name != "test" {
			t.Errorf("Expected name 'test', got '%s'", result.Name)
		}
		if result.Iterations != 100 {
			t.Errorf("Expected 100 iterations, got %d", result.Iterations)
		}
		if result.SuccessCount != 100 {
			t.Errorf("Expected 100 successes, got %d", result.SuccessCount)
		}
		if result.ErrorCount != 0 {
			t.Errorf("Expected 0 errors, got %d", result.ErrorCount)
		}
	})

	t.Run("Benchmark with errors", func(t *testing.T) {
		callCount := 0
		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			if callCount%10 == 0 {
				return nil, context.DeadlineExceeded
			}
			return "ok", nil
		}

		result := Run(ctx, "error-test", 100, fn)

		if result.SuccessCount != 90 {
			t.Errorf("Expected 90 successes, got %d", result.SuccessCount)
		}
		if result.ErrorCount != 10 {
			t.Errorf("Expected 10 errors, got %d", result.ErrorCount)
		}
	})

	t.Run("Empty benchmark", func(t *testing.T) {
		fn := func(ctx context.Context) (interface{}, error) {
			return nil, nil
		}

		result := Run(ctx, "empty", 0, fn)

		if result.Name != "empty" {
			t.Errorf("Expected name 'empty', got '%s'", result.Name)
		}
	})
}

func TestRunConcurrent(t *testing.T) {
	ctx := context.Background()

	t.Run("Concurrent benchmark", func(t *testing.T) {
		fn := func(ctx context.Context) (interface{}, error) {
			time.Sleep(1 * time.Millisecond)
			return "concurrent-result", nil
		}

		result := RunConcurrent(ctx, "concurrent-test", 100, 4, fn)

		if result.Name != "concurrent-test" {
			t.Errorf("Expected name 'concurrent-test', got '%s'", result.Name)
		}
		if result.Iterations != 100 {
			t.Errorf("Expected 100 iterations, got %d", result.Iterations)
		}
		if result.SuccessCount != 100 {
			t.Errorf("Expected 100 successes, got %d", result.SuccessCount)
		}
	})

	t.Run("More workers than iterations", func(t *testing.T) {
		fn := func(ctx context.Context) (interface{}, error) {
			return "test", nil
		}

		result := RunConcurrent(ctx, "many-workers", 10, 20, fn)

		if result.SuccessCount != 10 {
			t.Errorf("Expected 10 successes, got %d", result.SuccessCount)
		}
	})
}

func TestBenchmarkMetrics(t *testing.T) {
	t.Run("Record measurements", func(t *testing.T) {
		m := NewBenchmarkMetrics("test", 10)

		m.Record(10*time.Millisecond, true, 100)
		m.Record(20*time.Millisecond, true, 200)
		m.Record(30*time.Millisecond, false, 0)

		if m.successes != 2 {
			t.Errorf("Expected 2 successes, got %d", m.successes)
		}
		if m.errors != 1 {
			t.Errorf("Expected 1 error, got %d", m.errors)
		}
	})

	t.Run("GetResult empty", func(t *testing.T) {
		m := NewBenchmarkMetrics("empty", 10)
		result := m.GetResult()

		if result.Name != "empty" {
			t.Errorf("Expected name 'empty', got '%s'", result.Name)
		}
	})
}

func TestCompareResults(t *testing.T) {
	t.Run("Compare different results", func(t *testing.T) {
		a := &BenchmarkResult{
			Name:         "A",
			MeanLatency:  100 * time.Millisecond,
			OpsPerSecond: 100,
		}
		b := &BenchmarkResult{
			Name:         "B",
			MeanLatency:  50 * time.Millisecond,
			OpsPerSecond: 200,
		}

		comparison := CompareResults(a, b)

		metrics := comparison["metric"].(map[string]map[string]float64)
		if metrics["ops_per_second"]["diff"] <= 0 {
			t.Error("B should have higher ops/s than A")
		}
	})

	t.Run("Equal results", func(t *testing.T) {
		a := &BenchmarkResult{
			Name:        "A",
			MeanLatency: 100 * time.Millisecond,
		}
		b := &BenchmarkResult{
			Name:        "B",
			MeanLatency: 100 * time.Millisecond,
		}

		comparison := CompareResults(a, b)
		winner := comparison["winner"].(map[string]bool)

		if !winner["equal"] {
			t.Error("Results should be equal")
		}
	})
}

func TestPerformanceThresholds(t *testing.T) {
	t.Run("Within thresholds", func(t *testing.T) {
		thresholds := DefaultThresholds()
		result := &BenchmarkResult{
			MeanLatency:  100 * time.Millisecond,
			OpsPerSecond: 50,
			SuccessCount: 99,
			ErrorCount:   1,
		}

		if !thresholds.IsAcceptable(result) {
			t.Error("Result should be acceptable")
		}
	})

	t.Run("Exceeds latency threshold", func(t *testing.T) {
		thresholds := DefaultThresholds()
		result := &BenchmarkResult{
			MeanLatency:  1000 * time.Millisecond,
			OpsPerSecond: 50,
		}

		if thresholds.IsAcceptable(result) {
			t.Error("Result should not be acceptable")
		}
	})

	t.Run("Below ops threshold", func(t *testing.T) {
		thresholds := DefaultThresholds()
		result := &BenchmarkResult{
			MeanLatency:  100 * time.Millisecond,
			OpsPerSecond: 5,
		}

		if thresholds.IsAcceptable(result) {
			t.Error("Result should not be acceptable")
		}
	})

	t.Run("Too many errors", func(t *testing.T) {
		thresholds := DefaultThresholds()
		result := &BenchmarkResult{
			MeanLatency:  100 * time.Millisecond,
			OpsPerSecond: 50,
			SuccessCount: 50,
			ErrorCount:   50,
		}

		if thresholds.IsAcceptable(result) {
			t.Error("Result should not be acceptable")
		}
	})
}

func TestQuickSort(t *testing.T) {
	t.Run("Sort latencies", func(t *testing.T) {
		arr := []time.Duration{
			100 * time.Millisecond,
			50 * time.Millisecond,
			200 * time.Millisecond,
			75 * time.Millisecond,
		}

		quickSort(arr)

		if arr[0] != 50*time.Millisecond {
			t.Errorf("Expected 50ms first, got %v", arr[0])
		}
		if arr[3] != 200*time.Millisecond {
			t.Errorf("Expected 200ms last, got %v", arr[3])
		}
	})

	t.Run("Empty slice", func(t *testing.T) {
		arr := []time.Duration{}
		quickSort(arr) // Should not panic
	})

	t.Run("Single element", func(t *testing.T) {
		arr := []time.Duration{100 * time.Millisecond}
		quickSort(arr)
		if arr[0] != 100*time.Millisecond {
			t.Error("Single element should remain the same")
		}
	})
}

func BenchmarkRun(b *testing.B) {
	ctx := context.Background()
	fn := func(ctx context.Context) (interface{}, error) {
		time.Sleep(1 * time.Millisecond)
		return "benchmark", nil
	}

	b.ResetTimer()
	Run(ctx, "benchmark", b.N, fn)
}

func BenchmarkRunConcurrent(b *testing.B) {
	ctx := context.Background()
	fn := func(ctx context.Context) (interface{}, error) {
		time.Sleep(1 * time.Millisecond)
		return "benchmark", nil
	}

	b.ResetTimer()
	RunConcurrent(ctx, "concurrent-benchmark", b.N, 4, fn)
}
