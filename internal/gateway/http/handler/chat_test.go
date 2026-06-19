package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/integration"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/runtime"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
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

// Mock integration clients
type mockModelPlatform struct {
	response *integration.ModelRouteResponse
	err      error
}

func (m *mockModelPlatform) Route(ctx context.Context, request *integration.ModelRouteRequest) (*integration.ModelRouteResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

type mockSkillsClient struct {
	response *integration.SkillResponse
	err      error
}

func (m *mockSkillsClient) Execute(ctx context.Context, request *integration.SkillRequest) (*integration.SkillResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

type mockKnowledgeClient struct {
	content string
	err     error
}

func (m *mockKnowledgeClient) Retrieve(ctx context.Context, query string) (string, error) {
	return m.content, m.err
}

func TestChatHandlerSuccess(t *testing.T) {
	rawKey := "ork_testkey123"
	keyHash := auth.HashKey(rawKey)

	keyRepo := &mockKeyRepo{
		key: &model.RegisteredKey{
			ID:         "test-key-id",
			KeyHash:    keyHash,
			DailyQuota: 10,
			Active:     true,
			SourceApp:  "test-app",
		},
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	// Pipeline mocks
	authMock := auth.NewAPIKeyAuthenticator(keyRepo)
	tenantMock := tenancy.NewDefaultTenantResolver()
	knowledgeMock := &mockKnowledgeClient{content: "System knowledge context"}
	skillsMock := &mockSkillsClient{}
	modelMock := &mockModelPlatform{
		response: &integration.ModelRouteResponse{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
			URL:        "https://api.openai.com/v1",
		},
	}

	pipelineExecutor := runtime.NewPipelineExecutor(
		authMock,
		tenantMock,
		piiEngine,
		knowledgeMock,
		skillsMock,
		modelMock,
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false, pipelineExecutor)

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
	req.Header.Set("X-Correlation-ID", "custom-trace-12345")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", contentType)
	}

	var resp RuntimeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success true, got false")
	}

	if resp.Data == nil {
		t.Fatalf("expected data field to be populated")
	}

	// Verify PII was redacted (secret@google.com should be scrubbed by privacy engine)
	if strings.Contains(resp.Data.Answer, "secret@google.com") {
		t.Errorf("expected sensitive email to be redacted in pipeline output")
	}

	// Verify trace ID propagation
	if resp.Data.TraceID != "custom-trace-12345" {
		t.Errorf("expected TraceID to be custom-trace-12345, got %s", resp.Data.TraceID)
	}

	if resp.Data.Usage.TotalTokens == 0 {
		t.Errorf("expected usage tokens to be non-zero")
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
	usageRepo := &mockUsageRepo{dailyUsage: 15}
	piiEngine := privacy.NewEngine()

	pipelineExecutor := runtime.NewPipelineExecutor(
		auth.NewAPIKeyAuthenticator(keyRepo),
		tenancy.NewDefaultTenantResolver(),
		piiEngine,
		&mockKnowledgeClient{},
		&mockSkillsClient{},
		&mockModelPlatform{},
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false, pipelineExecutor)

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

	keyRepo := &mockKeyRepo{
		err: errors.New("key not found by hash"),
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	pipelineExecutor := runtime.NewPipelineExecutor(
		auth.NewAPIKeyAuthenticator(keyRepo),
		tenancy.NewDefaultTenantResolver(),
		piiEngine,
		&mockKnowledgeClient{},
		&mockSkillsClient{},
		&mockModelPlatform{},
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false, pipelineExecutor)

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

	keyRepo := &mockKeyRepo{
		err: errors.New("pq: connection refused"),
	}
	usageRepo := &mockUsageRepo{dailyUsage: 0}
	piiEngine := privacy.NewEngine()

	pipelineExecutor := runtime.NewPipelineExecutor(
		auth.NewAPIKeyAuthenticator(keyRepo),
		tenancy.NewDefaultTenantResolver(),
		piiEngine,
		&mockKnowledgeClient{},
		&mockSkillsClient{},
		&mockModelPlatform{},
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, true, pipelineExecutor)

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
	usageRepo := &mockUsageRepo{
		err: errors.New("driver: bad connection"),
	}
	piiEngine := privacy.NewEngine()

	pipelineExecutor := runtime.NewPipelineExecutor(
		auth.NewAPIKeyAuthenticator(keyRepo),
		tenancy.NewDefaultTenantResolver(),
		piiEngine,
		&mockKnowledgeClient{},
		&mockSkillsClient{},
		&mockModelPlatform{},
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false, pipelineExecutor)

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

	authMock := auth.NewAPIKeyAuthenticator(keyRepo)
	tenantMock := tenancy.NewDefaultTenantResolver()
	knowledgeMock := &mockKnowledgeClient{}
	skillsMock := &mockSkillsClient{}
	modelMock := &mockModelPlatform{
		err: errors.New("model routing failed: upstream provider is unreachable"),
	}

	pipelineExecutor := runtime.NewPipelineExecutor(
		authMock,
		tenantMock,
		piiEngine,
		knowledgeMock,
		skillsMock,
		modelMock,
	)

	handler := NewChatHandler(keyRepo, usageRepo, piiEngine, false, pipelineExecutor)

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
