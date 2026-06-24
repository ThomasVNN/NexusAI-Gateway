package webhooks

import (
	"log/slog"
	"time"
)

// Config holds the configuration for the webhook receiver
type Config struct {
	// MaxRetries is the maximum number of delivery attempts
	MaxRetries int
	// BaseDelay is the base delay for exponential backoff
	BaseDelay time.Duration
	// Timeout is the HTTP request timeout for webhook delivery
	Timeout time.Duration
	// BufferSize is the size of the delivery queue
	BufferSize int
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxRetries: 5,
		BaseDelay:  time.Second,
		Timeout:    10 * time.Second,
		BufferSize: 1000,
	}
}

// Validate checks if the config is valid
func (c Config) Validate() error {
	if c.MaxRetries < 0 {
		slog.Warn("webhook config: MaxRetries cannot be negative, using 0")
		c.MaxRetries = 0
	}
	if c.BaseDelay <= 0 {
		slog.Warn("webhook config: BaseDelay must be positive, using 1s")
		c.BaseDelay = time.Second
	}
	if c.Timeout <= 0 {
		slog.Warn("webhook config: Timeout must be positive, using 10s")
		c.Timeout = 10 * time.Second
	}
	if c.BufferSize <= 0 {
		slog.Warn("webhook config: BufferSize must be positive, using 1000")
		c.BufferSize = 1000
	}
	return nil
}
