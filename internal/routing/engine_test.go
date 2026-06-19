package routing

import (
	"context"
	"testing"
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
			name: "fallback to default when model not found",
			req: &RoutingRequest{
				RequestedModel: "unknown-model",
				TenantID:       "tenant-1",
			},
			wantModel: "gpt-4o-mini", // cheapest default
			wantErr:   false,
		},
		{
			name: "route by capability - vision",
			req: &RoutingRequest{
				TenantID:             "tenant-1",
				RequiredCapabilities: []string{"vision"},
			},
			wantModel: "gemini-1.5-pro", // cheapest vision-capable model (cost: 0.00625)
			wantErr:   false,
		},
		{
			name: "route by capability - function_calling",
			req: &RoutingRequest{
				TenantID:             "tenant-1",
				RequiredCapabilities: []string{"function_calling"},
			},
			wantModel: "gemini-1.5-flash", // cheapest function_calling model (cost: 0.000375)
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
			if got != nil && got.ModelName != tt.wantModel {
				t.Errorf("Route() model = %v, want %v", got.ModelName, tt.wantModel)
			}
		})
	}
}

func TestModelRouter_SelectByCost(t *testing.T) {
	router := NewModelRouter()

	// All models with function_calling
	req := &RoutingRequest{
		TenantID:             "tenant-1",
		RequiredCapabilities: []string{"function_calling"},
		Strategy:             StrategyCostOptimized,
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Should select cheapest model with function_calling
	if got.ModelName != "gemini-1.5-flash" {
		t.Errorf("Expected cheapest model gemini-1.5-flash, got %s", got.ModelName)
	}
}

func TestModelRouter_SelectByLatency(t *testing.T) {
	router := NewModelRouter()

	req := &RoutingRequest{
		TenantID: "tenant-1",
		Strategy: StrategyLatencyOptimized,
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Should select fastest model
	if got.ModelName != "gemini-1.5-flash" {
		t.Errorf("Expected fastest model gemini-1.5-flash, got %s", got.ModelName)
	}
}

func TestModelRouter_TenantStrategy(t *testing.T) {
	router := NewModelRouter()

	// Set tenant to use latency optimization
	router.SetTenantStrategy("premium-tenant", StrategyLatencyOptimized)

	req := &RoutingRequest{
		TenantID: "premium-tenant",
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Should respect tenant strategy
	if got.ModelName != "gemini-1.5-flash" {
		t.Errorf("Expected fastest model, got %s", got.ModelName)
	}
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
	router := NewModelRouter()

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
			got := router.hasCapabilities(model, tt.required)
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
		MaxLatencyMs: 500, // Only models under 500ms
	}

	ctx := context.Background()
	got, err := router.Route(ctx, req)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Should exclude models with higher latency
	if got.ModelName != "gemini-1.5-flash" {
		// gemini-1.5-flash has 300ms latency, should be the only one fitting
		t.Errorf("Expected low-latency model, got %s", got.ModelName)
	}
}

func TestDefaultModels(t *testing.T) {
	// Verify default models are configured correctly
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
		if totalCost < expected.cost-0.000001 || totalCost > expected.cost+0.000001 {
			t.Errorf("Model %s: expected cost %f, got %f", name, expected.cost, totalCost)
		}
	}
}
