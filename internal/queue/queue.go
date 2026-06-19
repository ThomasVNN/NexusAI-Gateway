package queue

import (
	"sync"
	"sync/atomic"
	"time"
)

// Priority levels for queue items
const (
	PriorityHigh   = 2
	PriorityMedium = 1
	PriorityLow    = 0
)

// Item represents a queue item with priority
type Item struct {
	ID        string
	Priority  int
	Data      interface{}
	CreatedAt time.Time
}

// Queue manages prioritized request queues
type Queue struct {
	mu          sync.RWMutex
	high        []*Item
	medium      []*Item
	low         []*Item
	maxSize     int64
	currentSize int64
	closed      bool
}

// NewQueue creates a new priority queue
func NewQueue(maxSize int64) *Queue {
	return &Queue{
		maxSize: maxSize,
		high:    make([]*Item, 0),
		medium:  make([]*Item, 0),
		low:     make([]*Item, 0),
	}
}

// Enqueue adds an item to the queue
func (q *Queue) Enqueue(item *Item) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return false
	}

	// Check backpressure
	if q.maxSize > 0 && atomic.LoadInt64(&q.currentSize) >= q.maxSize {
		return false
	}

	item.CreatedAt = time.Now()

	switch item.Priority {
	case PriorityHigh:
		q.high = append(q.high, item)
	case PriorityMedium:
		q.medium = append(q.medium, item)
	default:
		q.low = append(q.low, item)
	}

	atomic.AddInt64(&q.currentSize, 1)
	return true
}

// Dequeue removes and returns the highest priority item
func (q *Queue) Dequeue() *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	var item *Item

	switch {
	case len(q.high) > 0:
		item = q.high[0]
		q.high = q.high[1:]
	case len(q.medium) > 0:
		item = q.medium[0]
		q.medium = q.medium[1:]
	case len(q.low) > 0:
		item = q.low[0]
		q.low = q.low[1:]
	}

	if item != nil {
		atomic.AddInt64(&q.currentSize, -1)
	}

	return item
}

// Peek returns the next item without removing it
func (q *Queue) Peek() *Item {
	q.mu.RLock()
	defer q.mu.RUnlock()

	switch {
	case len(q.high) > 0:
		return q.high[0]
	case len(q.medium) > 0:
		return q.medium[0]
	case len(q.low) > 0:
		return q.low[0]
	}

	return nil
}

// Size returns the current queue size
func (q *Queue) Size() int64 {
	return atomic.LoadInt64(&q.currentSize)
}

// IsEmpty returns true if the queue is empty
func (q *Queue) IsEmpty() bool {
	return atomic.LoadInt64(&q.currentSize) == 0
}

// Close marks the queue as closed
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}

// Stats returns queue statistics
func (q *Queue) Stats() QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return QueueStats{
		HighPriority:   len(q.high),
		MediumPriority: len(q.medium),
		LowPriority:    len(q.low),
		TotalSize:      atomic.LoadInt64(&q.currentSize),
		MaxSize:        q.maxSize,
		Utilization:    float64(atomic.LoadInt64(&q.currentSize)) / float64(q.maxSize),
		Closed:         q.closed,
	}
}

// QueueStats holds statistics about the queue
type QueueStats struct {
	HighPriority   int
	MediumPriority int
	LowPriority    int
	TotalSize      int64
	MaxSize        int64
	Utilization    float64
	Closed         bool
}

// Drain removes all items from the queue
func (q *Queue) Drain() []*Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	var items []*Item

	for _, item := range q.high {
		items = append(items, item)
	}
	for _, item := range q.medium {
		items = append(items, item)
	}
	for _, item := range q.low {
		items = append(items, item)
	}

	q.high = make([]*Item, 0)
	q.medium = make([]*Item, 0)
	q.low = make([]*Item, 0)
	atomic.StoreInt64(&q.currentSize, 0)

	return items
}

// BackpressureHandler manages queue backpressure
type BackpressureHandler struct {
	mu              sync.RWMutex
	thresholds      map[string]float64
	active          bool
	currentPressure float64
}

// NewBackpressureHandler creates a new backpressure handler
func NewBackpressureHandler() *BackpressureHandler {
	return &BackpressureHandler{
		thresholds: map[string]float64{
			"low":      0.5,
			"medium":   0.75,
			"high":     0.9,
			"critical": 0.95,
		},
		active: true,
	}
}

// GetPressureLevel returns the current pressure level
func (h *BackpressureHandler) GetPressureLevel(utilization float64) string {
	switch {
	case utilization >= h.thresholds["critical"]:
		return "critical"
	case utilization >= h.thresholds["high"]:
		return "high"
	case utilization >= h.thresholds["medium"]:
		return "medium"
	case utilization >= h.thresholds["low"]:
		return "low"
	default:
		return "normal"
	}
}

// ShouldReject returns true if requests should be rejected
func (h *BackpressureHandler) ShouldReject(utilization float64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.active {
		return false
	}

	return utilization >= h.thresholds["critical"]
}

// GetRetryAfter returns the retry-after duration based on pressure
func (h *BackpressureHandler) GetRetryAfter(utilization float64) time.Duration {
	level := h.GetPressureLevel(utilization)

	switch level {
	case "critical":
		return 10 * time.Second
	case "high":
		return 5 * time.Second
	case "medium":
		return 2 * time.Second
	case "low":
		return 500 * time.Millisecond
	default:
		return 0
	}
}

// SetThresholds updates the pressure thresholds
func (h *BackpressureHandler) SetThresholds(thresholds map[string]float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for k, v := range thresholds {
		h.thresholds[k] = v
	}
}

// Enable enables backpressure handling
func (h *BackpressureHandler) Enable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active = true
}

// Disable disables backpressure handling
func (h *BackpressureHandler) Disable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active = false
}

// IsActive returns whether backpressure is active
func (h *BackpressureHandler) IsActive() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.active
}
