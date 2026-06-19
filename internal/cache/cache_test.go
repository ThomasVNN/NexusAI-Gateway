package cache

import (
	"sync"
	"testing"
	"time"
)

func TestTTLCacheCreation(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	if cache == nil {
		t.Error("Expected non-nil cache")
	}
}

func TestTTLCacheSetGet(t *testing.T) {
	cache := NewTTLCache(time.Minute)

	cache.Set("key1", "value1")

	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}
}

func TestTTLCacheExpiration(t *testing.T) {
	cache := NewTTLCache(50 * time.Millisecond)

	cache.Set("key1", "value1")

	// Should exist immediately
	_, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestTTLCacheDelete(t *testing.T) {
	cache := NewTTLCache(time.Minute)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestTTLCacheClear(t *testing.T) {
	cache := NewTTLCache(time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be cleared")
	}
	_, found = cache.Get("key2")
	if found {
		t.Error("Expected key2 to be cleared")
	}
}

func TestTTLCacheConcurrency(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Set("key", i)
			cache.Get("key")
		}(i)
	}

	wg.Wait()
}
