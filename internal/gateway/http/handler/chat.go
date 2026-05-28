package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type ChatHandler struct {
	keyRepo   repository.KeyRepository
	usageRepo repository.UsageRepository
	piiEngine *privacy.Engine
}

func NewChatHandler(kr repository.KeyRepository, ur repository.UsageRepository, pe *privacy.Engine) *ChatHandler {
	return &ChatHandler{
		keyRepo:   kr,
		usageRepo: ur,
		piiEngine: pe,
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
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	keyHash := auth.HashKey(rawKey)
	key, err := h.keyRepo.GetByHash(r.Context(), keyHash)
	if err != nil {
		// Fallback mock key for local architectural sandbox if DB is empty
		key = &model.RegisteredKey{
			ID:          "mock-local-key",
			KeyHash:     keyHash,
			Name:        "Default Sandbox Key",
			SourceApp:   "sandbox",
			DailyQuota:  1000,
			HourlyQuota: 200,
			Active:      true,
		}
	}

	if !key.Active {
		http.Error(w, "Forbidden: API key is deactivated", http.StatusForbidden)
		return
	}

	// 2. Enforce Daily & Hourly Quotas
	dailyUsage, err := h.usageRepo.GetDailyUsage(r.Context(), key.ID)
	if err == nil && dailyUsage >= key.DailyQuota {
		http.Error(w, "Too Many Requests: Daily quota limit exceeded", http.StatusTooManyRequests)
		return
	}

	// 3. Classify Source App
	sourceApp := privacy.DetectSourceApp(r)

	// 4. Parse & Redact Prompt (PII scrubbing)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request: Failed to read request body", http.StatusBadRequest)
		return
	}

	var payload RequestPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Bad Request: Invalid JSON payload", http.StatusBadRequest)
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
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
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
			log.Printf("Upstream request creation failed: %v", err)
			h.executeMockFallbackStream(w, flusher, payload.Model)
			completionTokensEstimate = 50
		} else {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+upstreamKey)
			req.Header.Set("Accept", "text/event-stream")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				log.Printf("Upstream invocation failed, falling back to mock stream: %v", err)
				h.executeMockFallbackStream(w, flusher, payload.Model)
				completionTokensEstimate = 50
			} else {
				defer resp.Body.Close()
				buffer := make([]byte, 1024)
				for {
					n, err := resp.Body.Read(buffer)
					if n > 0 {
						_, _ = w.Write(buffer[:n])
						flusher.Flush()
						completionTokensEstimate += n / 10 // Simple token estimation from raw SSE stream length
					}
					if err == io.EOF {
						break
					}
					if err != nil {
						break
					}
				}
			}
		}
	} else {
		// Fallback to internal high-performance Mock Stream
		h.executeMockFallbackStream(w, flusher, payload.Model)
		completionTokensEstimate = 50
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
