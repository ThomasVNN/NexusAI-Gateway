package benchmark

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkResult represents the result of a benchmark run
type BenchmarkResult struct {
	Name           string        `json:"name"`
	Iterations     int           `json:"iterations"`
	Duration       time.Duration `json:"duration"`
	OpsPerSecond   float64       `json:"ops_per_second"`
	MeanLatency    time.Duration `json:"mean_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	P50Latency     time.Duration `json:"p50_latency"`
	P90Latency     time.Duration `json:"p90_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	StdDev         time.Duration `json:"std_dev"`
	SuccessCount   int64         `json:"success_count"`
	ErrorCount     int64         `json:"error_count"`
	TotalBytes     int64         `json:"total_bytes"`
	ThroughputMBps float64       `json:"throughput_mbps"`
}

// BenchmarkMetrics holds aggregated benchmark data
type BenchmarkMetrics struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Iterations int
	mu         sync.Mutex
	latencies  []time.Duration
	successes  int64
	errors     int64
	bytes      int64
}

// NewBenchmarkMetrics creates new benchmark metrics tracker
func NewBenchmarkMetrics(name string, iterations int) *BenchmarkMetrics {
	return &BenchmarkMetrics{
		Name:       name,
		Iterations: iterations,
		latencies:  make([]time.Duration, 0, iterations),
	}
}

// Record records a benchmark result
func (m *BenchmarkMetrics) Record(latency time.Duration, success bool, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, latency)
	if success {
		atomic.AddInt64(&m.successes, 1)
	} else {
		atomic.AddInt64(&m.errors, 1)
	}
	atomic.AddInt64(&m.bytes, bytes)
}

// GetResult computes and returns the final benchmark result
func (m *BenchmarkMetrics) GetResult() *BenchmarkResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.latencies) == 0 {
		return &BenchmarkResult{Name: m.Name}
	}

	m.EndTime = time.Now()
	duration := m.EndTime.Sub(m.StartTime)

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	quickSort(sorted)

	// Calculate statistics
	var sum, sumSq int64
	var min, max time.Duration = sorted[0], sorted[len(sorted)-1]
	for _, lat := range sorted {
		sum += int64(lat)
		sumSq += int64(lat) * int64(lat)
	}
	n := float64(len(sorted))
	meanNs := float64(sum) / n

	// Standard deviation
	variance := (float64(sumSq) / n) - (meanNs * meanNs)
	stdDev := time.Duration(math.Sqrt(variance))

	return &BenchmarkResult{
		Name:           m.Name,
		Iterations:     len(sorted),
		Duration:       duration,
		OpsPerSecond:   float64(len(sorted)) / duration.Seconds(),
		MeanLatency:    time.Duration(meanNs),
		MinLatency:     min,
		MaxLatency:     max,
		P50Latency:     sorted[int(n*0.50)],
		P90Latency:     sorted[int(n*0.90)],
		P95Latency:     sorted[int(n*0.95)],
		P99Latency:     sorted[int(n*0.99)],
		StdDev:         stdDev,
		SuccessCount:   atomic.LoadInt64(&m.successes),
		ErrorCount:     atomic.LoadInt64(&m.errors),
		TotalBytes:     atomic.LoadInt64(&m.bytes),
		ThroughputMBps: float64(atomic.LoadInt64(&m.bytes)) / (1024 * 1024) / duration.Seconds(),
	}
}

// Benchmark represents a benchmark function
type Benchmark func(ctx context.Context) (interface{}, error)

// Run executes a benchmark with the given configuration
func Run(ctx context.Context, name string, iterations int, fn Benchmark) *BenchmarkResult {
	m := NewBenchmarkMetrics(name, iterations)
	m.StartTime = time.Now()

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		start := time.Now()
		result, err := fn(ctx)
		latency := time.Since(start)

		var bytes int64
		if result != nil {
			if s, ok := result.(string); ok {
				bytes = int64(len(s))
			}
		}

		m.Record(latency, err == nil, bytes)
	}

	return m.GetResult()
}

