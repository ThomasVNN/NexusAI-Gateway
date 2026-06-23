package acp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Adapter represents an ACP agent adapter
type Adapter struct {
	Name       string   `json:"name"`
	Binary     string   `json:"binary"`
	VersionCmd string   `json:"version_cmd"`
	SpawnArgs  []string `json:"spawn_args,omitempty"`
	Config     AdapterConfig `json:"config,omitempty"`
}

// AdapterConfig contains adapter configuration
type AdapterConfig struct {
	Timeout       time.Duration `json:"timeout"`
	RetryCount   int           `json:"retry_count"`
	Capabilities []string      `json:"capabilities"`
}

// Registry manages ACP agent adapters
type Registry struct {
	mu      sync.RWMutex
	adapters map[string]*Adapter
	logger   *slog.Logger
}

// NewRegistry creates a new ACP registry
func NewRegistry() *Registry {
	r := &Registry{
		adapters: make(map[string]*Adapter),
		logger:   slog.Default(),
	}

	r.registerBuiltInAdapters()
	return r
}

// BuiltInAgents returns the list of 18 built-in agent adapters
func BuiltInAgents() []*Adapter {
	return []*Adapter{
		{
			Name:       "codex",
			Binary:     "codex",
			VersionCmd: "codex --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"code", "debug", "refactor"},
			},
		},
		{
			Name:       "claude",
			Binary:     "claude",
			VersionCmd: "claude --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"reasoning", "writing", "analysis"},
			},
		},
		{
			Name:       "goose",
			Binary:     "goose",
			VersionCmd: "goose --version",
			Config: AdapterConfig{
				Timeout:       45 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"automation", "testing"},
			},
		},
		{
			Name:       "devin",
			Binary:     "devin",
			VersionCmd: "devin --version",
			Config: AdapterConfig{
				Timeout:       90 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"coding", "planning", "debugging"},
			},
		},
		{
			Name:       "jules",
			Binary:     "jules",
			VersionCmd: "jules --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"code_review", "refactoring"},
			},
		},
		{
			Name:       "copilot",
			Binary:     "copilot",
			VersionCmd: "copilot --version",
			Config: AdapterConfig{
				Timeout:       30 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"completion", "suggestion"},
			},
		},
		{
			Name:       "tabnine",
			Binary:     "tabnine",
			VersionCmd: "tabnine --version",
			Config: AdapterConfig{
				Timeout:       15 * time.Second,
				RetryCount:   1,
				Capabilities: []string{"completion", "refactor"},
			},
		},
		{
			Name:       "cursor",
			Binary:     "cursor",
			VersionCmd: "cursor --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"coding", "reasoning", "editing"},
			},
		},
		{
			Name:       "aider",
			Binary:     "aider",
			VersionCmd: "aider --version",
			Config: AdapterConfig{
				Timeout:       45 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"pair_programming", "editing"},
			},
		},
		{
			Name:       "continue",
			Binary:     "continue",
			VersionCmd: "continue --version",
			Config: AdapterConfig{
				Timeout:       45 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"context_aware", "completion"},
			},
		},
		{
			Name:       "replit",
			Binary:     "replit",
			VersionCmd: "replit --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"coding", "deployment", "testing"},
			},
		},
		{
			Name:       "figma",
			Binary:     "figma-agent",
			VersionCmd: "figma-agent --version",
			Config: AdapterConfig{
				Timeout:       30 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"design", "prototyping"},
			},
		},
		{
			Name:       "github-copilot",
			Binary:     "github-copilot",
			VersionCmd: "github-copilot --version",
			Config: AdapterConfig{
				Timeout:       30 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"completion", "suggestion", "refactor"},
			},
		},
		{
			Name:       "amazon-codewhisperer",
			Binary:     "codewhisperer",
			VersionCmd: "codewhisperer --version",
			Config: AdapterConfig{
				Timeout:       30 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"completion", "security_scan"},
			},
		},
		{
			Name:       "codestral",
			Binary:     "codestral",
			VersionCmd: "codestral --version",
			Config: AdapterConfig{
				Timeout:       45 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"coding", "completion", "refactor"},
			},
		},
		{
			Name:       "mistral-codestral",
			Binary:     "mistral-codestral",
			VersionCmd: "mistral-codestral --version",
			Config: AdapterConfig{
				Timeout:       45 * time.Second,
				RetryCount:   3,
				Capabilities: []string{"coding", "reasoning"},
			},
		},
		{
			Name:       "wizardcoder",
			Binary:     "wizardcoder",
			VersionCmd: "wizardcoder --version",
			Config: AdapterConfig{
				Timeout:       60 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"coding", "instruction_following"},
			},
		},
		{
			Name:       "phind",
			Binary:     "phind",
			VersionCmd: "phind --version",
			Config: AdapterConfig{
				Timeout:       30 * time.Second,
				RetryCount:   2,
				Capabilities: []string{"search", "coding_assistance"},
			},
		},
	}
}

