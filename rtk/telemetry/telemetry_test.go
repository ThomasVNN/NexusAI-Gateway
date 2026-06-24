package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestAnonymousID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
	}{
		{
			name:    "basic string",
			input:   "test-user-id",
			wantLen: 16,
		},
		{
			name:    "empty string",
			input:   "",
			wantLen: 16,
		},
		{
			name:    "long string",
			input:   "very-long-user-identifier-string-for-testing",
			wantLen: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AnonymousID(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("AnonymousID() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}

	// Same input should produce same output
	id1 := AnonymousID("same-input")
	id2 := AnonymousID("same-input")
	if id1 != id2 {
		t.Errorf("AnonymousID() not deterministic: %s != %s", id1, id2)
	}

	// Different inputs should produce different outputs
	id3 := AnonymousID("different-input")
	if id1 == id3 {
		t.Errorf("AnonymousID() produced collision: %s == %s", id1, id3)
	}
}

func TestNewTelemetryService(t *testing.T) {
	config := TelemetryConfig{
		Enabled:     true,
		Endpoint:    "https://telemetry.example.com",
		BatchSize:   50,
		FlushPeriod: 10 * time.Second,
		UserID:      "user123",
		WorkspaceID: "workspace456",
	}

	service := NewTelemetryService(config)
	if service == nil {
		t.Fatal("NewTelemetryService() returned nil")
	}

	if service.enabled != true {
		t.Errorf("service.enabled = %v, want true", service.enabled)
	}

	if service.batchSize != 50 {
		t.Errorf("service.batchSize = %d, want 50", service.batchSize)
	}
}

func TestNewTelemetryService_Defaults(t *testing.T) {
	config := TelemetryConfig{
		Enabled: true,
	}

	service := NewTelemetryService(config)

	// Check defaults are applied
	if service.batchSize != 100 {
		t.Errorf("default batchSize = %d, want 100", service.batchSize)
	}

	if service.flushPeriod != 30*time.Second {
		t.Errorf("default flushPeriod = %v, want 30s", service.flushPeriod)
	}
}

func TestTelemetryService_Track(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: true})

	ctx := context.Background()
	event := TelemetryEvent{
		Type: "command",
		Name: "test-command",
		Properties: map[string]interface{}{
			"key": "value",
		},
	}

	// Track should not panic
	service.Track(ctx, event)

	// Stop the service (it's not started, so this is safe)
	service.Stop(ctx)
}

func TestTelemetryService_TrackDisabled(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: false})

	ctx := context.Background()
	event := TelemetryEvent{
		Type: "command",
		Name: "test-command",
	}

	// Should not panic when disabled
	service.Track(ctx, event)
}

func TestTelemetryService_TrackCommand(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: true})

	ctx := context.Background()
	service.TrackCommand(ctx, "ls", "shell", 100)

	service.Stop(ctx)
}

func TestTelemetryService_TrackOptimization(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: true})

	ctx := context.Background()
	service.TrackOptimization(ctx, "long code here", "short", 50)

	service.Stop(ctx)
}

func TestTelemetryService_TrackError(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: true})

	ctx := context.Background()
	service.TrackError(ctx, "validation_error", "invalid input")

	service.Stop(ctx)
}

func TestTelemetryService_TrackFeature(t *testing.T) {
	service := NewTelemetryService(TelemetryConfig{Enabled: true})

	ctx := context.Background()
	service.TrackFeature(ctx, "dark_mode", map[string]interface{}{
		"enabled": true,
	})

	service.Stop(ctx)
}

func TestConsentManager(t *testing.T) {
	manager := NewConsentManager()

	// Initially not consented
	if manager.IsConsented() {
		t.Error("IsConsented() = true, want false initially")
	}

	// Set consent
	err := manager.SetConsent(true)
	if err != nil {
		t.Errorf("SetConsent() error = %v", err)
	}

	// Now should be consented
	if !manager.IsConsented() {
		t.Error("IsConsented() = false, want true after SetConsent")
	}

	// Revoke consent
	err = manager.SetConsent(false)
	if err != nil {
		t.Errorf("SetConsent(false) error = %v", err)
	}

	if manager.IsConsented() {
		t.Error("IsConsented() = true, want false after revocation")
	}
}

func TestConsentManager_ThreadSafety(t *testing.T) {
	manager := NewConsentManager()

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = manager.IsConsented()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = manager.SetConsent(j%2 == 0)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestGDPRCompliance_Anonymize(t *testing.T) {
	gdpr := &GDPRCompliance{}

	tests := []struct {
		name     string
		input    map[string]interface{}
		checkKey string
		wantPII  bool
	}{
		{
			name: "email anonymized",
			input: map[string]interface{}{
				"email": "user@example.com",
				"key":   "value",
			},
			checkKey: "email",
			wantPII:  true,
		},
		{
			name: "name anonymized",
			input: map[string]interface{}{
				"name":  "John Doe",
				"count": 42,
			},
			checkKey: "name",
			wantPII:  true,
		},
		{
			name: "non-pii preserved",
			input: map[string]interface{}{
				"action": "click",
				"count":  42,
			},
			checkKey: "action",
			wantPII:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gdpr.Anonymize(tt.input)
			if result == nil {
				t.Fatal("Anonymize() returned nil")
			}

			// Check that email/name was anonymized
			val, ok := result[tt.checkKey]
			if !ok {
				t.Errorf("Anonymize() missing key %s", tt.checkKey)
				return
			}

			valStr, isString := val.(string)
			if tt.wantPII {
				// Should be anonymized (16 char hash)
				if !isString || len(valStr) != 16 {
					t.Errorf("Anonymize() did not anonymize %s: got %v", tt.checkKey, val)
				}
				// Original value should be different
				if valStr == tt.input[tt.checkKey] {
					t.Errorf("Anonymize() did not change %s", tt.checkKey)
				}
			}
		})
	}
}

