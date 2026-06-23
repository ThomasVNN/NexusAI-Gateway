package acp

import (
	"testing"
)

// TestBuiltInAgents tests that all 18 agents are registered
func TestBuiltInAgents(t *testing.T) {
	adapters := BuiltInAgents()

	expectedAgents := []string{
		"codex", "claude", "goose", "devin", "jules",
		"copilot", "tabnine", "cursor", "aider", "continue",
		"replit", "figma", "github-copilot", "amazon-codewhisperer",
		"codestral", "mistral-codestral", "wizardcoder", "phind",
	}

	if len(adapters) != 18 {
		t.Errorf("Expected 18 built-in agents, got %d", len(adapters))
	}

	agentMap := make(map[string]bool)
	for _, a := range adapters {
		agentMap[a.Name] = true
	}

	for _, name := range expectedAgents {
		if !agentMap[name] {
			t.Errorf("Expected agent %s not found", name)
		}
	}
}

// TestRegistry tests the ACP registry
func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	t.Run("BuiltInAdaptersRegistered", func(t *testing.T) {
		count := registry.Count()
		if count != 18 {
			t.Errorf("Expected 18 adapters, got %d", count)
		}
	})

	t.Run("GetAdapter", func(t *testing.T) {
		adapter, exists := registry.Get("codex")
		if !exists {
			t.Fatal("Expected codex adapter to exist")
		}
		if adapter.Binary != "codex" {
			t.Errorf("Expected binary 'codex', got '%s'", adapter.Binary)
		}
	})

	t.Run("ListAdapters", func(t *testing.T) {
		adapters := registry.List()
		if len(adapters) != 18 {
			t.Errorf("Expected 18 adapters, got %d", len(adapters))
		}
	})

	t.Run("RegisterDuplicate", func(t *testing.T) {
		err := registry.Register(&Adapter{Name: "codex"})
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	t.Run("GetByCapability", func(t *testing.T) {
		adapters := registry.GetByCapability("coding")
		if len(adapters) == 0 {
			t.Error("Expected adapters with 'coding' capability")
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		err := registry.Unregister("phind")
		if err != nil {
			t.Fatalf("Failed to unregister: %v", err)
		}

		_, exists := registry.Get("phind")
		if exists {
			t.Error("Adapter should not exist after unregister")
		}
	})
}

// TestAdapterConfig tests adapter configuration
func TestAdapterConfig(t *testing.T) {
	adapter := BuiltInAgents()[0]

	if adapter.Config.Timeout == 0 {
		t.Error("Expected non-zero timeout")
	}
	if adapter.Config.RetryCount == 0 {
		t.Error("Expected non-zero retry count")
	}
	if len(adapter.Config.Capabilities) == 0 {
		t.Error("Expected capabilities to be set")
	}
}

// TestAgentManager tests agent management
func TestAgentManager(t *testing.T) {
	registry := NewRegistry()
	adapter, _ := registry.Get("codex")
	manager := NewAgentManager(adapter)

	t.Run("Spawn", func(t *testing.T) {
		info, err := manager.Spawn()
		if err != nil {
			t.Fatalf("Failed to spawn: %v", err)
		}
		if info.Name != "codex" {
			t.Errorf("Expected name 'codex', got '%s'", info.Name)
		}
		if info.Status != "running" {
			t.Errorf("Expected status 'running', got '%s'", info.Status)
		}
	})

	t.Run("ListAgents", func(t *testing.T) {
		agents := manager.List()
		if len(agents) != 1 {
			t.Errorf("Expected 1 agent, got %d", len(agents))
		}
	})

	t.Run("GetAgent", func(t *testing.T) {
		agents := manager.List()
		info, exists := manager.Get(agents[0].ID)
		if !exists {
			t.Fatal("Agent not found")
		}
		if info.ID != agents[0].ID {
			t.Errorf("Expected ID %s, got %s", agents[0].ID, info.ID)
		}
	})

	t.Run("TerminateAgent", func(t *testing.T) {
		agents := manager.List()
		err := manager.Terminate(agents[0].ID)
		if err != nil {
			t.Fatalf("Failed to terminate: %v", err)
		}

		_, exists := manager.Get(agents[0].ID)
		if exists {
			t.Error("Agent should not exist after terminate")
		}
	})
}
