package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected 3 max attempts, got %d", cfg.MaxAttempts)
	}

	if cfg.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected 100ms initial delay, got %v", cfg.InitialDelay)
	}

	if cfg.BackoffFactor != 2.0 {
		t.Errorf("Expected 2.0 backoff factor, got %f", cfg.BackoffFactor)
	}
}

func TestDo_Success(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	err := Do(context.Background(), cfg, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestDo_RetryOnFailure(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	err := Do(context.Background(), cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestDo_ExceedMaxAttempts(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	err := Do(context.Background(), cfg, func() error {
		callCount++
		return errors.New("permanent error")
	})

	if err == nil {
		t.Error("Expected error")
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls (max attempts), got %d", callCount)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InitialDelay = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	callCount := 0

	err := Do(ctx, cfg, func() error {
		callCount++
		return errors.New("error")
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Should be limited by context timeout
	if callCount > 1 {
		t.Errorf("Expected 1 call due to context cancellation, got %d", callCount)
	}
}

func TestDoWithResult_Success(t *testing.T) {
	cfg := DefaultConfig()

	result, err := DoWithResult(context.Background(), cfg, func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got '%v'", result)
	}
}

func TestDoWithResult_RetryAndSucceed(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	result, err := DoWithResult(context.Background(), cfg, func() (interface{}, error) {
		callCount++
		if callCount < 2 {
			return nil, errors.New("retry error")
		}
		return "final result", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "final result" {
		t.Errorf("Expected 'final result', got '%v'", result)
	}
}

func TestRetryableError(t *testing.T) {
	err := &RetryableError{Err: errors.New("test error")}

	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}

	// Just verify Unwrap returns something
	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Error("Expected Unwrap to return non-nil")
	}
}

func TestStrategy(t *testing.T) {
	cfg := DefaultConfig()
	strategy := NewStrategy(cfg)

	callCount := 0

	err := strategy.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestAttemptRecorder(t *testing.T) {
	recorder := NewAttemptRecorder()

	recorder.Record(1, 100*time.Millisecond, errors.New("error 1"))
	recorder.Record(2, 200*time.Millisecond, nil)

	attempts := recorder.GetAttempts()

	if len(attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", len(attempts))
	}

	if attempts[0].Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", attempts[0].Attempt)
	}

	if attempts[0].Error.Error() != "error 1" {
		t.Errorf("Expected 'error 1', got '%s'", attempts[0].Error.Error())
	}
}

func TestAttemptRecorder_Reset(t *testing.T) {
	recorder := NewAttemptRecorder()

	recorder.Record(1, 100*time.Millisecond, nil)
	recorder.Record(2, 200*time.Millisecond, nil)

	recorder.Reset()

	attempts := recorder.GetAttempts()
	if len(attempts) != 0 {
		t.Errorf("Expected 0 attempts after reset, got %d", len(attempts))
	}
}

func TestCustomRetryable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retryable = func(err error) bool {
		return err.Error() == "retryable"
	}

	// Should retry
	err := Do(context.Background(), cfg, func() error {
		return errors.New("retryable")
	})

	if err == nil {
		t.Error("Expected error for retryable")
	}

	// Should not retry (non-retryable)
	cfg2 := DefaultConfig()
	cfg2.Retryable = func(err error) bool {
		return err.Error() == "retryable"
	}

	callCount := 0
	err2 := Do(context.Background(), cfg2, func() error {
		callCount++
		return errors.New("non-retryable")
	})

	if err2 == nil {
		t.Error("Expected error for non-retryable")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry for non-retryable), got %d", callCount)
	}
}