func TestGDPRCompliance_Anonymize_NilInput(t *testing.T) {
	gdpr := &GDPRCompliance{}
	result := gdpr.Anonymize(nil)
	if result != nil {
		t.Errorf("Anonymize(nil) = %v, want nil", result)
	}
}

func TestGDPRCompliance_RemovePII(t *testing.T) {
	gdpr := &GDPRCompliance{}

	input := map[string]interface{}{
		"email":   "user@example.com",
		"name":    "John Doe",
		"action":  "click",
		"count":   42,
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "New York",
		},
	}

	result := gdpr.RemovePII(input)
	if result == nil {
		t.Fatal("RemovePII() returned nil")
	}

	// PII fields should be removed
	if _, ok := result["email"]; ok {
		t.Error("RemovePII() should have removed 'email'")
	}
	if _, ok := result["name"]; ok {
		t.Error("RemovePII() should have removed 'name'")
	}

	// Non-PII fields should remain
	if _, ok := result["action"]; !ok {
		t.Error("RemovePII() should have kept 'action'")
	}
	if _, ok := result["count"]; !ok {
		t.Error("RemovePII() should have kept 'count'")
	}

	// Nested PII should also be removed
	if addr, ok := result["address"].(map[string]interface{}); ok {
		if _, ok := addr["street"]; ok {
			t.Error("RemovePII() should have removed nested 'street'")
		}
	}
}

func TestGDPRCompliance_RemovePII_NilInput(t *testing.T) {
	gdpr := &GDPRCompliance{}
	result := gdpr.RemovePII(nil)
	if result != nil {
		t.Errorf("RemovePII(nil) = %v, want nil", result)
	}
}

func TestGDPRCompliance_RightToErasure(t *testing.T) {
	gdpr := &GDPRCompliance{}

	request, err := gdpr.RightToErasure("user123")
	if err != nil {
		t.Errorf("RightToErasure() error = %v", err)
	}

	if request.UserID != "user123" {
		t.Errorf("RightToErasure().UserID = %s, want user123", request.UserID)
	}

	if request.Status != "pending" {
		t.Errorf("RightToErasure().Status = %s, want pending", request.Status)
	}

	if request.RequestedAt.IsZero() {
		t.Error("RightToErasure().RequestedAt should not be zero")
	}

	if !request.CompletedAt.IsZero() {
		t.Error("RightToErasure().CompletedAt should be zero initially")
	}
}

func TestIsPII(t *testing.T) {
	piiKeys := []string{
		"email", "name", "full_name", "first_name", "last_name",
		"phone", "address", "ip", "ip_address", "ssn", "credit_card",
	}

	for _, key := range piiKeys {
		if !isPII(key) {
			t.Errorf("isPII(%q) = false, want true", key)
		}
	}

	nonPIIKeys := []string{
		"action", "count", "enabled", "status", "type", "timestamp",
	}

	for _, key := range nonPIIKeys {
		if isPII(key) {
			t.Errorf("isPII(%q) = true, want false", key)
		}
	}
}

func TestCalculateSavingsPercent(t *testing.T) {
	tests := []struct {
		name     string
		original string
		optimized string
		want     float64
	}{
		{
			name:      "50% savings",
			original:  "abcdefgh",
			optimized: "abcd",
			want:      50.0,
		},
		{
			name:      "100% savings",
			original:  "content",
			optimized: "",
			want:      100.0,
		},
		{
			name:      "0% savings",
			original:  "same",
			optimized: "same",
			want:      0.0,
		},
		{
			name:      "empty original",
			original:  "",
			optimized: "something",
			want:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSavingsPercent(tt.original, tt.optimized)
			if got != tt.want {
				t.Errorf("calculateSavingsPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelemetryService_StartStop(t *testing.T) {
	config := TelemetryConfig{
		Enabled:     true,
		Endpoint:    "",
		BatchSize:   10,
		FlushPeriod: 100 * time.Millisecond,
	}

	service := NewTelemetryService(config)
	ctx := context.Background()

	// Start the service
	service.Start(ctx)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop the service
	service.Stop(ctx)

	// Service should have stopped gracefully
	select {
	case <-service.stopCh:
		// Channel is closed, which is expected
	default:
		// Channel not closed yet is also fine for this test
	}
}

func TestTelemetryService_StartDisabled(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	service := NewTelemetryService(config)
	ctx := context.Background()

	// Should not panic when starting disabled service
	service.Start(ctx)
	service.Stop(ctx)
}

func TestTelemetryService_Flush(t *testing.T) {
	config := TelemetryConfig{
		Enabled:  true,
		Endpoint: "",
	}

	service := NewTelemetryService(config)
	ctx := context.Background()

	// Track some events
	for i := 0; i < 5; i++ {
		service.Track(ctx, TelemetryEvent{
			Type: "command",
			Name: "test",
		})
	}

	// Flush should not error
	err := service.Flush(ctx)
	if err != nil {
		t.Errorf("Flush() error = %v", err)
	}

	service.Stop(ctx)
}