// registerBuiltInAdapters registers the built-in adapters
func (r *Registry) registerBuiltInAdapters() {
	for _, adapter := range BuiltInAgents() {
		r.adapters[adapter.Name] = adapter
	}
}

// Register registers a new adapter
func (r *Registry) Register(adapter *Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[adapter.Name]; exists {
		return fmt.Errorf("adapter %s already registered", adapter.Name)
	}

	r.adapters[adapter.Name] = adapter
	r.logger.Info("Registered ACP adapter", slog.String("name", adapter.Name))

	return nil
}

// Get retrieves an adapter by name
func (r *Registry) Get(name string) (*Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	return adapter, exists
}

// List returns all registered adapters
func (r *Registry) List() []*Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]*Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}

// Unregister removes an adapter
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[name]; !exists {
		return fmt.Errorf("adapter %s not found", name)
	}

	delete(r.adapters, name)
	r.logger.Info("Unregistered ACP adapter", slog.String("name", name))

	return nil
}

// Count returns the number of registered adapters
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// GetByCapability returns adapters with a specific capability
func (r *Registry) GetByCapability(capability string) []*Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var adapters []*Adapter
	for _, adapter := range r.adapters {
		for _, cap := range adapter.Config.Capabilities {
			if cap == capability {
				adapters = append(adapters, adapter)
				break
			}
		}
	}
	return adapters
}

// AgentInfo represents information about a running agent
type AgentInfo struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	StartedAt  time.Time             `json:"started_at"`
	TasksCount int                   `json:"tasks_count"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AgentManager manages running agent instances
type AgentManager struct {
	mu      sync.RWMutex
	agents  map[string]*AgentInfo
	adapter *Adapter
	logger  *slog.Logger
}

// NewAgentManager creates a new agent manager
func NewAgentManager(adapter *Adapter) *AgentManager {
	return &AgentManager{
		agents:  make(map[string]*AgentInfo),
		adapter: adapter,
		logger:  slog.Default(),
	}
}

// Spawn starts a new agent instance
func (m *AgentManager) Spawn() (*AgentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := &AgentInfo{
		ID:         fmt.Sprintf("%s-%d", m.adapter.Name, time.Now().Unix()),
		Name:       m.adapter.Name,
		Status:     "running",
		StartedAt:  time.Now(),
		TasksCount: 0,
		Metadata:   make(map[string]interface{}),
	}

	m.agents[info.ID] = info
	m.logger.Info("Spawned agent", slog.String("agent_id", info.ID), slog.String("name", info.Name))

	return info, nil
}

// Get retrieves an agent by ID
func (m *AgentManager) Get(id string) (*AgentInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, exists := m.agents[id]
	return info, exists
}

// List returns all running agents
func (m *AgentManager) List() []*AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*AgentInfo, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

// Terminate stops an agent
func (m *AgentManager) Terminate(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[id]
	if !exists {
		return fmt.Errorf("agent %s not found", id)
	}

	agent.Status = "terminated"
	delete(m.agents, id)
	m.logger.Info("Terminated agent", slog.String("agent_id", id))

	return nil
}

// ToJSON converts registry to JSON
func (r *Registry) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r.adapters, "", "  ")
}

// FromJSON creates registry from JSON
func FromJSON(data []byte) (*Registry, error) {
	var adapters map[string]*Adapter
	if err := json.Unmarshal(data, &adapters); err != nil {
		return nil, err
	}

	r := &Registry{
		adapters: adapters,
		logger:   slog.Default(),
	}

	return r, nil
}
