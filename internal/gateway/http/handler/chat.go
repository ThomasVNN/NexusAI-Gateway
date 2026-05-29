package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/runtime"
)

type ChatHandler struct {
	keyRepo               repository.KeyRepository
	usageRepo             repository.UsageRepository
	piiEngine             *privacy.Engine
	enableSandboxFallback bool
	pipelineExecutor      *runtime.PipelineExecutor
}

func NewChatHandler(
	kr repository.KeyRepository,
	ur repository.UsageRepository,
	pe *privacy.Engine,
	enableSandboxFallback bool,
	pipelineExecutor *runtime.PipelineExecutor,
) *ChatHandler {
	return &ChatHandler{
		keyRepo:               kr,
		usageRepo:             ur,
		piiEngine:             pe,
		enableSandboxFallback: enableSandboxFallback,
		pipelineExecutor:      pipelineExecutor,
	}
}

// RequestPayload represents the standard OpenAI chat completions format
type RequestPayload struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RuntimeResponse represents the standardized JSON structure for success responses
type RuntimeResponse struct {
	Success bool                      `json:"success"`
	Data    *runtime.PipelineResponse `json:"data"`
	Meta    map[string]interface{}    `json:"meta"`
	Error   *ErrorPayload             `json:"error"`
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()

	// 1. Authenticate API Key & Pre-verify limits
	authHeader := r.Header.Get("Authorization")
	rawKey, err := auth.ParseKey(authHeader)
	if err != nil {
		LogSecurityEvent(r, "WARN", "Authentication failed due to malformed or missing key", "authentication_failed", err.Error())
		WriteError(w, http.StatusUnauthorized, "AUTHENTICATION_FAILED", fmt.Sprintf("Unauthorized: %v", err))
		return
	}

	keyHash := auth.HashKey(rawKey)
	key, err := h.keyRepo.GetByHash(r.Context(), keyHash)
	if err != nil {
		isNotFound := err == sql.ErrNoRows || err.Error() == "key not found by hash"

		if isNotFound {
			if h.enableSandboxFallback {
				LogSecurityEvent(r, "WARN", "API key not found, using sandbox fallback key", "sandbox_fallback_activated", "key hash not found in datastore")
				key = &model.RegisteredKey{
					ID:          "mock-local-key",
					KeyHash:     keyHash,
					Name:        "Default Sandbox Key",
					SourceApp:   "sandbox",
					DailyQuota:  1000,
					HourlyQuota: 200,
					Active:      true,
				}
			} else {
				LogSecurityEvent(r, "WARN", "Authentication failed: API key not found", "authentication_failed", "key hash not found in datastore")
				WriteError(w, http.StatusUnauthorized, "AUTHENTICATION_FAILED", "Unauthorized: Invalid API key")
				return
			}
		} else {
			LogSecurityEvent(r, "ERROR", "Authentication failed: Datastore unavailable", "datastore_failure", err.Error())
			WriteError(w, http.StatusServiceUnavailable, "INFRASTRUCTURE_FAILURE", "Service Unavailable: Authentication datastore is currently offline")
			return
		}
	}

	if !key.Active {
		LogSecurityEvent(r, "WARN", "Authorization failed: API key is deactivated", "authorization_failed", fmt.Sprintf("Key ID %s is deactivated", key.ID))
		WriteError(w, http.StatusForbidden, "AUTHORIZATION_FAILED", "Forbidden: API key is deactivated")
		return
	}

	// 2. Enforce Daily & Hourly Quotas
	dailyUsage, err := h.usageRepo.GetDailyUsage(r.Context(), key.ID)
	if err != nil {
		LogSecurityEvent(r, "ERROR", "Quota verification failed: Datastore unavailable", "datastore_failure", err.Error())
		WriteError(w, http.StatusServiceUnavailable, "INFRASTRUCTURE_FAILURE", "Service Unavailable: Quota check datastore is offline")
		return
	}
	if dailyUsage >= key.DailyQuota {
		LogSecurityEvent(r, "WARN", "Quota limit exceeded", "quota_exceeded", fmt.Sprintf("Key ID %s has daily usage %d / daily quota %d", key.ID, dailyUsage, key.DailyQuota))
		WriteError(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "Too Many Requests: Daily quota limit exceeded")
		return
	}

	// 3. Classify Source App
	sourceApp := privacy.DetectSourceApp(r)

	// 4. Parse payload
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Bad Request: Failed to read request body")
		return
	}

	var payload RequestPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Bad Request: Invalid JSON payload")
		return
	}

	// 5. Extract message
	var message string
	if len(payload.Messages) > 0 {
		message = payload.Messages[len(payload.Messages)-1].Content
	}

	// 6. Build runtime metadata and trace propagation
	metadata := make(map[string]interface{})
	metadata["model"] = payload.Model
	metadata["source_app"] = sourceApp

	corrID := r.Header.Get("X-Correlation-ID")
	if corrID == "" {
		corrID = r.Header.Get("X-Trace-ID")
	}
	if corrID != "" {
		metadata["trace_id"] = corrID
	}

	pipelineReq := &runtime.PipelineRequest{
		Message:   message,
		Tenant:    key.SourceApp,
		AuthToken: rawKey,
		Metadata:  metadata,
	}

	// 7. Execute Pipeline
	resp, err := h.pipelineExecutor.Execute(r.Context(), pipelineReq)
	if err != nil {
		LogSecurityEvent(r, "ERROR", "Pipeline execution failed", "pipeline_failure", err.Error())

		// Map errors to correct HTTP responses
		errStr := err.Error()
		if strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "invalid API key") {
			WriteError(w, http.StatusUnauthorized, "AUTHENTICATION_FAILED", err.Error())
		} else if strings.Contains(errStr, "model routing failed") {
			WriteError(w, http.StatusBadGateway, "PROVIDER_FAILURE", err.Error())
		} else if strings.Contains(errStr, "upstream provider failed") || strings.Contains(errStr, "unreachable") {
			WriteError(w, http.StatusBadGateway, "PROVIDER_FAILURE", err.Error())
		} else if strings.Contains(errStr, "configuration failure") || strings.Contains(errStr, "not configured") {
			WriteError(w, http.StatusInternalServerError, "CONFIGURATION_FAILURE", err.Error())
		} else {
			WriteError(w, http.StatusInternalServerError, "PIPELINE_ERROR", err.Error())
		}
		return
	}

	// 8. Return standard JSON RuntimeResponse
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	runtimeResp := RuntimeResponse{
		Success: true,
		Data:    resp,
		Meta:    map[string]interface{}{},
		Error:   nil,
	}
	_ = json.NewEncoder(w).Encode(runtimeResp)

	// 9. Log Usage Record Async using pipeline metrics
	go func() {
		latency := int(time.Since(startTime).Milliseconds())
		record := &model.UsageRecord{
			KeyID:            key.ID,
			ModelID:          payload.Model,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			LatencyMS:        latency,
			SourceApp:        sourceApp,
		}
		_ = h.usageRepo.LogUsage(context.Background(), record)
	}()
}
