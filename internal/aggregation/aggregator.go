package aggregation

import (
	"sync"
	"time"
)

// Aggregator aggregates metrics over time windows
type Aggregator struct {
	mu      sync.RWMutex
	window  time.Duration
	buckets map[int64]*Bucket
	maxAge  time.Duration
}

// Bucket holds aggregated values for a time window
type Bucket struct {
	mu        sync.RWMutex
	Count     int64
	Sum       float64
	Min       float64
	Max       float64
	LastValue float64
	LastTime  time.Time
}

// AggregationResult represents the result of an aggregation
type AggregationResult struct {
	Count     int64
	Sum       float64
	Avg       float64
	Min       float64
	Max       float64
	Rate      float64
	LastValue float64
	LastTime  time.Time
}

// NewAggregator creates a new aggregator with the given window size
func NewAggregator(window time.Duration) *Aggregator {
	return &Aggregator{
		window:  window,
		buckets: make(map[int64]*Bucket),
		maxAge:  window * 2,
	}
}

// Record records a value
func (a *Aggregator) Record(value float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	bucketKey := now.UnixNano() / a.window.Nanoseconds()

	bucket, exists := a.buckets[bucketKey]
	if !exists {
		bucket = &Bucket{
			Min: value,
			Max: value,
		}
		a.buckets[bucketKey] = bucket
	}

	bucket.mu.Lock()
	bucket.Count++
	bucket.Sum += value
	if value < bucket.Min {
		bucket.Min = value
	}
	if value > bucket.Max {
		bucket.Max = value
	}
	bucket.LastValue = value
	bucket.LastTime = now
	bucket.mu.Unlock()

	// Clean up old buckets
	a.cleanup(now)
}

// cleanup removes old buckets
func (a *Aggregator) cleanup(now time.Time) {
	cutoff := now.Add(-a.maxAge).UnixNano() / a.window.Nanoseconds()
	for key := range a.buckets {
		if key < cutoff {
			delete(a.buckets, key)
		}
	}
}

// Get returns the aggregated result for the current window
func (a *Aggregator) Get() *AggregationResult {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var count int64
	var sum float64
	var min float64 = 0
	var max float64 = 0
	var lastValue float64
	var lastTime time.Time
	hasData := false

	now := time.Now()
	cutoff := now.Add(-a.window).UnixNano() / a.window.Nanoseconds()

	for key, bucket := range a.buckets {
		if key < cutoff {
			continue
		}

		bucket.mu.RLock()
		count += bucket.Count
		sum += bucket.Sum
		if !hasData || bucket.Min < min {
			min = bucket.Min
		}
		if !hasData || bucket.Max > max {
			max = bucket.Max
		}
		if bucket.LastTime.After(lastTime) {
			lastValue = bucket.LastValue
			lastTime = bucket.LastTime
		}
		hasData = true
		bucket.mu.RUnlock()
	}

	if !hasData {
		return &AggregationResult{}
	}

	avg := 0.0
	if count > 0 {
		avg = sum / float64(count)
	}

	return &AggregationResult{
		Count:     count,
		Sum:       sum,
		Avg:       avg,
		Min:       min,
		Max:       max,
		Rate:      float64(count) / a.window.Seconds(),
		LastValue: lastValue,
		LastTime:  lastTime,
	}
}

// Percentile calculates a percentile of recorded values
func (a *Aggregator) Percentile(p float64) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var values []float64
	now := time.Now()
	cutoff := now.Add(-a.window).UnixNano() / a.window.Nanoseconds()

	for key, bucket := range a.buckets {
		if key < cutoff {
			continue
		}

		bucket.mu.RLock()
		// Simple approximation: use average of last values
		if bucket.Count > 0 {
			values = append(values, bucket.Sum/float64(bucket.Count))
		}
		bucket.mu.RUnlock()
	}

	if len(values) == 0 {
		return 0
	}

	// Simple percentile approximation
	idx := int(float64(len(values)) * p / 100)
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

// Clear removes all buckets
func (a *Aggregator) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buckets = make(map[int64]*Bucket)
}

// Count returns the number of values recorded
func (a *Aggregator) Count() int64 {
	return a.Get().Count
}

// Sum returns the sum of all values
func (a *Aggregator) Sum() float64 {
	return a.Get().Sum
}

// Avg returns the average of all values
func (a *Aggregator) Avg() float64 {
	return a.Get().Avg
}

// Min returns the minimum value
func (a *Aggregator) Min() float64 {
	return a.Get().Min
}

// Max returns the maximum value
func (a *Aggregator) Max() float64 {
	return a.Get().Max
}

// Rate returns the rate per second
func (a *Aggregator) Rate() float64 {
	return a.Get().Rate
}
