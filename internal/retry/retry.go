package retry

import (
	"context"
	"time"
)

// Config contains retry configuration
type Config struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Retryable     func(error) bool
}

// DefaultConfig returns default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Retryable:     DefaultRetryable,
	}
}

// DefaultRetryable returns true for retryable errors
func DefaultRetryable(err error) bool {
	if err == nil {
		return false
	}
	return true // Default: retry all errors
}

// RetryableError represents a retryable error
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// Do executes a function with retry logic
func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Check context
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Calculate next delay
			delay = time.Duration(float64(delay) * cfg.BackoffFactor)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if cfg.Retryable != nil && !cfg.Retryable(lastErr) {
			return lastErr
		}
	}

	return lastErr
}

// DoWithResult executes a function with retry logic and returns a result
func DoWithResult(ctx context.Context, cfg Config, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * cfg.BackoffFactor)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		if cfg.Retryable != nil && !cfg.Retryable(lastErr) {
			return nil, lastErr
		}
	}

	return nil, lastErr
}

// Attempt represents a single retry attempt
type Attempt struct {
	Attempt int
	Delay   time.Duration
	Error   error
}

// Strategy provides different retry strategies
type Strategy struct {
	config Config
}

// NewStrategy creates a new retry strategy
func NewStrategy(config Config) *Strategy {
	return &Strategy{config: config}
}

// Execute runs the function with retries
func (s *Strategy) Execute(ctx context.Context, fn func() error) error {
	return Do(ctx, s.config, fn)
}

// ExecuteWithResult runs the function with retries and returns a result
func (s *Strategy) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return DoWithResult(ctx, s.config, fn)
}

// Record attempts
type AttemptRecorder struct {
	attempts []Attempt
}

// NewAttemptRecorder creates a new attempt recorder
func NewAttemptRecorder() *AttemptRecorder {
	return &AttemptRecorder{
		attempts: make([]Attempt, 0),
	}
}

// Record records an attempt
func (r *AttemptRecorder) Record(attempt int, delay time.Duration, err error) {
	r.attempts = append(r.attempts, Attempt{
		Attempt: attempt,
		Delay:   delay,
		Error:   err,
	})
}

// GetAttempts returns all recorded attempts
func (r *AttemptRecorder) GetAttempts() []Attempt {
	return r.attempts
}

// Reset clears recorded attempts
func (r *AttemptRecorder) Reset() {
	r.attempts = r.attempts[:0]
}
