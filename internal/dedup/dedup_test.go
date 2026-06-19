package dedup

import (
	"testing"
	"time"
)

func TestDeduplicatorCreation(t *testing.T) {
	d := NewDeduplicator(time.Minute)
	if d == nil {
		t.Error("Expected non-nil deduplicator")
	}
}

func TestDeduplicatorGetOrCreate(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	// First request
	req1, _ := d.GetOrCreate("key1")
	if req1 == nil {
		t.Error("Expected non-nil request")
	}

	// Second request with same key - should return same request
	req2, _ := d.GetOrCreate("key1")
	if req2 == nil {
		t.Error("Expected non-nil request for second get")
	}

	// Both should return the same request
	if req1 != req2 {
		t.Error("Expected same request object")
	}
}

func TestDeduplicatorRefCount(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	// Get same key twice
	req1, _ := d.GetOrCreate("key1")
	d.GetOrCreate("key1")

	if req1.RefCount != 2 {
		t.Errorf("Expected ref count 2, got %d", req1.RefCount)
	}

	// Release one
	d.Release("key1")

	// Get stats to verify ref count
	stats := d.Stats()
	if stats.ActiveRequests != 1 {
		t.Errorf("Expected 1 active request, got %d", stats.ActiveRequests)
	}
}

func TestDeduplicatorComplete(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	req, _ := d.GetOrCreate("key1")

	d.Complete("key1", "result", nil)

	// Wait for completion
	select {
	case <-req.Done:
		// Success
	case <-time.After(time.Second):
		t.Error("Expected request to be completed")
	}

	if req.Result != "result" {
		t.Errorf("Expected 'result', got '%v'", req.Result)
	}
}

func TestDeduplicatorCompleteWithError(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	req, _ := d.GetOrCreate("key1")

	expectedErr := &testError{msg: "test error"}
	d.Complete("key1", nil, expectedErr)

	// Wait for completion
	select {
	case <-req.Done:
		// Success
	case <-time.After(time.Second):
		t.Error("Expected request to be completed")
	}

	if req.Err != expectedErr {
		t.Errorf("Expected error, got '%v'", req.Err)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestDeduplicatorRelease(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	// Get twice, release once
	d.GetOrCreate("key1")
	d.GetOrCreate("key1")
	d.Release("key1")

	// Should still exist (ref count = 1)
	stats := d.Stats()
	if stats.ActiveRequests != 1 {
		t.Errorf("Expected 1 active request, got %d", stats.ActiveRequests)
	}

	// Release again
	d.Release("key1")

	// Should be removed
	stats = d.Stats()
	if stats.ActiveRequests != 0 {
		t.Errorf("Expected 0 active requests, got %d", stats.ActiveRequests)
	}
}

func TestDeduplicatorStats(t *testing.T) {
	d := NewDeduplicator(time.Minute)

	d.GetOrCreate("key1")
	d.GetOrCreate("key2")
	d.GetOrCreate("key3")

	stats := d.Stats()
	if stats.ActiveRequests != 3 {
		t.Errorf("Expected 3 active requests, got %d", stats.ActiveRequests)
	}
}

func TestHashKey(t *testing.T) {
	key1 := HashKey("GET", "/api/test", nil)
	key2 := HashKey("GET", "/api/test", nil)
	key3 := HashKey("POST", "/api/test", nil)

	if key1 != key2 {
		t.Error("Expected same key for same request")
	}

	if key1 == key3 {
		t.Error("Expected different key for different method")
	}

	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}
}

func TestHashKeyWithBody(t *testing.T) {
	body := []byte("test body")
	key1 := HashKey("POST", "/api/test", body)
	key2 := HashKey("POST", "/api/test", body)
	key3 := HashKey("POST", "/api/test", []byte("different body"))

	if key1 != key2 {
		t.Error("Expected same key for same body")
	}

	if key1 == key3 {
		t.Error("Expected different key for different body")
	}
}
