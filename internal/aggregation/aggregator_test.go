package aggregation

import (
	"testing"
	"time"
)

func TestAggregatorCreation(t *testing.T) {
	a := NewAggregator(time.Minute)
	if a == nil {
		t.Error("Expected non-nil aggregator")
	}
}

func TestAggregatorRecord(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(20.0)
	a.Record(30.0)

	result := a.Get()

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}

	if result.Sum != 60.0 {
		t.Errorf("Expected sum 60, got %f", result.Sum)
	}

	if result.Avg != 20.0 {
		t.Errorf("Expected avg 20, got %f", result.Avg)
	}
}

func TestAggregatorMinMax(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(50.0)
	a.Record(10.0)
	a.Record(30.0)

	result := a.Get()

	if result.Min != 10.0 {
		t.Errorf("Expected min 10, got %f", result.Min)
	}

	if result.Max != 50.0 {
		t.Errorf("Expected max 50, got %f", result.Max)
	}
}

func TestAggregatorEmpty(t *testing.T) {
	a := NewAggregator(time.Minute)

	result := a.Get()

	if result.Count != 0 {
		t.Errorf("Expected count 0, got %d", result.Count)
	}
}

func TestAggregatorCount(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(20.0)

	if a.Count() != 2 {
		t.Errorf("Expected count 2, got %d", a.Count())
	}
}

func TestAggregatorSum(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(20.0)
	a.Record(30.0)

	if a.Sum() != 60.0 {
		t.Errorf("Expected sum 60, got %f", a.Sum())
	}
}

func TestAggregatorAvg(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(20.0)
	a.Record(30.0)

	if a.Avg() != 20.0 {
		t.Errorf("Expected avg 20, got %f", a.Avg())
	}
}

func TestAggregatorMin(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(30.0)
	a.Record(10.0)
	a.Record(20.0)

	if a.Min() != 10.0 {
		t.Errorf("Expected min 10, got %f", a.Min())
	}
}

func TestAggregatorMax(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(50.0)
	a.Record(30.0)

	if a.Max() != 50.0 {
		t.Errorf("Expected max 50, got %f", a.Max())
	}
}

func TestAggregatorRate(t *testing.T) {
	a := NewAggregator(time.Second)

	a.Record(1.0)
	a.Record(1.0)
	a.Record(1.0)

	result := a.Get()

	// 3 records in 1 second = 3 per second
	if result.Rate < 2.5 || result.Rate > 3.5 {
		t.Errorf("Expected rate ~3, got %f", result.Rate)
	}
}

func TestAggregatorClear(t *testing.T) {
	a := NewAggregator(time.Minute)

	a.Record(10.0)
	a.Record(20.0)

	if a.Count() != 2 {
		t.Errorf("Expected count 2, got %d", a.Count())
	}

	a.Clear()

	if a.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", a.Count())
	}
}

func TestAggregatorPercentile(t *testing.T) {
	a := NewAggregator(time.Minute)

	// Record values 1-10
	for i := 1; i <= 10; i++ {
		a.Record(float64(i))
	}

	p50 := a.Percentile(50)
	if p50 < 4.0 || p50 > 6.0 {
		t.Errorf("Expected p50 ~5, got %f", p50)
	}
}

func TestBucketCreation(t *testing.T) {
	b := &Bucket{
		Count: 10,
		Sum:   100,
		Min:   5,
		Max:   15,
	}

	if b.Count != 10 {
		t.Errorf("Expected count 10, got %d", b.Count)
	}

	if b.Sum != 100 {
		t.Errorf("Expected sum 100, got %f", b.Sum)
	}
}

func TestAggregationResultCreation(t *testing.T) {
	r := &AggregationResult{
		Count: 10,
		Sum:   100,
		Avg:   10,
		Min:   5,
		Max:   15,
		Rate:  5.0,
	}

	if r.Count != 10 {
		t.Errorf("Expected count 10, got %d", r.Count)
	}

	if r.Avg != 10 {
		t.Errorf("Expected avg 10, got %f", r.Avg)
	}

	if r.Rate != 5.0 {
		t.Errorf("Expected rate 5, got %f", r.Rate)
	}
}

func TestAggregatorConcurrentRecord(t *testing.T) {
	a := NewAggregator(time.Minute)

	done := make(chan bool)

	// Concurrent recording
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				a.Record(float64(n * j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 1000 records
	count := a.Count()
	if count != 1000 {
		t.Errorf("Expected 1000 records, got %d", count)
	}
}
