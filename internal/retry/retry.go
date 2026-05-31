package retry

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	BaseDelay      time.Duration // Initial delay before first retry
	MaxDelay       time.Duration // Maximum delay between retries
	Jitter         bool          // Add random jitter to delays
	RetryableCodes []int         // HTTP status codes to retry
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		Jitter:         true,
		RetryableCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

// IsRetryable checks if an error or status code is retryable
func (c *RetryConfig) IsRetryable(statusCode int, err error) bool {
	for _, code := range c.RetryableCodes {
		if statusCode == code {
			return true
		}
	}
	return err != nil // Retry on any error by default
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) (interface{}, error)

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn RetryableFunc) (interface{}, error) {
	var lastErr error
	var result interface{}

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, lastErr = fn(ctx)

		if lastErr == nil {
			return result, nil
		}

		if attempt == config.MaxRetries {
			break
		}

		delay := CalculateDelay(attempt, config)
		slog.Warn("Retry attempt",
			slog.Int("attempt", attempt+1),
			slog.Int("max_retries", config.MaxRetries),
			slog.Duration("delay", delay),
			slog.Any("error", lastErr),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, lastErr
}

// CalculateDelay calculates the delay for a given attempt
func CalculateDelay(attempt int, config *RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(config.BaseDelay) * math.Pow(2, float64(attempt))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter if enabled
	if config.Jitter {
		// Add up to 25% random jitter
		jitter := delay * 0.25 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

var ErrCircuitOpen = &CircuitBreakerError{Message: "circuit breaker is open"}
