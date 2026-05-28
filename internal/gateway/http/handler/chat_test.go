package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type mockKeyRepo struct {
	key *model.RegisteredKey
}

func (m *mockKeyRepo) GetByID(ctx context.Context, id string) (*model.RegisteredKey, error) {
	return m.key, nil
}

func (m *mockKeyRepo) GetByHash(ctx context.Context, hash string) (*model.RegisteredKey, error) {
	return m.key, nil
}

func (m *mockKeyRepo) Save(ctx context.Context, key *model.RegisteredKey) error {
	return nil
}

func (m *mockKeyRepo) ListAll(ctx context.Context) ([]*model.RegisteredKey, error) {
	return []*model.RegisteredKey{m.key}, nil
}

type mockUsageRepo struct {
	dailyUsage int
}

func (m *mockUsageRepo) LogUsage(ctx context.Context, record *model.UsageRecord) error {
	return nil
}

func (m *mockUsageRepo) GetHourlyUsage(ctx context.Context, keyID string) (int, error) {
	return 0, nil
}

func (m *mockUsageRepo) GetDailyUsage(ctx context.Context, keyID string) (int, error) {
	return m.dailyUsage, nil
}

func (m *mockUsageRepo) GetAggregateUsage(ctx context.Context) (map[string]interface{}, error) {
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

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine)

	// Build payload containing sensitive PII data (email)
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
		t.Fatalf("expected status 200, got %d", rr.Code)
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

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine)

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
}
