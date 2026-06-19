package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StreamEvent represents a single event in a stream
type StreamEvent struct {
	ID        string      `json:"id,omitempty"`
	Type      string      `json:"type,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
}

// StreamHandler handles SSE streaming
type StreamHandler struct {
	client  *http.Client
	timeout time.Duration
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(timeout time.Duration) *StreamHandler {
	return &StreamHandler{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// StreamEventFunc is called for each event
type StreamEventFunc func(event *StreamEvent) error

// Stream performs a streaming request
func (h *StreamHandler) Stream(ctx context.Context, url string, req interface{}, callback StreamEventFunc) error {
	// Create request body
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	_ = body // Body marshaled for future use

	// Execute request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read stream
	reader := resp.Body
	buf := make([]byte, 0, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read more data
		tmp := make([]byte, 1024)
		n, err := reader.Read(tmp)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}

		buf = append(buf, tmp[:n]...)

		// Process complete events
		for {
			delimIdx := -1
			for i := 0; i < len(buf)-1; i++ {
				if i < len(buf)-1 && buf[i] == '\n' && buf[i+1] == '\n' {
					delimIdx = i
					break
				}
			}

			if delimIdx < 0 {
				break
			}

			// Extract event data
			eventData := buf[:delimIdx]
			buf = buf[delimIdx+2:]

			// Parse SSE format
			for _, line := range splitLines(eventData) {
				if len(line) < 7 || string(line[:6]) != "data: " {
					continue
				}

				data := string(line[6:])

				// Parse JSON
				var event StreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}

				event.Timestamp = time.Now()

				// Call callback
				if err := callback(&event); err != nil {
					return fmt.Errorf("callback error: %w", err)
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// StreamConfig contains streaming configuration
type StreamConfig struct {
	URL           string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultStreamConfig returns default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		Timeout:       60 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
}