// RunConcurrent executes a benchmark with concurrent workers
func RunConcurrent(ctx context.Context, name string, iterations, workers int, fn Benchmark) *BenchmarkResult {
	m := NewBenchmarkMetrics(name, iterations)
	m.StartTime = time.Now()

	type result struct {
		latency time.Duration
		success bool
		bytes   int64
	}

	resultCh := make(chan result, iterations)

	var wg sync.WaitGroup
	perWorker := (iterations + workers - 1) / workers

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * perWorker
			end := start + perWorker
			if end > iterations {
				end = iterations
			}

			for i := start; i < end; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				runStart := time.Now()
				res, err := fn(ctx)
				latency := time.Since(runStart)

				var bytes int64
				if res != nil {
					if s, ok := res.(string); ok {
						bytes = int64(len(s))
					}
				}

				resultCh <- result{latency: latency, success: err == nil, bytes: bytes}
			}
		}(w)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for r := range resultCh {
		m.Record(r.latency, r.success, r.bytes)
	}

	return m.GetResult()
}

// quickSort performs quick sort on latencies
func quickSort(arr []time.Duration) {
	if len(arr) <= 1 {
		return
	}

	pivot := arr[len(arr)/2]
	i, j := 0, len(arr)-1

	for i <= j {
		for arr[i] < pivot {
			i++
		}
		for arr[j] > pivot {
			j--
		}
		if i <= j {
			arr[i], arr[j] = arr[j], arr[i]
			i++
			j--
		}
	}

	if j > 0 {
		quickSort(arr[:j+1])
	}
	if i < len(arr) {
		quickSort(arr[i:])
	}
}

// CompareResults compares two benchmark results
func CompareResults(a, b *BenchmarkResult) map[string]interface{} {
	return map[string]interface{}{
		"metric": map[string]map[string]float64{
			"ops_per_second": {
				"a":    a.OpsPerSecond,
				"b":    b.OpsPerSecond,
				"diff": (b.OpsPerSecond - a.OpsPerSecond) / a.OpsPerSecond * 100,
			},
			"mean_latency_ms": {
				"a":    float64(a.MeanLatency.Milliseconds()),
				"b":    float64(b.MeanLatency.Milliseconds()),
				"diff": (float64(b.MeanLatency.Milliseconds()) - float64(a.MeanLatency.Milliseconds())) / float64(a.MeanLatency.Milliseconds()) * 100,
			},
			"p99_latency_ms": {
				"a":    float64(a.P99Latency.Milliseconds()),
				"b":    float64(b.P99Latency.Milliseconds()),
				"diff": (float64(b.P99Latency.Milliseconds()) - float64(a.P99Latency.Milliseconds())) / float64(a.P99Latency.Milliseconds()) * 100,
			},
		},
		"winner": map[string]bool{
			"a_faster": a.MeanLatency < b.MeanLatency,
			"b_faster": b.MeanLatency < a.MeanLatency,
			"equal":    a.MeanLatency == b.MeanLatency,
		},
	}
}

// PerformanceThresholds defines performance acceptance thresholds
type PerformanceThresholds struct {
	MaxLatency      time.Duration
	MinOpsPerSecond float64
	MaxErrorRate    float64
}

// IsAcceptable checks if a result meets the performance thresholds
func (t *PerformanceThresholds) IsAcceptable(r *BenchmarkResult) bool {
	if r.MeanLatency > t.MaxLatency {
		return false
	}
	if r.OpsPerSecond < t.MinOpsPerSecond {
		return false
	}
	total := r.SuccessCount + r.ErrorCount
	if total > 0 {
		errorRate := float64(r.ErrorCount) / float64(total)
		if errorRate > t.MaxErrorRate {
			return false
		}
	}
	return true
}

// DefaultThresholds returns default performance thresholds
func DefaultThresholds() *PerformanceThresholds {
	return &PerformanceThresholds{
		MaxLatency:      500 * time.Millisecond,
		MinOpsPerSecond: 10,
		MaxErrorRate:    0.01,
	}
}
