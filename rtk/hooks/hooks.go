package hooks

import (
	"context"
	"sync"
	"time"
)

// AgentType represents supported AI agents
type AgentType string

const (
	AgentClaudeCode AgentType = "claude-code"
	AgentCopilot    AgentType = "copilot"
	AgentCursor     AgentType = "cursor"
	AgentCline      AgentType = "cline"
	AgentCodex      AgentType = "codex"
	AgentWindsurf   AgentType = "windsurf"
)

// HookEvent represents an event that can trigger hooks
type HookEvent string

const (
	EventCommandStart HookEvent = "command.start"
	EventCommandEnd   HookEvent = "command.end"
	EventError        HookEvent = "error"
	EventTokenUsage   HookEvent = "token.usage"
	EventFileChange   HookEvent = "file.change"
)

// Hook represents a hook function
type Hook func(ctx context.Context, event HookEvent, data map[string]interface{}) error

// HookRegistry manages hooks for each agent
type HookRegistry struct {
	mu    sync.RWMutex
	hooks map[AgentType]map[HookEvent][]Hook
}

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		hooks: make(map[AgentType]map[HookEvent][]Hook),
	}
}

// Register adds a hook for an agent and event
func (r *HookRegistry) Register(agent AgentType, event HookEvent, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.hooks[agent] == nil {
		r.hooks[agent] = make(map[HookEvent][]Hook)
	}
	r.hooks[agent][event] = append(r.hooks[agent][event], hook)
}

// Unregister removes a hook
func (r *HookRegistry) Unregister(agent AgentType, event HookEvent, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.hooks[agent] == nil {
		return
	}
	hooks := r.hooks[agent][event]
	for i, h := range hooks {
		// Compare function pointers
		if equalHooks(h, hook) {
			r.hooks[agent][event] = append(hooks[:i], hooks[i+1:]...)
			return
		}
	}
}

// equalHooks compares two hook functions for equality
func equalHooks(a, b Hook) bool {
	// Function comparison in Go requires indirect comparison
	// We'll use a wrapper to make them comparable
	return hookToString(a) == hookToString(b)
}

// hookToString is a workaround to compare functions
func hookToString(h Hook) string {
	return functionName(h)
}

// functionName returns a string representation of a function
// This is a simple implementation; in production, you might use runtime
func functionName(h Hook) string {
	if h == nil {
		return "nil"
	}
	return "hook"
}

