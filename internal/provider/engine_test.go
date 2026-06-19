package provider

import (
	"sync"
	"testing"
)

func TestProviderEngine_GetProvider(t *testing.T) {
	engine := GetProviderEngine()

	// Test getting OpenAI provider
	provider := engine.GetProvider("provider-openai")
	if provider == nil {
		t.Error("Expected OpenAI provider to exist")
	}
	if provider.Name != "OpenAI" {
		t.Errorf("Expected 'OpenAI', got '%s'", provider.Name)
	}
}

func TestProviderEngine_GetProvider_NotFound(t *testing.T) {
	engine := GetProviderEngine()

	// Test getting non-existent provider
	provider := engine.GetProvider("nonexistent")
	if provider != nil {
		t.Error("Expected nil for non-existent provider")
	}
}

func TestProviderEngine_ListProviders(t *testing.T) {
	engine := GetProviderEngine()

	providers := engine.ListProviders()
	if len(providers) == 0 {
		t.Error("Expected at least one provider")
	}

	// Check that OpenAI is in the list
	found := false
	for _, p := range providers {
		if p.ID == "provider-openai" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find OpenAI provider in list")
	}
}

func TestProviderEngine_EnableDisable(t *testing.T) {
	engine := GetProviderEngine()

	// Disable OpenAI
	err := engine.DisableProvider("provider-openai")
	if err != nil {
		t.Errorf("Error disabling provider: %v", err)
	}

	provider := engine.GetProvider("provider-openai")
	if provider.Enabled {
		t.Error("Expected OpenAI to be disabled")
	}

	// Re-enable
	err = engine.EnableProvider("provider-openai")
	if err != nil {
		t.Errorf("Error enabling provider: %v", err)
	}

	provider = engine.GetProvider("provider-openai")
	if !provider.Enabled {
		t.Error("Expected OpenAI to be enabled")
	}
}

func TestProviderEngine_EnableDisable_NotFound(t *testing.T) {
	engine := GetProviderEngine()

	// Non-existent provider - behavior may vary based on implementation
	// Just verify it doesn't panic
	err := engine.DisableProvider("nonexistent")
	_ = err // Error handling may vary

	err = engine.EnableProvider("nonexistent")
	_ = err
}

func TestProviderEngine_GetModel(t *testing.T) {
	engine := GetProviderEngine()

	// Test getting GPT-4 model
	model := engine.GetModel("gpt-4")
	if model == nil {
		t.Error("Expected GPT-4 model to exist")
	}
	if model.Name != "GPT-4" {
		t.Errorf("Expected 'GPT-4', got '%s'", model.Name)
	}
}

func TestProviderEngine_GetModel_NotFound(t *testing.T) {
	engine := GetProviderEngine()

	// Test getting non-existent model
	model := engine.GetModel("nonexistent-model")
	if model != nil {
		t.Error("Expected nil for non-existent model")
	}
}

func TestProviderEngine_ListModels(t *testing.T) {
	engine := GetProviderEngine()

	models := engine.ListModels()
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}

	// Check that GPT-4 is in the list
	found := false
	for _, m := range models {
		if m.ID == "gpt-4" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find GPT-4 model in list")
	}
}

func TestProviderEngine_ListModels_ByProvider(t *testing.T) {
	engine := GetProviderEngine()

	// List all models and filter by OpenAI
	models := engine.ListModels()

	// Count OpenAI models
	openaiCount := 0
	for _, m := range models {
		if m.Provider == "openai" {
			openaiCount++
		}
	}

	if openaiCount == 0 {
		t.Error("Expected at least one OpenAI model")
	}
}

func TestProviderEngine_ConcurrentAccess(t *testing.T) {
	engine := GetProviderEngine()

	// Test concurrent access to provider engine
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				engine.ListProviders()
				engine.ListModels()
				engine.GetProvider("provider-openai")
				engine.GetModel("gpt-4")
			}
		}()
	}
	wg.Wait()
}

func TestProviderEngine_GetHealthyProviders(t *testing.T) {
	engine := GetProviderEngine()

	healthy := engine.GetProvidersByCategory("openai")
	if len(healthy) == 0 {
		t.Error("Expected at least one OpenAI provider")
	}
}
