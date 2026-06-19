package streaming

import (
	"testing"
	"time"
)

func TestStreamEventCreation(t *testing.T) {
	event := StreamEvent{
		ID:   "event-1",
		Type: "message",
		Data: "test data",
	}

	if event.ID != "event-1" {
		t.Errorf("Expected ID 'event-1', got '%s'", event.ID)
	}
	if event.Type != "message" {
		t.Errorf("Expected Type 'message', got '%s'", event.Type)
	}
	if event.Data != "test data" {
		t.Errorf("Expected Data 'test data', got '%s'", event.Data)
	}
}

func TestStreamHandlerCreation(t *testing.T) {
	handler := NewStreamHandler(30 * time.Second)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()
	if config.Timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", config.Timeout)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts 3, got %d", config.RetryAttempts)
	}
}

func TestSplitLines(t *testing.T) {
	data := []byte("line1\nline2\nline3")
	lines := splitLines(data)

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestSplitLinesEmpty(t *testing.T) {
	lines := splitLines([]byte(""))
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(lines))
	}
}

func TestSplitLinesNoNewline(t *testing.T) {
	lines := splitLines([]byte("single line"))
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
}

func TestStreamEventWithTimestamp(t *testing.T) {
	event := StreamEvent{
		ID:        "event-2",
		Type:      "update",
		Data:      "update data",
		Timestamp: time.Now(),
	}

	if event.ID != "event-2" {
		t.Errorf("Expected ID 'event-2', got '%s'", event.ID)
	}
	if event.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestStreamEventDataTypes(t *testing.T) {
	// Test with string data
	event1 := StreamEvent{Data: "string data"}
	if event1.Data != "string data" {
		t.Error("Expected string data")
	}

	// Test with map data
	event2 := StreamEvent{Data: map[string]interface{}{"key": "value"}}
	if event2.Data == nil {
		t.Error("Expected non-nil map data")
	}
}
