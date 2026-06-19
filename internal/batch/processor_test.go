package batch

import (
	"context"
	"testing"
	"time"
)

func TestBatchProcessor_Submit(t *testing.T) {
	processor := NewBatchProcessor(BatchConfig{
		MaxBatchSize: 10,
		MaxWaitTime:  100 * time.Millisecond,
	})

	item, err := processor.Submit(context.Background(), "test-1", "request")
	if err != nil {
		t.Errorf("Failed to submit: %v", err)
	}

	if item.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", item.ID)
	}

	processor.Close()
}

func TestBatchProcessor_Stats(t *testing.T) {
	processor := NewBatchProcessor(BatchConfig{
		MaxBatchSize: 50,
		MaxWaitTime:  200 * time.Millisecond,
	})

	stats := processor.Stats()

	if stats.MaxBatchSize != 50 {
		t.Errorf("Expected max batch size 50, got %d", stats.MaxBatchSize)
	}

	processor.Close()
}

func TestBatchProcessor_Close(t *testing.T) {
	processor := NewBatchProcessor(BatchConfig{
		MaxBatchSize: 10,
		MaxWaitTime:  100 * time.Millisecond,
	})

	// Should not panic
	processor.Close()
}

func TestBatchItem_Result(t *testing.T) {
	item := &BatchItem{
		ID:      "test-1",
		Request: "test-request",
		Result:  "test-result",
		Done:    true,
	}

	if item.Result != "test-result" {
		t.Errorf("Expected result 'test-result', got '%v'", item.Result)
	}

	if !item.Done {
		t.Error("Expected item to be done")
	}
}

func TestBatchItem_Error(t *testing.T) {
	item := &BatchItem{
		ID:      "test-1",
		Request: "test-request",
		Error:   nil,
		Done:    false,
	}

	if item.Error != nil {
		t.Errorf("Expected nil error, got %v", item.Error)
	}
}

func TestBatchProcessor_Wait(t *testing.T) {
	processor := NewBatchProcessor(BatchConfig{
		MaxBatchSize: 10,
		MaxWaitTime:  50 * time.Millisecond,
		Processor: func(items []interface{}) ([]interface{}, []error) {
			results := make([]interface{}, len(items))
			for i := range items {
				results[i] = "processed"
			}
			return results, nil
		},
	})

	item, _ := processor.Submit(context.Background(), "test-1", "request")

	// Wait with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	result, err := processor.Wait(ctx, item)
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}

	if result != "processed" {
		t.Errorf("Expected 'processed', got '%v'", result)
	}

	processor.Close()
}
