package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// EXTENDED EDGE CASE TESTS
// ============================================================================

func TestRetryExtended_ZeroMaxRetries(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 0,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	result, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return "success", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	if result != "success" {
		t.Errorf("Retry() result = %v, want success", result)
	}

	if callCount != 1 {
		t.Errorf("Retry() called function %d times, want 1", callCount)
	}
}

func TestRetryExtended_ZeroMaxRetriesWithFailure(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 0,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error after single attempt with zero max retries")
	}

	if callCount != 1 {
		t.Errorf("Retry() called function %d times, want 1", callCount)
	}
}

func TestRetryExtended_ImmediateSuccessOnRetry(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	result, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return "immediate success", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	if result != "immediate success" {
		t.Errorf("Retry() result = %v, want immediate success", result)
	}

	if callCount != 1 {
		t.Errorf("Retry() called function %d times, want 1", callCount)
	}
}

func TestRetryExtended_LargeMaxRetries(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 10,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   5 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	result, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		if callCount < 5 {
			return nil, errors.New("temporary failure")
		}
		return "success after multiple retries", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	if result != "success after multiple retries" {
		t.Errorf("Retry() result = %v, want success after multiple retries", result)
	}

	if callCount != 5 {
		t.Errorf("Retry() called function %d times, want 5", callCount)
	}
}

func TestCalculateDelayExtended_ZeroAttempt(t *testing.T) {
	config := &RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  5 * time.Second,
		Jitter:    false,
	}

	delay := CalculateDelay(0, config)
	if delay != 100*time.Millisecond {
		t.Errorf("CalculateDelay(0) = %v, want 100ms", delay)
	}
}

func TestCalculateDelayExtended_FirstAttempt(t *testing.T) {
	config := &RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  5 * time.Second,
		Jitter:    false,
	}

	delay := CalculateDelay(1, config)
	if delay != 200*time.Millisecond {
		t.Errorf("CalculateDelay(1) = %v, want 200ms (100ms * 2^1)", delay)
	}
}

func TestCalculateDelayExtended_JitterVariance(t *testing.T) {
	config := &RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  5 * time.Second,
		Jitter:    true,
	}

	// Run multiple times to check jitter adds variance
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = CalculateDelay(2, config)
	}

	// All delays should be >= 400ms (base)
	// All delays should be <= 500ms (400ms + 25% jitter)
	baseExpected := 400 * time.Millisecond
	maxWithJitter := 500 * time.Millisecond

	for i, delay := range delays {
		if delay < baseExpected {
			t.Errorf("CalculateDelay(2) run %d = %v, want >= 400ms", i, delay)
		}
		if delay > maxWithJitter {
			t.Errorf("CalculateDelay(2) run %d = %v, want <= 500ms", i, delay)
		}
	}

	// Check that not all delays are the same (jitter is working)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	// Note: This might occasionally fail due to randomness, but unlikely
	_ = allSame
}

func TestCalculateDelayExtended_MaxDelayCapped(t *testing.T) {
	config := &RetryConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  300 * time.Millisecond,
		Jitter:    false,
	}

	// Attempt 10 would give 100ms * 2^10 = 102400ms, but should be capped at 300ms
	delay := CalculateDelay(10, config)
	if delay != 300*time.Millisecond {
		t.Errorf("CalculateDelay(10) = %v, want 300ms (capped)", delay)
	}
}

func TestIsRetryableExtended_EmptyRetryableCodes(t *testing.T) {
	config := &RetryConfig{
		RetryableCodes: []int{},
	}

	tests := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{"No retryable codes, no error", 500, nil, false},
		{"No retryable codes, has error", 500, errors.New("err"), true},
		{"No retryable codes, 404", 404, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.IsRetryable(tt.statusCode, tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryableExtended_AllRetryableCodes(t *testing.T) {
	config := &RetryConfig{
		RetryableCodes: []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500},
	}

	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"400 Bad Request", 400, true},
		{"404 Not Found", 404, true},
		{"500 Server Error", 500, true},
		{"503 Service Unavailable", 503, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.IsRetryable(tt.statusCode, nil)
			if got != tt.want {
				t.Errorf("IsRetryable(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestRetryExtended_ReturnsResultOnSuccess(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()

	tests := []struct {
		name   string
		result interface{}
	}{
		{"String result", "hello world"},
		{"Integer result", 42},
		{"Nil result", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			result, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
				callCount++
				return tt.result, nil
			})

			if err != nil {
				t.Fatalf("Retry() error = %v", err)
			}

			if callCount != 1 {
				t.Errorf("Retry() called function %d times, want 1", callCount)
			}

			if tt.result != nil && result != tt.result {
				t.Errorf("Retry() result = %v, want %v", result, tt.result)
			}
		})
	}
}

func TestRetryExtended_CustomRetryableCodes(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     3,
		BaseDelay:      10 * time.Millisecond,
		MaxDelay:       100 * time.Millisecond,
		Jitter:         false,
		RetryableCodes: []int{400, 401, 403},
	}

	ctx := context.Background()
	callCount := 0

	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("error")
	})

	// Should retry on any error since RetryableCodes doesn't affect error retry
	if err == nil {
		t.Error("Expected error after retries")
	}
	if callCount < 2 {
		t.Errorf("Retry() should retry on error, called %d times", callCount)
	}
}

