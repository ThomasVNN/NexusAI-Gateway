package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
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

func TestRetry_RetriesOnFailure(t *testing.T) {
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
		if callCount < 3 {
			return nil, errors.New("temporary failure")
		}
		return "success after retry", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}

	if result != "success after retry" {
		t.Errorf("Retry() result = %v, want success after retry", result)
	}

	if callCount != 3 {
		t.Errorf("Retry() called function %d times, want 3", callCount)
	}
}

func TestRetry_GivesUpAfterMaxRetries(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     false,
	}

	ctx := context.Background()
	callCount := 0

	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if callCount != 3 { // 1 initial + 2 retries
		t.Errorf("Retry() called function %d times, want 3", callCount)
	}
}

func TestRetry_RespectsContextCancellation(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 10,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Cancel after first call
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("failure")
	})

	if err != context.Canceled {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}
}

func TestRetry_RespectsContextTimeout(t *testing.T) {
	config := &RetryConfig{
		MaxRetries: 10,
		BaseDelay:  50 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Jitter:     false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	callCount := 0

	_, err := Retry(ctx, config, func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, errors.New("failure")
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Retry() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestCalculateDelay(t *testing.T) {
	tests := []struct {
		name      string
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
		jitter    bool
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "Exponential backoff without jitter",
			attempt:   2,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  5 * time.Second,
			jitter:    false,
			wantMin:   400 * time.Millisecond, // 100ms * 2^2 = 400ms
			wantMax:   400 * time.Millisecond,
		},
		{
			name:      "Exponential backoff with jitter",
			attempt:   2,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  5 * time.Second,
			jitter:    true,
			wantMin:   400 * time.Millisecond,
			wantMax:   500 * time.Millisecond, // 400ms + up to 25% jitter
		},
		{
			name:      "Capped at max delay",
			attempt:   10,
			baseDelay: 100 * time.Millisecond,
			maxDelay:  1 * time.Second,
			jitter:    false,
			wantMin:   1 * time.Second,
			wantMax:   1 * time.Second,
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

			if delay < tt.wantMin {
				t.Errorf("CalculateDelay() = %v, want >= %v", delay, tt.wantMin)
			}
			if delay > tt.wantMax {
				t.Errorf("CalculateDelay() = %v, want <= %v", delay, tt.wantMax)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	config := &RetryConfig{
		RetryableCodes: []int{408, 429, 500, 502, 503, 504},
	}

	tests := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{"Retryable status code 429", 429, nil, true},
		{"Retryable status code 503", 503, nil, true},
		{"Non-retryable status code 404", 404, nil, false},
		{"Error present - retryable", 0, errors.New("network error"), true},
		{"No error and non-retryable status", 404, nil, false},
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

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("DefaultRetryConfig().MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("DefaultRetryConfig().BaseDelay = %v, want 100ms", config.BaseDelay)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("DefaultRetryConfig().MaxDelay = %v, want 5s", config.MaxDelay)
	}

	if !config.Jitter {
		t.Error("DefaultRetryConfig().Jitter should be true")
	}

	expectedCodes := []int{408, 429, 500, 502, 503, 504}
	if len(config.RetryableCodes) != len(expectedCodes) {
		t.Errorf("DefaultRetryConfig().RetryableCodes length = %d, want %d", len(config.RetryableCodes), len(expectedCodes))
	}
}

func TestCircuitBreakerError(t *testing.T) {
	err := ErrCircuitOpen

	if err.Error() != "circuit breaker is open" {
		t.Errorf("CircuitBreakerError.Error() = %v, want 'circuit breaker is open'", err.Error())
	}

	customErr := &CircuitBreakerError{Message: "custom message"}
	if customErr.Error() != "custom message" {
		t.Errorf("Custom CircuitBreakerError.Error() = %v, want 'custom message'", customErr.Error())
	}
}
