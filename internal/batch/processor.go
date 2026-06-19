package batch

import (
	"context"
	"sync"
	"time"
)

// BatchItem represents a single item in a batch
type BatchItem struct {
	ID      string
	Request interface{}
	Result  interface{}
	Error   error
	Done    bool
}

// BatchProcessor processes items in batches
type BatchProcessor struct {
	mu           sync.RWMutex
	pending      []*BatchItem
	processor    func([]interface{}) ([]interface{}, []error)
	maxBatchSize int
	maxWaitTime  time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// BatchConfig contains configuration for batch processing
type BatchConfig struct {
	MaxBatchSize int
	MaxWaitTime  time.Duration
	Processor    func([]interface{}) ([]interface{}, []error)
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(cfg BatchConfig) *BatchProcessor {
	if cfg.MaxBatchSize == 0 {
		cfg.MaxBatchSize = 100
	}
	if cfg.MaxWaitTime == 0 {
		cfg.MaxWaitTime = 100 * time.Millisecond
	}
	if cfg.Processor == nil {
		cfg.Processor = defaultProcessor
	}

	bp := &BatchProcessor{
		pending:      make([]*BatchItem, 0),
		processor:    cfg.Processor,
		maxBatchSize: cfg.MaxBatchSize,
		maxWaitTime:  cfg.MaxWaitTime,
		stopCh:       make(chan struct{}),
	}

	bp.wg.Add(1)
	go bp.processLoop()

	return bp
}

// Submit submits an item for batch processing
func (bp *BatchProcessor) Submit(ctx context.Context, id string, request interface{}) (*BatchItem, error) {
	item := &BatchItem{
		ID:      id,
		Request: request,
		Done:    false,
	}

	bp.mu.Lock()
	bp.pending = append(bp.pending, item)
	shouldProcess := len(bp.pending) >= bp.maxBatchSize
	bp.mu.Unlock()

	if shouldProcess {
		// Signal processing
		select {
		case bp.stopCh <- struct{}{}:
		default:
		}
	}

	return item, nil
}

// Wait waits for an item to be processed
func (bp *BatchProcessor) Wait(ctx context.Context, item *BatchItem) (interface{}, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			bp.mu.RLock()
			done := item.Done
			result := item.Result
			err := item.Error
			bp.mu.RUnlock()

			if done {
				return result, err
			}
		}
	}
}

// Close stops the batch processor
func (bp *BatchProcessor) Close() {
	close(bp.stopCh)
	bp.wg.Wait()
}

func (bp *BatchProcessor) processLoop() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.maxWaitTime)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopCh:
			bp.processBatch()
			return
		case <-ticker.C:
			bp.processBatch()
		}
	}
}

func (bp *BatchProcessor) processBatch() {
	bp.mu.Lock()
	if len(bp.pending) == 0 {
		bp.mu.Unlock()
		return
	}

	items := bp.pending
	bp.pending = make([]*BatchItem, 0)
	bp.mu.Unlock()

	// Extract requests
	requests := make([]interface{}, len(items))
	for i, item := range items {
		requests[i] = item.Request
	}

	// Process batch
	results, errors := bp.processor(requests)

	// Update items
	for i, item := range items {
		bp.mu.Lock()
		if errors != nil && i < len(errors) && errors[i] != nil {
			item.Error = errors[i]
		}
		if results != nil && i < len(results) {
			item.Result = results[i]
		}
		item.Done = true
		bp.mu.Unlock()
	}
}

func defaultProcessor(items []interface{}) ([]interface{}, []error) {
	results := make([]interface{}, len(items))
	errors := make([]error, len(items))
	return results, errors
}

// BatchStats returns statistics about the batch processor
type BatchStats struct {
	PendingItems int
	MaxBatchSize int
	MaxWaitTime  time.Duration
}

func (bp *BatchProcessor) Stats() BatchStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return BatchStats{
		PendingItems: len(bp.pending),
		MaxBatchSize: bp.maxBatchSize,
		MaxWaitTime:  bp.maxWaitTime,
	}
}
