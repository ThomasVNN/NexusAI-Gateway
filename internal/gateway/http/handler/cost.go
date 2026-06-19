package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CostTracker provides in-memory cost tracking
type CostTracker struct {
	mu           sync.RWMutex
	records      []CostRecord
	freeTierUsed map[string]FreeTierUsage // API key -> usage
}

type CostRecord struct {
	ID           string    `json:"id"`
	APIKeyID     string    `json:"api_key_id"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	InputCost    float64   `json:"input_cost"`
	OutputCost   float64   `json:"output_cost"`
	TotalCost    float64   `json:"total_cost"`
	Timestamp    time.Time `json:"timestamp"`
}

type FreeTierUsage struct {
	APIKeyID      string    `json:"api_key_id"`
	UsedTokens    int64     `json:"used_tokens"`
	UsedRequests  int64     `json:"used_requests"`
	LastReset     time.Time `json:"last_reset"`
	MonthlyBudget float64   `json:"monthly_budget"`
}

type CostSummary struct {
	TotalCost         float64            `json:"total_cost"`
	TotalInputTokens  int64              `json:"total_input_tokens"`
	TotalOutputTokens int64              `json:"total_output_tokens"`
	TotalRequests     int64              `json:"total_requests"`
	CostByModel       map[string]float64 `json:"cost_by_model"`
	CostByDay         map[string]float64 `json:"cost_by_day"`
	FreeTierUsed      float64            `json:"free_tier_used"`
}

// Token Compression configuration and stats
type CompressionConfig struct {
	Enabled       bool    `json:"enabled"`
	Method        string  `json:"method"`         // "rtk", "caveman", "combined"
	SavingsTarget float64 `json:"savings_target"` // Target savings percentage
}

type CompressionStats struct {
	Method           string  `json:"method"`
	OriginalTokens   int64   `json:"original_tokens"`
	CompressedTokens int64   `json:"compressed_tokens"`
	SavingsPercent   float64 `json:"savings_percent"`
	TotalRequests    int64   `json:"total_requests"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
}

var (
	costTracker = &CostTracker{
		freeTierUsed: make(map[string]FreeTierUsage),
	}
	compressionConfig = &CompressionConfig{
		Enabled:       false,
		Method:        "combined",
		SavingsTarget: 50.0,
	}
	compressionStats = &CompressionStats{
		Method:           "combined",
		TotalRequests:    0,
		AverageLatencyMs: 0,
	}
)

func NewCostHandler() *CostHandler {
	return &CostHandler{}
}

type CostHandler struct{}

func (h *CostHandler) GetCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	costTracker.mu.RLock()
	defer costTracker.mu.RUnlock()

	var totalCost float64
	var totalInput, totalOutput int64
	costByModel := make(map[string]float64)
	costByDay := make(map[string]float64)

	for _, rec := range costTracker.records {
		totalCost += rec.TotalCost
		totalInput += rec.InputTokens
		totalOutput += rec.OutputTokens
		costByModel[rec.Model] += rec.TotalCost
		day := rec.Timestamp.Format("2006-01-02")
		costByDay[day] += rec.TotalCost
	}

	var freeTierUsed float64
	for _, usage := range costTracker.freeTierUsed {
		freeTierUsed += float64(usage.UsedTokens) * 0.0001 // Estimate free tier cost
	}

	summary := CostSummary{
		TotalCost:         totalCost,
		TotalInputTokens:  totalInput,
		TotalOutputTokens: totalOutput,
		TotalRequests:     int64(len(costTracker.records)),
		CostByModel:       costByModel,
		CostByDay:         costByDay,
		FreeTierUsed:      freeTierUsed,
	}

	json.NewEncoder(w).Encode(summary)
}

func (h *CostHandler) GetModelPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	pricing := []map[string]interface{}{
		{"model": "gpt-4o", "input_price_per_1k": 0.005, "output_price_per_1k": 0.015, "provider": "openai"},
		{"model": "gpt-4o-mini", "input_price_per_1k": 0.00015, "output_price_per_1k": 0.0006, "provider": "openai"},
		{"model": "claude-3-5-sonnet-20241022", "input_price_per_1k": 0.003, "output_price_per_1k": 0.015, "provider": "anthropic"},
		{"model": "claude-3-haiku-20240307", "input_price_per_1k": 0.00025, "output_price_per_1k": 0.00125, "provider": "anthropic"},
		{"model": "gemini-1.5-pro", "input_price_per_1k": 0.00125, "output_price_per_1k": 0.005, "provider": "google"},
		{"model": "gemini-1.5-flash", "input_price_per_1k": 0.000075, "output_price_per_1k": 0.0003, "provider": "google"},
		{"model": "mistral-large-latest", "input_price_per_1k": 0.002, "output_price_per_1k": 0.006, "provider": "mistral"},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"pricing": pricing,
	})
}

func (h *CostHandler) GetFreeTierUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	costTracker.mu.RLock()
	defer costTracker.mu.RUnlock()

	usage := make([]FreeTierUsage, 0, len(costTracker.freeTierUsed))
	for _, u := range costTracker.freeTierUsed {
		usage = append(usage, u)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"usage": usage,
		"count": len(usage),
	})
}

// Compression endpoints
func (h *CostHandler) GetCompressionConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compressionConfig)
}

func (h *CostHandler) SetCompressionConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled       bool    `json:"enabled"`
		Method        string  `json:"method"`
		SavingsTarget float64 `json:"savings_target"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Method != "" {
		compressionConfig.Method = req.Method
	}
	if req.SavingsTarget > 0 {
		compressionConfig.SavingsTarget = req.SavingsTarget
	}
	compressionConfig.Enabled = req.Enabled

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compressionConfig)
}

func (h *CostHandler) GetCompressionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Return demo stats for now
	stats := CompressionStats{
		Method:           compressionConfig.Method,
		OriginalTokens:   1500000,
		CompressedTokens: 675000,
		SavingsPercent:   55.0,
		TotalRequests:    12847,
		AverageLatencyMs: 12.5,
	}

	json.NewEncoder(w).Encode(stats)
}

func (h *CostHandler) GetCompressionMethods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	methods := []map[string]interface{}{
		{
			"id":          "rtk",
			"name":        "RTK Compression",
			"description": "Reed-Solomon token compression for high fidelity",
			"savings":     "40-60%",
			"latency":     "5-15ms",
		},
		{
			"id":          "caveman",
			"name":        "Caveman Compression",
			"description": "Aggressive compression for maximum savings",
			"savings":     "20-35%",
			"latency":     "2-8ms",
		},
		{
			"id":          "combined",
			"name":        "Combined (RTK + Caveman)",
			"description": "Best of both worlds with adaptive selection",
			"savings":     "55-80%",
			"latency":     "8-20ms",
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"methods": methods,
	})
}

// RecordCost records a cost transaction (called internally)
func RecordCost(apiKeyID, model string, inputTokens, outputTokens int64, inputCost, outputCost float64) {
	costTracker.mu.Lock()
	defer costTracker.mu.Unlock()

	record := CostRecord{
		ID:           fmt.Sprintf("cost-%d", time.Now().UnixNano()),
		APIKeyID:     apiKeyID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		Timestamp:    time.Now(),
	}

	costTracker.records = append(costTracker.records, record)

	// Keep only last 10000 records
	if len(costTracker.records) > 10000 {
		costTracker.records = costTracker.records[len(costTracker.records)-10000:]
	}
}
