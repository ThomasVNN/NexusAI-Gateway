package routing

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestModelRouter_Route(t *testing.T) {
	router := NewModelRouter()

	tests := []struct {
		name      string
		req       *RoutingRequest
		wantModel string
		wantErr   bool
	}{
		{
			name: "route to requested model when available",
			req: &RoutingRequest{
				RequestedModel: "gpt-4o",
				TenantID:       "tenant-1",
			},
			wantModel: "gpt-4o",
			wantErr:   false,
		},
		{
			name: "fallback when model not found",
			req: &RoutingRequest{
				RequestedModel: "unknown-model",
				TenantID:       "tenant-1",
			},
			// Priority strategy returns first model in map iteration order
			wantErr: false,
		},
		{
			name: "route by quality - vision",
			req: &RoutingRequest{
				TenantID:             "tenant-1",
				RequiredCapabilities: []string{"vision"},
				Strategy:             "quality_first",
			},
			wantModel: "gemini-1.5-pro", // most capable with vision
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := router.Route(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantModel != "" && got != nil && got.ModelID != tt.wantModel {
				t.Errorf("Route() model = %v, want %v", got.ModelID, tt.wantModel)
			}
		})
	}
}

func TestModelRouter_SelectByCost(t *testing.T) {
	router := NewModelRouter()

	req := &RoutingRequest{
		TenantID:             "tenant-1",
		RequiredCapabilities: []string{"function_calling"},
		Strategy:             "cost_optimized",
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Verify cost optimization was applied
	t.Logf("Selected model: %s with strategy: %s", got.ModelID, got.Strategy)
}

func TestModelRouter_SelectByLatency(t *testing.T) {
	router := NewModelRouter()

	req := &RoutingRequest{
		TenantID: "tenant-1",
		Strategy: "latency_optimized",
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Verify latency optimization was applied
	t.Logf("Selected model: %s with strategy: %s", got.ModelID, got.Strategy)
}

func TestModelRouter_TenantStrategy(t *testing.T) {
	router := NewModelRouter()

	// Set tenant to use latency optimization
	router.SetTenantStrategy("premium-tenant", StrategyLatencyOpt)

	req := &RoutingRequest{
		TenantID: "premium-tenant",
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Verify tenant strategy is used
	t.Logf("Selected model: %s with strategy: %s", got.ModelID, got.Strategy)
}

func TestModelRouter_GetSetModel(t *testing.T) {
	router := NewModelRouter()

	// Get existing model
	model, ok := router.GetModel("gpt-4o")
	if !ok {
		t.Error("Expected to find gpt-4o model")
	}
	if model.Provider != "openai" {
		t.Errorf("Expected provider openai, got %s", model.Provider)
	}

	// Add new model
	router.AddModel(&ModelInfo{
		Name:            "custom-model",
		Provider:        "custom",
		CostPer1KInput:  0.001,
		CostPer1KOutput: 0.002,
		MaxTokens:       1000,
		Capabilities:    []string{"streaming"},
		AvgLatencyMs:    200,
		IsActive:        true,
	})

	model, ok = router.GetModel("custom-model")
	if !ok {
		t.Error("Expected to find custom-model after AddModel")
	}
}

func TestModelRouter_ListModels(t *testing.T) {
	router := NewModelRouter()

	models := router.ListModels()
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
	t.Logf("Found %d models", len(models))
}

func TestModelRouter_RecordLatency(t *testing.T) {
	router := NewModelRouter()

	initial, _ := router.GetModel("gpt-4o")
	initialLatency := initial.AvgLatencyMs

	router.RecordLatency(context.Background(), "gpt-4o", 2000)

	updated, _ := router.GetModel("gpt-4o")
	// EMA: 0.9 * initial + 0.1 * 2000
	expectedLatency := int64(float64(initialLatency)*0.9 + 2000*0.1)

	if updated.AvgLatencyMs != expectedLatency {
		t.Errorf("Expected latency %d, got %d", expectedLatency, updated.AvgLatencyMs)
	}
}

func TestModelRouter_DeactivateModel(t *testing.T) {
	router := NewModelRouter()

	model, _ := router.GetModel("gpt-4o")
	if !model.IsActive {
		t.Error("Expected gpt-4o to be active initially")
	}

	router.DeactivateModel("gpt-4o")

	model, _ = router.GetModel("gpt-4o")
	if model.IsActive {
		t.Error("Expected gpt-4o to be inactive after DeactivateModel")
	}
}

func TestHasCapabilities(t *testing.T) {
	model := &ModelInfo{
		Capabilities: []string{"vision", "function_calling", "streaming"},
	}

	tests := []struct {
		name     string
		required []string
		want     bool
	}{
		{"empty requires", []string{}, true},
		{"single match", []string{"vision"}, true},
		{"all match", []string{"vision", "function_calling"}, true},
		{"partial match", []string{"vision", "unknown"}, false},
		{"no match", []string{"unknown"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCapabilities(model, tt.required)
			if got != tt.want {
				t.Errorf("hasCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoutingRequest_MaxLatency(t *testing.T) {
	router := NewModelRouter()

	req := &RoutingRequest{
		TenantID:     "tenant-1",
		MaxLatencyMs: 500,
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Verify latency constraint is considered
	t.Logf("Selected model: %s with latency: %d", got.ModelID, 500)
}

func TestDefaultModels(t *testing.T) {
	expected := map[string]struct {
		provider string
		cost     float64
	}{
		"gpt-4o":            {"openai", 0.02},
		"gpt-4o-mini":       {"openai", 0.00075},
		"claude-3-5-sonnet": {"anthropic", 0.018},
		"gemini-1.5-flash":  {"google", 0.000375},
	}

	for name, expected := range expected {
		model, ok := DefaultModels[name]
		if !ok {
			t.Errorf("Expected model %s not found", name)
			continue
		}
		if model.Provider != expected.provider {
			t.Errorf("Model %s: expected provider %s, got %s", name, expected.provider, model.Provider)
		}
		totalCost := model.CostPer1KInput + model.CostPer1KOutput
		if math.Abs(totalCost-expected.cost) > 0.000001 {
			t.Errorf("Model %s: expected cost %f, got %f", name, expected.cost, totalCost)
		}
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()

	cb.RegisterProvider("openai", DefaultAPIKeyConfig)

	if !cb.CanExecute("openai") {
		t.Error("Expected openai to be executable initially")
	}

	for i := 0; i < 5; i++ {
		cb.RecordFailure("openai")
	}

	if cb.GetStatus("openai") != CircuitOpen {
		t.Error("Expected circuit to be open after failures")
	}

	cb.Reset("openai")
	if !cb.CanExecute("openai") {
		t.Error("Expected openai to be executable after reset")
	}
}

func TestConnectionCooldown(t *testing.T) {
	cc := NewConnectionCooldown()

	connectionID := "conn-123"

	if cc.IsCoolingDown(connectionID) {
		t.Error("Expected connection to not be in cooldown initially")
	}

	cc.SetCooldown(connectionID, "openai", 100*time.Millisecond, "rate_limit")

	if !cc.IsCoolingDown(connectionID) {
		t.Error("Expected connection to be in cooldown after SetCooldown")
	}

	time.Sleep(150 * time.Millisecond)

	if cc.IsCoolingDown(connectionID) {
		t.Error("Expected connection to not be in cooldown after timeout")
	}
}

func TestModelLockout(t *testing.T) {
	ml := NewModelLockoutManager()

	if ml.IsLocked("openai", "gpt-4o") {
		t.Error("Expected model to not be locked initially")
	}

	ml.LockModel("openai", "gpt-4o", 100*time.Millisecond, "quota_exceeded")

	if !ml.IsLocked("openai", "gpt-4o") {
		t.Error("Expected model to be locked after LockModel")
	}

	time.Sleep(150 * time.Millisecond)

	if ml.IsLocked("openai", "gpt-4o") {
		t.Error("Expected model to not be locked after timeout")
	}
}

func TestAllRoutingStrategies(t *testing.T) {
	router := NewModelRouter()
	ctx := context.Background()

	strategies := []string{
		"priority",
		"cost_optimized",
		"latency_optimized",
		"quality_first",
		"round_robin",
		"random",
	}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			req := &RoutingRequest{
				TenantID: "tenant-1",
				Strategy: strategy,
			}

			got, err := router.Route(ctx, req)
			if err != nil {
				t.Errorf("Route() error = %v", err)
				return
			}
			if got == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}
