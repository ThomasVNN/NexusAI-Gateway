package queue

import (
	"sync"
	"testing"
	"time"
)

func TestQueueCreation(t *testing.T) {
	q := NewQueue(100)
	if q == nil {
		t.Error("Expected non-nil queue")
	}
}

func TestQueueEnqueueDequeue(t *testing.T) {
	q := NewQueue(100)

	item := &Item{
		ID:       "test1",
		Priority: PriorityHigh,
		Data:     "test data",
	}

	if !q.Enqueue(item) {
		t.Error("Expected enqueue to succeed")
	}

	if q.Size() != 1 {
		t.Errorf("Expected size 1, got %d", q.Size())
	}

	dequeued := q.Dequeue()
	if dequeued == nil {
		t.Error("Expected non-nil dequeued item")
	}
	if dequeued.ID != "test1" {
		t.Errorf("Expected ID 'test1', got '%s'", dequeued.ID)
	}
}

func TestQueuePriorityOrder(t *testing.T) {
	q := NewQueue(100)

	// Enqueue in mixed order
	q.Enqueue(&Item{ID: "low", Priority: PriorityLow, Data: nil})
	q.Enqueue(&Item{ID: "high", Priority: PriorityHigh, Data: nil})
	q.Enqueue(&Item{ID: "medium", Priority: PriorityMedium, Data: nil})

	// Should dequeue high first
	first := q.Dequeue()
	if first.ID != "high" {
		t.Errorf("Expected 'high' first, got '%s'", first.ID)
	}

	// Then medium
	second := q.Dequeue()
	if second.ID != "medium" {
		t.Errorf("Expected 'medium' second, got '%s'", second.ID)
	}

	// Then low
	third := q.Dequeue()
	if third.ID != "low" {
		t.Errorf("Expected 'low' third, got '%s'", third.ID)
	}
}

func TestQueueBackpressure(t *testing.T) {
	q := NewQueue(2)

	// Enqueue two items
	q.Enqueue(&Item{ID: "1", Priority: PriorityLow, Data: nil})
	q.Enqueue(&Item{ID: "2", Priority: PriorityLow, Data: nil})

	// Third should fail due to backpressure
	if q.Enqueue(&Item{ID: "3", Priority: PriorityLow, Data: nil}) {
		t.Error("Expected enqueue to fail due to backpressure")
	}
}

func TestQueuePeek(t *testing.T) {
	q := NewQueue(100)

	// Peek on empty queue
	if q.Peek() != nil {
		t.Error("Expected nil peek on empty queue")
	}

	q.Enqueue(&Item{ID: "test", Priority: PriorityLow, Data: nil})

	// Peek should return item without removing
	peeked := q.Peek()
	if peeked == nil {
		t.Error("Expected non-nil peek")
	}
	if peeked.ID != "test" {
		t.Errorf("Expected 'test', got '%s'", peeked.ID)
	}

	// Size should still be 1
	if q.Size() != 1 {
		t.Errorf("Expected size 1, got %d", q.Size())
	}
}

func TestQueueIsEmpty(t *testing.T) {
	q := NewQueue(100)

	if !q.IsEmpty() {
		t.Error("Expected empty queue")
	}

	q.Enqueue(&Item{ID: "test", Priority: PriorityLow, Data: nil})

	if q.IsEmpty() {
		t.Error("Expected non-empty queue")
	}
}

func TestQueueClose(t *testing.T) {
	q := NewQueue(100)
	q.Close()

	// Enqueue should fail after close
	if q.Enqueue(&Item{ID: "test", Priority: PriorityLow, Data: nil}) {
		t.Error("Expected enqueue to fail after close")
	}
}

func TestQueueStats(t *testing.T) {
	q := NewQueue(10)

	q.Enqueue(&Item{ID: "1", Priority: PriorityHigh, Data: nil})
	q.Enqueue(&Item{ID: "2", Priority: PriorityHigh, Data: nil})
	q.Enqueue(&Item{ID: "3", Priority: PriorityMedium, Data: nil})
	q.Enqueue(&Item{ID: "4", Priority: PriorityLow, Data: nil})

	stats := q.Stats()
	if stats.HighPriority != 2 {
		t.Errorf("Expected 2 high priority, got %d", stats.HighPriority)
	}
	if stats.MediumPriority != 1 {
		t.Errorf("Expected 1 medium priority, got %d", stats.MediumPriority)
	}
	if stats.LowPriority != 1 {
		t.Errorf("Expected 1 low priority, got %d", stats.LowPriority)
	}
	if stats.TotalSize != 4 {
		t.Errorf("Expected total 4, got %d", stats.TotalSize)
	}
	if stats.MaxSize != 10 {
		t.Errorf("Expected max 10, got %d", stats.MaxSize)
	}
}

func TestQueueDrain(t *testing.T) {
	q := NewQueue(100)

	q.Enqueue(&Item{ID: "1", Priority: PriorityLow, Data: nil})
	q.Enqueue(&Item{ID: "2", Priority: PriorityLow, Data: nil})

	items := q.Drain()
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	if !q.IsEmpty() {
		t.Error("Expected empty queue after drain")
	}
}

func TestQueueConcurrent(t *testing.T) {
	q := NewQueue(1000)
	var wg sync.WaitGroup

	// Concurrent enqueue
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			q.Enqueue(&Item{ID: string(rune(id)), Priority: PriorityLow, Data: nil})
		}(i)
	}

	wg.Wait()

	// Concurrent dequeue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Dequeue()
		}()
	}

	wg.Wait()

	// Should have 50 items left
	if q.Size() != 50 {
		t.Errorf("Expected 50 items, got %d", q.Size())
	}
}

func TestBackpressureHandlerCreation(t *testing.T) {
	h := NewBackpressureHandler()
	if h == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestGetPressureLevel(t *testing.T) {
	h := NewBackpressureHandler()

	tests := []struct {
		utilization float64
		expected    string
	}{
		{0.3, "normal"},
		{0.5, "low"},
		{0.6, "low"},
		{0.76, "medium"},
		{0.8, "medium"},
		{0.9, "high"},
		{0.95, "critical"},
		{1.0, "critical"},
	}

	for _, tt := range tests {
		got := h.GetPressureLevel(tt.utilization)
		if got != tt.expected {
			t.Errorf("GetPressureLevel(%f) = %s, want %s", tt.utilization, got, tt.expected)
		}
	}
}

func TestShouldReject(t *testing.T) {
	h := NewBackpressureHandler()

	if h.ShouldReject(0.5) {
		t.Error("Should not reject at 50%")
	}

	if !h.ShouldReject(0.96) {
		t.Error("Should reject at 96%")
	}
}

func TestGetRetryAfter(t *testing.T) {
	h := NewBackpressureHandler()

	if h.GetRetryAfter(0.3) != 0 {
		t.Error("Expected 0 retry-after for normal")
	}

	if h.GetRetryAfter(0.96) != 10*time.Second {
		t.Error("Expected 10s retry-after for critical")
	}
}

func TestBackpressureEnableDisable(t *testing.T) {
	h := NewBackpressureHandler()

	if !h.IsActive() {
		t.Error("Expected active initially")
	}

	h.Disable()
	if h.IsActive() {
		t.Error("Expected inactive after disable")
	}

	h.Enable()
	if !h.IsActive() {
		t.Error("Expected active after enable")
	}
}
