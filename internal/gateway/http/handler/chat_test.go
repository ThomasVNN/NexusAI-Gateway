package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type mockKeyRepo struct {
	key *model.RegisteredKey
	err error
}

func (m *mockKeyRepo) GetByID(ctx context.Context, id string) (*model.RegisteredKey, error) {
	return m.key, m.err
}

func (m *mockKeyRepo) GetByHash(ctx context.Context, hash string) (*model.RegisteredKey, error) {
	return m.key, m.err
}

func (m *mockKeyRepo) Save(ctx context.Context, key *model.RegisteredKey) error {
	return nil
}

func (m *mockKeyRepo) ListAll(ctx context.Context) ([]*model.RegisteredKey, error) {
	return []*model.RegisteredKey{m.key}, m.err
}

type mockUsageRepo struct {
	dailyUsage int
	err        error
}

func (m *mockUsageRepo) LogUsage(ctx context.Context, record *model.UsageRecord) error {
	return nil
}

func (m *mockUsageRepo) GetHourlyUsage(ctx context.Context, keyID string) (int, error) {
	return 0, nil
}

func (m *mockUsageRepo) GetDailyUsage(ctx context.Context, keyID string) (int, error) {
	return m.dailyUsage, m.err
}

func (m *mockUsageRepo) GetAggregateUsage(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockUsageRepo) ListLogs(ctx context.Context) ([]*model.UsageRecord, error) {
	return nil, nil
}

func TestChatHandlerPIIScrubbingAndStreaming(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
		},
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	// Enable sandbox fallback so it falls back to mock stream when upstream is empty
	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, true)

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello, my email is secret@google.com!"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("expected content-type text/event-stream, got %s", contentType)
	}

	responseBody := rr.Body.String()
	if !strings.Contains(responseBody, "data:") {
		t.Fatalf("expected SSE data format in body, got: %s", responseBody)
	}
}

func TestChatHandlerQuotaExceeded(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
		},
	}
	// Setup mock usage repo to simulate quota exceeded limit
	usageRepo := &mockUsageRepo{dailyUsage: 15}
	piiEngine := privacy.NewEngine()

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false)

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "QUOTA_EXCEEDED" {
		t.Fatalf("expected error code QUOTA_EXCEEDED, got %s", errResp.Error.Code)
	}
}

func TestChatHandlerUnauthorized(t *testing.T) {
	rawKey := "ork_nonexistentkey"

	// Mock DB returning key not found (sql.ErrNoRows or key not found)
	keyRepo := &mockKeyRepo{
		err: errors.New("key not found by hash"),
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	// Disable sandbox fallback so it fails securely
	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false)

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "AUTHENTICATION_FAILED" {
		t.Fatalf("expected error code AUTHENTICATION_FAILED, got %s", errResp.Error.Code)
	}
}

func TestChatHandlerDatabaseFailure(t *testing.T) {
	rawKey := "ork_testkey123"

	// Simulate an actual infrastructure failure (like connection timeout/closed)
	keyRepo := &mockKeyRepo{
		err: errors.New("pq: connection refused"),
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	// Even if sandbox fallback is enabled, actual infrastructure failures must deny requests!
	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, true)

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "INFRASTRUCTURE_FAILURE" {
		t.Fatalf("expected error code INFRASTRUCTURE_FAILURE, got %s", errResp.Error.Code)
	}
}

func TestChatHandlerQuotaDatabaseFailure(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
		},
	}
	// Simulate datastore error during quota retrieval
	usageRepo := &mockUsageRepo{
		err: errors.New("driver: bad connection"),
	}
	piiEngine := privacy.NewEngine()

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false)

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "INFRASTRUCTURE_FAILURE" {
		t.Fatalf("expected error code INFRASTRUCTURE_FAILURE, got %s", errResp.Error.Code)
	}
}

func TestChatHandlerUpstreamProviderFailure(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
		},
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false)

	// Create an invalid/unreachable upstream URL to trigger provider failure
	os.Setenv("UPSTREAM_API_URL", "http://127.0.0.1:9999/v1/invalid-path")
	os.Setenv("UPSTREAM_API_KEY", "some-key")
	defer func() {
		os.Unsetenv("UPSTREAM_API_URL")
		os.Unsetenv("UPSTREAM_API_KEY")
	}()

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should deny requests and return a 502 Bad Gateway on provider invocation failure
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "PROVIDER_FAILURE" {
		t.Fatalf("expected error code PROVIDER_FAILURE, got %s", errResp.Error.Code)
	}
}

func TestChatHandlerConfigMissing(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
		},
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	// Disable sandbox fallback so it returns 500 when upstream is not configured
	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false)

	// Ensure upstream env is unset
	os.Unsetenv("UPSTREAM_API_URL")
	os.Unsetenv("UPSTREAM_API_KEY")

	payload := RequestPayload{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}

	var errResp APIErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode JSON error response: %v", err)
	}
	if errResp.Error.Code != "CONFIGURATION_FAILURE" {
		t.Fatalf("expected error code CONFIGURATION_FAILURE, got %s", errResp.Error.Code)
	}
}
