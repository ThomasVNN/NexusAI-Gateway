package cache

import (
	"testing"
	"time"
)

func TestCacheManagerCreation(t *testing.T) {
	m := NewCacheManager(time.Minute)
	if m == nil {
		t.Error("Expected non-nil cache manager")
	}
}

func TestCacheManagerSetGet(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Set("key1", "value1")

	value, found := m.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}

	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}
}

func TestCacheManagerInvalidateAll(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Set("key1", "value1")
	m.Set("key2", "value2")

	m.InvalidateAll()

	if m.Size() != 0 {
		t.Errorf("Expected size 0, got %d", m.Size())
	}
}

func TestCacheManagerInvalidateKeys(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Set("key1", "value1")
	m.Set("key2", "value2")
	m.Set("key3", "value3")

	m.InvalidateKeys([]string{"key1", "key3"})

	if m.Size() != 1 {
		t.Errorf("Expected size 1, got %d", m.Size())
	}

	if _, found := m.Get("key1"); found {
		t.Error("Expected key1 to be deleted")
	}

	if _, found := m.Get("key2"); !found {
		t.Error("Expected key2 to exist")
	}
}

func TestCacheManagerInvalidatePattern(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Set("user:1", "user1")
	m.Set("user:2", "user2")
	m.Set("product:1", "product1")
	m.Set("product:2", "product2")

	m.InvalidatePattern("user:*")

	if m.Size() != 2 {
		t.Errorf("Expected size 2, got %d", m.Size())
	}
}

func TestCacheManagerSubscribe(t *testing.T) {
	m := NewCacheManager(time.Minute)

	ch := m.Subscribe("test-subscriber")
	if ch == nil {
		t.Error("Expected non-nil channel")
	}
}

func TestCacheManagerUnsubscribe(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Subscribe("test-subscriber")
	m.Unsubscribe("test-subscriber")

	// Should not panic
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		key      string
		pattern  string
		expected bool
	}{
		{"user:1", "user:*", true},
		{"user:1", "product:*", false},
		{"any", "*", true},
		{"exact", "exact", true},
		{"different", "exact", false},
	}

	for _, tt := range tests {
		result := matchesPattern(tt.key, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchesPattern(%s, %s) = %v, want %v", tt.key, tt.pattern, result, tt.expected)
		}
	}
}

func TestCacheManagerStats(t *testing.T) {
	m := NewCacheManager(time.Minute)

	m.Set("key1", "value1")
	m.Set("key2", "value2")

	stats := m.Stats()

	if stats.Items != 2 {
		t.Errorf("Expected 2 items, got %d", stats.Items)
	}
}

func TestCacheManagerSubscribeNotification(t *testing.T) {
	m := NewCacheManager(time.Minute)

	ch := m.Subscribe("test")

	go func() {
		m.InvalidateAll()
	}()

	select {
	case req := <-ch:
		if req.Strategy != InvalidateAll {
			t.Errorf("Expected InvalidateAll, got %v", req.Strategy)
		}
	case <-time.After(time.Second):
		t.Error("Expected notification within 1 second")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "b") {
		t.Error("Expected 'b' to be in slice")
	}

	if contains(slice, "d") {
		t.Error("Expected 'd' not to be in slice")
	}
}
