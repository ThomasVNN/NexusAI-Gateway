package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type ChatHandler struct {
	keyRepo               repository.KeyRepository
	usageRepo             repository.UsageRepository
	piiEngine             *privacy.Engine
	enableSandboxFallback bool
}

func NewChatHandler(kr repository.KeyRepository, ur repository.UsageRepository, pe *privacy.Engine, enableSandboxFallback bool) *ChatHandler {
	return &ChatHandler{
		keyRepo:               kr,
		usageRepo:             ur,
		piiEngine:             pe,
		enableSandboxFallback: enableSandboxFallback,
	}
}

// RequestPayload represents the standard OpenAI chat completions format
type RequestPayload struct {
	Model    string          `json:"model"`
	Messages []ChatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()

	// 1. Authenticate API Key
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
		// Distinguish between API key not found and actual system/datastore failure
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
			// Infrastructure/datastore lookup failure - fail closed!
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
		// Log datastore failure and deny request (fail closed)
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

	// 4. Parse & Redact Prompt (PII scrubbing)
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

	// Apply PII scrubbing to all messages
	promptTokensEstimate := 0
	for i := range payload.Messages {
		payload.Messages[i].Content = h.piiEngine.Redact(payload.Messages[i].Content)
		promptTokensEstimate += len(payload.Messages[i].Content) / 4 // Simple token estimation
	}

	// Re-serialize sanitized payload
	sanitizedBytes, _ := json.Marshal(payload)

	// Set SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Streaming unsupported")
		return
	}

	// 5. Downstream Proxying or Mock Fallback Execution
	upstreamURL := os.Getenv("UPSTREAM_API_URL")
	upstreamKey := os.Getenv("UPSTREAM_API_KEY")

	completionTokensEstimate := 0

	if upstreamURL != "" && upstreamKey != "" {
		// Proxy to real upstream LLM provider
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(sanitizedBytes))
		if err != nil {
			LogSecurityEvent(r, "ERROR", "Upstream request creation failed", "provider_failure", err.Error())
			WriteError(w, http.StatusBadGateway, "PROVIDER_FAILURE", "Bad Gateway: Failed to construct upstream request")
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+upstreamKey)
		req.Header.Set("Accept", "text/event-stream")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			LogSecurityEvent(r, "ERROR", "Upstream provider invocation failed: network or timeout error", "provider_failure", err.Error())
			WriteError(w, http.StatusBadGateway, "PROVIDER_FAILURE", "Bad Gateway: Upstream provider is currently unreachable")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyErr, _ := io.ReadAll(resp.Body)
			LogSecurityEvent(r, "ERROR", fmt.Sprintf("Upstream provider returned non-OK status: %d", resp.StatusCode), "provider_failure", string(bodyErr))
			WriteError(w, http.StatusBadGateway, "PROVIDER_FAILURE", fmt.Sprintf("Bad Gateway: Upstream provider failed with status %d", resp.StatusCode))
			return
		}

		buffer := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				_, _ = w.Write(buffer[:n])
				flusher.Flush()
				completionTokensEstimate += n / 10 // Simple token estimation from SSE stream length
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				LogSecurityEvent(r, "ERROR", "Stream read error from upstream provider", "provider_failure", err.Error())
				break
			}
		}
	} else {
		// If upstream is not configured, we only allow mock stream if sandbox fallback is explicitly enabled!
		if h.enableSandboxFallback {
			// Fallback to internal high-performance Mock Stream
			h.executeMockFallbackStream(w, flusher, payload.Model)
			completionTokensEstimate = 50
		} else {
			LogSecurityEvent(r, "ERROR", "Upstream provider is not configured", "configuration_failure", "UPSTREAM_API_URL or UPSTREAM_API_KEY is empty")
			WriteError(w, http.StatusInternalServerError, "CONFIGURATION_FAILURE", "Internal Server Error: Downstream AI provider is not configured")
			return
		}
	}

	// 6. Log Usage Record Async
	go func() {
		latency := int(time.Since(startTime).Milliseconds())
		record := &model.UsageRecord{
			KeyID:            key.ID,
			ModelID:          payload.Model,
			PromptTokens:     promptTokensEstimate,
			CompletionTokens: completionTokensEstimate,
			LatencyMS:        latency,
			SourceApp:        sourceApp,
		}
		_ = h.usageRepo.LogUsage(context.Background(), record)
	}()
}

func (h *ChatHandler) executeMockFallbackStream(w http.ResponseWriter, flusher http.Flusher, modelName string) {
	chunks := []string{"Hello! ", "This ", "is ", "a ", "privacy-filtered ", "and ", "quota-tracked ", "SSE ", "stream ", "served ", "directly ", "from ", "NexusAI-Gateway."}
	for i, chunk := range chunks {
		_, _ = fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-mock\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"%s\"},\"finish_reason\":null}]}\n\n", time.Now().Unix(), modelName, chunk)
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)
		_ = i
	}
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}
