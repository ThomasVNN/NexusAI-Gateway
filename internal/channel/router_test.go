package channel

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/retry"
)

func TestWeightedRandomSelect(t *testing.T) {
	channels := []*Channel{
		{ID: 1, Name: "high-priority", Priority: 10, Ratio: 2},
		{ID: 2, Name: "medium-priority", Priority: 5, Ratio: 1},
		{ID: 3, Name: "low-priority", Priority: 1, Ratio: 1},
	}

	// Run selection multiple times to test randomness
	selectionCounts := map[int64]int{
		1: 0,
		2: 0,
		3: 0,
	}

	iterations := 10000
	for i := 0; i < iterations; i++ {
		selected := weightedRandomSelect(channels)
		selectionCounts[selected.ID]++
	}

	t.Logf("Selection distribution after %d iterations: %v", iterations, selectionCounts)

	// Channel 1 should be selected most often (weight: 10*2 = 20)
	// Channel 2 should be second (weight: 5*1 = 5)
	// Channel 3 should be selected least (weight: 1*1 = 1)

	if selectionCounts[1] <= selectionCounts[2] {
		t.Errorf("Channel 1 should be selected more than Channel 2")
	}
	if selectionCounts[2] <= selectionCounts[3] {
		t.Errorf("Channel 2 should be selected more than Channel 3")
	}
}

func TestRouterCircuitBreaker(t *testing.T) {
	// Use the retry package's CircuitBreaker
	cb := retry.NewCircuitBreaker(5, 2, 30*time.Second)

	// Initial state should be closed
	if state := cb.State(); state != retry.CircuitClosed {
		t.Errorf("Initial state should be closed, got %v", state)
	}

	// Record failures up to threshold
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if state := cb.State(); state != retry.CircuitClosed {
		t.Errorf("State should still be closed before threshold, got %v", state)
	}

	// Record one more failure to trip circuit
	cb.RecordFailure()
	if state := cb.State(); state != retry.CircuitOpen {
		t.Errorf("State should be open after threshold, got %v", state)
	}

	// Record success - in open state, success should not close it
	cb.RecordSuccess()
	if state := cb.State(); state != retry.CircuitOpen {
		t.Errorf("State should still be open (success doesn't close from open), got %v", state)
	}
}

func TestRouterCircuitBreakerReset(t *testing.T) {
	cb := retry.NewCircuitBreaker(3, 2, 100*time.Millisecond)

	// Trip the circuit
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != retry.CircuitOpen {
		t.Fatal("Circuit should be open")
	}

	// Reset should close the circuit
	cb.Reset()
	if cb.State() != retry.CircuitClosed {
		t.Errorf("State should be closed after reset, got %v", cb.State())
	}
}

func TestRouterCircuitBreakerStats(t *testing.T) {
	cb := retry.NewCircuitBreaker(5, 2, 30*time.Second)

	// Record some failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	stats := cb.Stats()
	if stats.State != "closed" {
		t.Errorf("Expected state 'closed', got %s", stats.State)
	}
	if stats.FailureCount != 3 {
		t.Errorf("Expected 3 failures, got %d", stats.FailureCount)
	}
}

func TestChannelModelSupport(t *testing.T) {
	tests := []struct {
		name     string
		models   []string
		test     string
		expected bool
	}{
		{
			name:     "empty models supports all",
			models:   []string{},
			test:     "gpt-4",
			expected: true,
		},
		{
			name:     "wildcard supports all",
			models:   []string{"*"},
			test:     "gpt-4",
			expected: true,
		},
		{
			name:     "specific model match",
			models:   []string{"gpt-4", "gpt-3.5"},
			test:     "gpt-4",
			expected: true,
		},
		{
			name:     "specific model no match",
			models:   []string{"gpt-4", "gpt-3.5"},
			test:     "claude-3",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &Channel{Models: tt.models}
			if result := ch.IsModelSupported(tt.test); result != tt.expected {
				t.Errorf("IsModelSupported(%q) = %v, want %v", tt.test, result, tt.expected)
			}
		})
	}
}

func TestChannelValidation(t *testing.T) {
	tests := []struct {
		name        string
		channel     *Channel
		expectError bool
	}{
		{
			name: "valid channel",
			channel: &Channel{
				Name:            "test-channel",
				Type:            ChannelTypeOpenAI,
				APIKeyEncrypted: "sk-test",
			},
			expectError: false,
		},
		{
			name: "missing name",
			channel: &Channel{
				Type:            ChannelTypeOpenAI,
				APIKeyEncrypted: "sk-test",
			},
			expectError: true,
		},
		{
			name: "missing type",
			channel: &Channel{
				Name:            "test-channel",
				APIKeyEncrypted: "sk-test",
			},
			expectError: true,
		},
		{
			name: "ollama without API key",
			channel: &Channel{
				Name: "test-ollama",
				Type: ChannelTypeOllama,
			},
			expectError: false,
		},
		{
			name: "openai without API key",
			channel: &Channel{
				Name: "test-openai",
				Type: ChannelTypeOpenAI,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.channel.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestGetEffectiveURL(t *testing.T) {
	tests := []struct {
		name     string
		channel  *Channel
		expected string
	}{
		{
			name:     "custom URL",
			channel:  &Channel{Type: ChannelTypeOpenAI, BaseURL: "https://custom.openai.com/v1"},
			expected: "https://custom.openai.com/v1",
		},
		{
			name:     "openai default",
			channel:  &Channel{Type: ChannelTypeOpenAI, BaseURL: ""},
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "anthropic default",
			channel:  &Channel{Type: ChannelTypeAnthropic, BaseURL: ""},
			expected: "https://api.anthropic.com/v1",
		},
		{
			name:     "google default",
			channel:  &Channel{Type: ChannelTypeGoogle, BaseURL: ""},
			expected: "https://generativelanguage.googleapis.com/v1",
		},
		{
			name:     "ollama default",
			channel:  &Channel{Type: ChannelTypeOllama, BaseURL: ""},
			expected: "http://localhost:11434/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.channel.GetEffectiveURL(); result != tt.expected {
				t.Errorf("GetEffectiveURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Mock HTTP server for testing channel connectivity
func setupMockServer(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}
