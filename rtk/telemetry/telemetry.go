package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TelemetryConfig contains telemetry configuration
type TelemetryConfig struct {
	Enabled     bool
	Endpoint    string
	BatchSize   int
	FlushPeriod time.Duration
	UserID      string // Anonymized
	WorkspaceID string // Anonymized
}

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "command", "optimization", "error", "feature"
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// AnonymousID generates an anonymous ID using SHA-256 hash
func AnonymousID(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:16]
}

// TelemetryService provides telemetry operations
type TelemetryService struct {
	config      TelemetryConfig
	events      []TelemetryEvent
	queue       chan TelemetryEvent
	batchSize   int
	flushPeriod time.Duration
	mu          sync.Mutex
	enabled     bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewTelemetryService creates a new telemetry service
func NewTelemetryService(config TelemetryConfig) *TelemetryService {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushPeriod <= 0 {
		config.FlushPeriod = 30 * time.Second
	}

	return &TelemetryService{
		config:      config,
		events:      make([]TelemetryEvent, 0, config.BatchSize),
		queue:       make(chan TelemetryEvent, 1000),
		batchSize:   config.BatchSize,
		flushPeriod:  config.FlushPeriod,
		enabled:      config.Enabled,
		stopCh:       make(chan struct{}),
	}
}

// Track records a telemetry event
func (t *TelemetryService) Track(ctx context.Context, event TelemetryEvent) {
	if !t.enabled {
		return
	}

	event.ID = fmt.Sprintf("%s-%d", event.Type, time.Now().UnixNano())
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	select {
	case t.queue <- event:
	default:
		// Queue full, drop event
	}
}

// TrackCommand tracks a command event
func (t *TelemetryService) TrackCommand(ctx context.Context, command string, commandType string, savings int) {
	event := TelemetryEvent{
		Type: "command",
		Name: command,
		Properties: map[string]interface{}{
			"command_type": commandType,
			"savings":      savings,
		},
	}
	t.Track(ctx, event)
}

// TrackOptimization tracks an optimization event
func (t *TelemetryService) TrackOptimization(ctx context.Context, original, optimized string, savings int) {
	event := TelemetryEvent{
		Type: "optimization",
		Name: "code_optimization",
		Properties: map[string]interface{}{
			"original_length": len(original),
			"optimized_length": len(optimized),
			"savings_bytes":    savings,
			"savings_percent":  calculateSavingsPercent(original, optimized),
		},
	}
	t.Track(ctx, event)
}

// TrackError tracks an error event
func (t *TelemetryService) TrackError(ctx context.Context, errorType string, message string) {
	event := TelemetryEvent{
		Type: "error",
		Name: errorType,
		Properties: map[string]interface{}{
			"message": message,
		},
	}
	t.Track(ctx, event)
}

// TrackFeature tracks a feature usage event
func (t *TelemetryService) TrackFeature(ctx context.Context, feature string, properties map[string]interface{}) {
	event := TelemetryEvent{
		Type:      "feature",
		Name:      feature,
		Properties: properties,
	}
	t.Track(ctx, event)
}

// Start starts the telemetry service
func (t *TelemetryService) Start(ctx context.Context) {
	if !t.enabled {
		return
	}

	t.wg.Add(1)
	go t.processEvents(ctx)
}

// processEvents processes events from the queue
func (t *TelemetryService) processEvents(ctx context.Context) {
	defer t.wg.Done()

	ticker := time.NewTicker(t.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.flush()
			return
		case <-t.stopCh:
			t.flush()
			return
		case event := <-t.queue:
			t.mu.Lock()
			t.events = append(t.events, event)
			shouldFlush := len(t.events) >= t.batchSize
			t.mu.Unlock()

			if shouldFlush {
				t.flush()
			}
		case <-ticker.C:
			t.flush()
		}
	}
}

// flush flushes all pending events
func (t *TelemetryService) flush() {
	t.mu.Lock()
	if len(t.events) == 0 {
		t.mu.Unlock()
		return
	}

	events := make([]TelemetryEvent, len(t.events))
	copy(events, t.events)
	t.events = t.events[:0]
	t.mu.Unlock()

	// Send to endpoint
	t.sendEvents(events)
}

// sendEvents sends events to the telemetry endpoint
func (t *TelemetryService) sendEvents(events []TelemetryEvent) {
	if t.config.Endpoint == "" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"user_id":       t.config.UserID,
		"workspace_id":  t.config.WorkspaceID,
		"events":        events,
		"sent_at":       time.Now().UTC(),
	})
	if err != nil {
		return
	}

	// In production, this would use HTTP client to POST to endpoint
	_ = payload
}

// Stop stops the telemetry service and flushes remaining events
func (t *TelemetryService) Stop(ctx context.Context) {
	close(t.stopCh)
	t.wg.Wait()
	t.flush()
}

// Flush flushes all pending events
func (t *TelemetryService) Flush(ctx context.Context) error {
	t.flush()
	return nil
}

// ConsentManager manages user consent for telemetry
type ConsentManager struct {
	mu        sync.RWMutex
	consented bool
	version   string
	updatedAt time.Time
}

// NewConsentManager creates a new consent manager
func NewConsentManager() *ConsentManager {
	return &ConsentManager{
		version: "1.0",
	}
}

// SetConsent sets user consent
func (c *ConsentManager) SetConsent(consented bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consented = consented
	c.updatedAt = time.Now().UTC()
	return nil
}

// IsConsented returns whether user has consented
func (c *ConsentManager) IsConsented() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consented
}

// GDPRCompliance provides GDPR compliance helpers
type GDPRCompliance struct{}

// Anonymize personally identifiable information
func (g *GDPRCompliance) Anonymize(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range data {
		switch val := v.(type) {
		case string:
			if isPII(k) {
				result[k] = AnonymousID(val)
			} else {
				result[k] = val
			}
		case map[string]interface{}:
			result[k] = g.Anonymize(val)
		default:
			result[k] = v
		}
	}
	return result
}

// RemovePII removes PII from data
func (g *GDPRCompliance) RemovePII(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range data {
		if isPII(k) {
			continue
		}
		if nested, ok := v.(map[string]interface{}); ok {
			result[k] = g.RemovePII(nested)
		} else {
			result[k] = v
		}
	}
	return result
}

// RightToErasure generates erasure request
func (g *GDPRCompliance) RightToErasure(userID string) (*ErasureRequest, error) {
	return &ErasureRequest{
		UserID:      userID,
		RequestedAt: time.Now().UTC(),
		Status:      "pending",
	}, nil
}

// ErasureRequest represents a GDPR right to erasure request
type ErasureRequest struct {
	UserID      string    `json:"user_id"`
	RequestedAt time.Time `json:"requested_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Status      string    `json:"status"` // "pending", "processing", "completed"
}

// isPII checks if a key might contain PII
func isPII(key string) bool {
	piiFields := map[string]bool{
		"email":      true,
		"name":       true,
		"full_name":  true,
		"first_name": true,
		"last_name":  true,
		"phone":      true,
		"address":    true,
		"ip":         true,
		"ip_address": true,
		"ssn":        true,
		"credit_card": true,
	}
	return piiFields[key]
}

// calculateSavingsPercent calculates the percentage savings
func calculateSavingsPercent(original, optimized string) float64 {
	if len(original) == 0 {
		return 0
	}
	return float64(len(original)-len(optimized)) / float64(len(original)) * 100
}
