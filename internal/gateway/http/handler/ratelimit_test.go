package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/ratelimit"
)

// mockQuotaManager implements quotaManagerInterface for testing
type mockQuotaManager struct {
	allowed   bool
	remaining int
	resetTime time.Time
}

func (m *mockQuotaManager) Allow(tenantID string) bool {
	return m.allowed
}

func (m *mockQuotaManager) GetRemaining(tenantID string) int {
	return m.remaining
}

func (m *mockQuotaManager) GetResetTime(tenantID string) time.Time {
	return m.resetTime
}

func (m *mockQuotaManager) Reset(tenantID string) {
}

func TestQuotaHandler_GetRateLimitStatus(t *testing.T) {
	manager := &mockQuotaManager{
		allowed:   true,
		remaining: 55,
		resetTime: time.Now().Add(time.Minute),
	}
	handler := NewQuotaHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/status?tenant_id=tenant-test", nil)
	rec := httptest.NewRecorder()

	handler.GetRateLimitStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestQuotaHandler_GetTierConfigs(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/tiers", nil)
	rec := httptest.NewRecorder()

	handler.GetTierConfigs(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	tiers, ok := response["tiers"].([]interface{})
	if !ok {
		t.Fatal("Expected tiers to be an array")
	}

	if len(tiers) != 4 {
		t.Errorf("Expected 4 tiers, got %d", len(tiers))
	}
}

func TestQuotaHandler_UpdateTierConfig(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"tier":"free","requests_per_min":20,"burst_size":10,"enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rate-limits/tiers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateTierConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "updated" {
		t.Errorf("Expected status 'updated', got '%s'", response["status"])
	}
}

func TestQuotaHandler_UpdateTierConfig_InvalidTier(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"tier":"invalid","requests_per_min":20}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rate-limits/tiers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateTierConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestQuotaHandler_UpdateTierConfig_InvalidJSON(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rate-limits/tiers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.UpdateTierConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestQuotaHandler_GetQuotaUsage(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/usage?tenant_id=tenant-usage", nil)
	rec := httptest.NewRecorder()

	handler.GetQuotaUsage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["tenant_id"] != "tenant-usage" {
		t.Errorf("Expected tenant_id 'tenant-usage', got '%s'", response["tenant_id"])
	}
}

func TestQuotaHandler_GetQuotaUsage_MissingTenant(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/usage", nil)
	rec := httptest.NewRecorder()

	handler.GetQuotaUsage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestQuotaHandler_ResetQuota(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"tenant_id":"tenant-reset","scope":["all"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/rate-limits/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ResetQuota(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "reset" {
		t.Errorf("Expected status 'reset', got '%s'", response["status"])
	}
}

func TestQuotaHandler_ResetQuota_MissingTenant(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"scope":["all"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/rate-limits/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ResetQuota(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestQuotaHandler_SetQuota(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"tenant_id":"tenant-quota","tier":"pro","custom_limit":200}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rate-limits/quota", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SetQuota(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "quota_set" {
		t.Errorf("Expected status 'quota_set', got '%s'", response["status"])
	}
}

func TestQuotaHandler_SetQuota_MissingTenant(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	body := `{"tier":"pro"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/rate-limits/quota", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SetQuota(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestQuotaHandler_HealthCheck(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestGetRateLimitsHandler_ServeHTTP(t *testing.T) {
	manager := &mockQuotaManager{}

	handler := NewGetRateLimitsHandler(manager)

	tests := []struct {
		path   string
		method string
		code   int
	}{
		{"/v1/rate-limits/status", http.MethodGet, http.StatusOK},
		{"/v1/rate-limits/tiers", http.MethodGet, http.StatusOK},
		{"/v1/rate-limits/usage?tenant_id=test", http.MethodGet, http.StatusOK},
		{"/v1/rate-limits/health", http.MethodGet, http.StatusOK},
		{"/v1/rate-limits/unknown", http.MethodGet, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.code {
				t.Errorf("Expected status %d, got %d", tt.code, rec.Code)
			}
		})
	}
}

func TestGetRateLimitsHandler_MethodNotAllowed(t *testing.T) {
	manager := &mockQuotaManager{}

	handler := NewGetRateLimitsHandler(manager)

	// Test POST on status endpoint (should be method not allowed)
	req := httptest.NewRequest(http.MethodPost, "/v1/rate-limits/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// POST is not defined for status, so it should be method not allowed
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", rec.Code)
	}
}

func TestGetRateLimitsHandler_UnknownPath(t *testing.T) {
	manager := &mockQuotaManager{}

	handler := NewGetRateLimitsHandler(manager)

	// Test unknown path should return 404
	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", rec.Code)
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		url        string
		defaultVal int
		expected   int
	}{
		{"Valid value", "page", "/?page=5", 0, 5},
		{"Missing param", "page", "/", 10, 10},
		{"Invalid value", "page", "/?page=abc", 10, 10},
		{"Empty value", "page", "/?page=", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			result := ParseIntParam(req, tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestQuotaHandler_InvalidMethod(t *testing.T) {
	manager := &mockQuotaManager{}
	handler := NewQuotaHandler(manager)

	// Test invalid method on UpdateTierConfig
	req := httptest.NewRequest(http.MethodGet, "/v1/rate-limits/tiers", nil)
	rec := httptest.NewRecorder()

	handler.UpdateTierConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", rec.Code)
	}

	// Test invalid method on GetQuotaUsage
	req = httptest.NewRequest(http.MethodPost, "/v1/rate-limits/usage", nil)
	rec = httptest.NewRecorder()

	handler.GetQuotaUsage(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", rec.Code)
	}

	// Test invalid method on ResetQuota
	req = httptest.NewRequest(http.MethodGet, "/v1/rate-limits/reset", nil)
	rec = httptest.NewRecorder()

	handler.ResetQuota(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", rec.Code)
	}

	// Test invalid method on SetQuota
	req = httptest.NewRequest(http.MethodGet, "/v1/rate-limits/quota", nil)
	rec = httptest.NewRecorder()

	handler.SetQuota(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", rec.Code)
	}
}

// Test with ratelimit manager integration
func TestQuotaHandler_WithRatelimitManager(t *testing.T) {
	storage := ratelimit.NewInMemoryStorage()
	config := ratelimit.DefaultRateLimitConfig()
	ratelimitManager := ratelimit.NewQuotaManager(storage, config)

	// Test that the manager works correctly
	ctx := context.Background()
	result, err := ratelimitManager.CheckTenantLimit(ctx, "test-tenant", ratelimit.TierFree)
	if err != nil {
		t.Fatalf("CheckTenantLimit failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}

	if result.Remaining <= 0 {
		t.Errorf("Expected remaining > 0, got %d", result.Remaining)
	}
}