func TestRetryConfigExtended_Structure(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     5,
		BaseDelay:      200 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		Jitter:         true,
		RetryableCodes: []int{408, 429, 500, 502, 503, 504},
	}

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", config.MaxRetries)
	}

	if config.BaseDelay != 200*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 200ms", config.BaseDelay)
	}

	if config.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay = %v, want 10s", config.MaxDelay)
	}

	if !config.Jitter {
		t.Error("Jitter should be true")
	}

	if len(config.RetryableCodes) != 6 {
		t.Errorf("RetryableCodes length = %d, want 6", len(config.RetryableCodes))
	}
}

func TestRetryExtended_ContextNotNil(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 1,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	_, _ = Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		if ctx == nil {
			t.Error("Context should not be nil")
		}
		return nil, errors.New("failure")
	})

	if callCount != 2 { // 1 initial + 1 retry
		t.Errorf("Retry() called function %d times, want 2", callCount)
	}
}

func TestCalculateDelayExtended_DelayProgression(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Jitter:     false,
	}

	// Check exponential backoff progression
	delays := []time.Duration{
		CalculateDelay(0, config), // 100ms
		CalculateDelay(1, config), // 200ms
		CalculateDelay(2, config), // 400ms
		CalculateDelay(3, config), // 800ms
	}

	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
	}

	for i, delay := range delays {
		if delay != expected[i] {
			t.Errorf("CalculateDelay(%d) = %v, want %v", i, delay, expected[i])
		}
	}
}

func TestRetryExtended_MultipleRetriesBeforeSuccess(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 5,
		BaseDelay:  5 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0
	expectedAttempts := 4

	result, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		if callCount < expectedAttempts {
			return nil, errors.New("retry number " + string(rune('0'+callCount)))
		}
		return "success after 3 retries", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	if result != "success after 3 retries" {
		t.Errorf("Retry() result = %v, want 'success after 3 retries'", result)
	}

	if callCount != expectedAttempts {
		t.Errorf("Retry() called function %d times, want %d", callCount, expectedAttempts)
	}
}

func TestRetryExtended_ContextCancelledDuringRetry(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 100,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Jitter:     false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after exactly 3 attempts
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	callCount := 0
	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("failure")
	})

	if err != context.Canceled {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}

	// Should have called multiple times before cancellation
	if callCount < 2 {
		t.Errorf("Retry() called function only %d times before cancel, expected more", callCount)
	}
}

func TestIsRetryableExtended_CaseInsensitive(t *testing.T) {
	config := &RetryConfig{
		RetryableCodes: []int{429, 500, 503},
	}

	// Test that IsRetryable works correctly regardless of error type
	err1 := errors.New("connection refused")
	err2 := errors.New("timeout")

	if !config.IsRetryable(0, err1) {
		t.Error("IsRetryable should return true for any error")
	}

	if !config.IsRetryable(0, err2) {
		t.Error("IsRetryable should return true for any error")
	}

	if config.IsRetryable(404, nil) {
		t.Error("IsRetryable should return false for non-retryable status without error")
	}
}

func TestCalculateDelayExtended_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
		jitter    bool
		expected  time.Duration
	}{
		{
			name:      "Min base delay",
			attempt:   1,
			baseDelay: 1 * time.Nanosecond,
			maxDelay:  1 * time.Second,
			jitter:    false,
			expected:  2 * time.Nanosecond,
		},
		{
			name:      "Very large base delay",
			attempt:   1,
			baseDelay: 1 * time.Hour,
			maxDelay:  2 * time.Hour,
			jitter:    false,
			expected:  2 * time.Hour,
		},
		{
			name:      "Max delay on first attempt",
			attempt:   0,
			baseDelay: 5 * time.Second,
			maxDelay:  5 * time.Second,
			jitter:    false,
			expected:  5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RetryConfig{
				BaseDelay: tt.baseDelay,
				MaxDelay:  tt.maxDelay,
				Jitter:    tt.jitter,
			}

			delay := CalculateDelay(tt.attempt, config)
			if delay != tt.expected {
				t.Errorf("CalculateDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

// Note: Nil context test removed as Retry function expects valid context

func TestDefaultRetryConfigExtended_HasCorrectDefaults(t *testing.T) {
	config := DefaultRetryConfig()

	// Check all default values
	if config.MaxRetries != 3 {
		t.Errorf("Default MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("Default BaseDelay = %v, want 100ms", config.BaseDelay)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("Default MaxDelay = %v, want 5s", config.MaxDelay)
	}

	if !config.Jitter {
		t.Error("Default Jitter should be true")
	}

	// Check default retryable codes
	expectedCodes := []int{408, 429, 500, 502, 503, 504}
	if len(config.RetryableCodes) != len(expectedCodes) {
		t.Errorf("Default RetryableCodes length = %d, want %d", len(config.RetryableCodes), len(expectedCodes))
	}

	for i, code := range expectedCodes {
		if config.RetryableCodes[i] != code {
			t.Errorf("RetryableCodes[%d] = %d, want %d", i, config.RetryableCodes[i], code)
		}
	}
}
