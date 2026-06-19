package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Request represents a deduplicated request
type Request struct {
	Key       string
	Timestamp time.Time
	Result    interface{}
	Err       error
	Done      chan struct{}
	RefCount  int
}

// Deduplicator prevents duplicate requests from hitting backend services
type Deduplicator struct {
	mu       sync.RWMutex
	requests map[string]*Request
	ttl      time.Duration
}

// NewDeduplicator creates a new request deduplicator
func NewDeduplicator(ttl time.Duration) *Deduplicator {
	d := &Deduplicator{
		requests: make(map[string]*Request),
		ttl:      ttl,
	}

	// Start cleanup goroutine
	go d.cleanup()

	return d
}

// GetOrCreate returns an existing request or creates a new one
func (d *Deduplicator) GetOrCreate(key string) (*Request, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if req, exists := d.requests[key]; exists {
		req.RefCount++
		return req, true
	}

	req := &Request{
		Key:       key,
		Timestamp: time.Now(),
		Done:      make(chan struct{}),
		RefCount:  1,
	}
	d.requests[key] = req

	return req, false
}

// Complete marks a request as complete with result or error
func (d *Deduplicator) Complete(key string, result interface{}, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if req, exists := d.requests[key]; exists {
		req.Result = result
		req.Err = err
		close(req.Done)
	}
}

// Release decrements reference count and removes if zero
func (d *Deduplicator) Release(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if req, exists := d.requests[key]; exists {
		req.RefCount--
		if req.RefCount <= 0 {
			delete(d.requests, key)
		}
	}
}

// cleanup periodically removes expired requests
func (d *Deduplicator) cleanup() {
	ticker := time.NewTicker(d.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for key, req := range d.requests {
			if now.Sub(req.Timestamp) > d.ttl {
				delete(d.requests, key)
			}
		}
		d.mu.Unlock()
	}
}

// Stats returns current deduplicator statistics
func (d *Deduplicator) Stats() Stats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return Stats{
		ActiveRequests: len(d.requests),
	}
}

// Stats holds deduplicator statistics
type Stats struct {
	ActiveRequests int
}

// HashKey generates a hash key from request components
func HashKey(method, path string, body []byte) string {
	hasher := sha256.New()
	hasher.Write([]byte(method))
	hasher.Write([]byte(path))
	if len(body) > 0 {
		hasher.Write(body)
	}
	return hex.EncodeToString(hasher.Sum(nil))[:32]
}