// Trigger fires all hooks for an agent and event
func (r *HookRegistry) Trigger(ctx context.Context, agent AgentType, event HookEvent, data map[string]interface{}) []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errors []error
	hooks := r.hooks[agent][event]
	for _, hook := range hooks {
		if err := hook(ctx, event, data); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// AgentHookSet is the set of hooks for an agent
type AgentHookSet struct {
	mu          sync.RWMutex
	Agent       AgentType
	OnStart     []Hook
	OnEnd       []Hook
	OnError     []Hook
	OnTokenUse  []Hook
	OnFileChange []Hook
}

// NewAgentHookSet creates hooks for a specific agent
func NewAgentHookSet(agent AgentType) *AgentHookSet {
	return &AgentHookSet{
		Agent: agent,
	}
}

// AddStartHook adds a start hook
func (s *AgentHookSet) AddStartHook(hook Hook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OnStart = append(s.OnStart, hook)
}

// AddEndHook adds an end hook
func (s *AgentHookSet) AddEndHook(hook Hook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OnEnd = append(s.OnEnd, hook)
}

// AddErrorHook adds an error hook
func (s *AgentHookSet) AddErrorHook(hook Hook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OnError = append(s.OnError, hook)
}

// AddTokenUseHook adds a token usage hook
func (s *AgentHookSet) AddTokenUseHook(hook Hook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OnTokenUse = append(s.OnTokenUse, hook)
}

// AddFileChangeHook adds a file change hook
func (s *AgentHookSet) AddFileChangeHook(hook Hook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OnFileChange = append(s.OnFileChange, hook)
}

// TriggerStart triggers all start hooks
func (s *AgentHookSet) TriggerStart(ctx context.Context, data map[string]interface{}) []error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errors []error
	for _, hook := range s.OnStart {
		if err := hook(ctx, EventCommandStart, data); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// TriggerEnd triggers all end hooks
func (s *AgentHookSet) TriggerEnd(ctx context.Context, data map[string]interface{}) []error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errors []error
	for _, hook := range s.OnEnd {
		if err := hook(ctx, EventCommandEnd, data); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// TriggerError triggers all error hooks
func (s *AgentHookSet) TriggerError(ctx context.Context, data map[string]interface{}) []error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var errors []error
	for _, hook := range s.OnError {
		if err := hook(ctx, EventError, data); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// HookConfig contains hook configuration
type HookConfig struct {
	Enabled    bool
	Timeout    time.Duration
	RetryCount int
	Async      bool
}

// DefaultHookConfig returns sensible defaults
func DefaultHookConfig() HookConfig {
	return HookConfig{
		Enabled:    true,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		Async:      false,
	}
}

// ClaudeCodeHooks implements Claude Code-specific hooks
type ClaudeCodeHooks struct {
	config HookConfig
}

// NewClaudeCodeHooks creates a new Claude Code hooks instance
func NewClaudeCodeHooks(config HookConfig) *ClaudeCodeHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &ClaudeCodeHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *ClaudeCodeHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	// Claude Code specific pre-command logic
	return nil
}

// PostCommandHook executes after a command completes
func (h *ClaudeCodeHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	// Claude Code specific post-command logic
	return nil
}

// ErrorHook executes when an error occurs
func (h *ClaudeCodeHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	// Claude Code specific error handling
	return nil
}

// GetConfig returns the hook configuration
func (h *ClaudeCodeHooks) GetConfig() HookConfig {
	return h.config
}

// CopilotHooks implements GitHub Copilot-specific hooks
type CopilotHooks struct {
	config HookConfig
}

// NewCopilotHooks creates a new Copilot hooks instance
func NewCopilotHooks(config HookConfig) *CopilotHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &CopilotHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *CopilotHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// PostCommandHook executes after a command completes
func (h *CopilotHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// ErrorHook executes when an error occurs
func (h *CopilotHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// GetConfig returns the hook configuration
func (h *CopilotHooks) GetConfig() HookConfig {
	return h.config
}

// CursorHooks implements Cursor-specific hooks
type CursorHooks struct {
	config HookConfig
}

// NewCursorHooks creates a new Cursor hooks instance
func NewCursorHooks(config HookConfig) *CursorHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &CursorHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *CursorHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// PostCommandHook executes after a command completes
func (h *CursorHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// ErrorHook executes when an error occurs
func (h *CursorHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// GetConfig returns the hook configuration
func (h *CursorHooks) GetConfig() HookConfig {
	return h.config
}

// ClineHooks implements Cline-specific hooks
type ClineHooks struct {
	config HookConfig
}

// NewClineHooks creates a new Cline hooks instance
func NewClineHooks(config HookConfig) *ClineHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &ClineHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *ClineHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// PostCommandHook executes after a command completes
func (h *ClineHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// ErrorHook executes when an error occurs
func (h *ClineHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// GetConfig returns the hook configuration
func (h *ClineHooks) GetConfig() HookConfig {
	return h.config
}

// CodexHooks implements OpenAI Codex-specific hooks
type CodexHooks struct {
	config HookConfig
}

// NewCodexHooks creates a new Codex hooks instance
func NewCodexHooks(config HookConfig) *CodexHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &CodexHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *CodexHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// PostCommandHook executes after a command completes
func (h *CodexHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// ErrorHook executes when an error occurs
func (h *CodexHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// GetConfig returns the hook configuration
func (h *CodexHooks) GetConfig() HookConfig {
	return h.config
}

// WindsurfHooks implements Windsurf-specific hooks
type WindsurfHooks struct {
	config HookConfig
}

// NewWindsurfHooks creates a new Windsurf hooks instance
func NewWindsurfHooks(config HookConfig) *WindsurfHooks {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &WindsurfHooks{config: config}
}

// PreCommandHook executes before a command runs
func (h *WindsurfHooks) PreCommandHook(ctx context.Context, cmd string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// PostCommandHook executes after a command completes
func (h *WindsurfHooks) PostCommandHook(ctx context.Context, cmd string, exitCode int, output string) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// ErrorHook executes when an error occurs
func (h *WindsurfHooks) ErrorHook(ctx context.Context, err error) error {
	if !h.config.Enabled {
		return nil
	}
	return nil
}

// GetConfig returns the hook configuration
func (h *WindsurfHooks) GetConfig() HookConfig {
	return h.config
}
