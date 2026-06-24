package hooks

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewHookRegistry(t *testing.T) {
	registry := NewHookRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.hooks == nil {
		t.Error("expected hooks map to be initialized")
	}
}

func TestHookRegistryRegister(t *testing.T) {
	registry := NewHookRegistry()
	
	hook := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		return nil
	}
	
	registry.Register(AgentClaudeCode, EventCommandStart, hook)
	
	ctx := context.Background()
	data := map[string]interface{}{"test": "data"}
	errors := registry.Trigger(ctx, AgentClaudeCode, EventCommandStart, data)
	
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
}

func TestHookRegistryMultipleHooks(t *testing.T) {
	registry := NewHookRegistry()
	
	callCount := 0
	hook1 := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		callCount++
		return nil
	}
	hook2 := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		callCount++
		return nil
	}
	
	registry.Register(AgentCursor, EventCommandEnd, hook1)
	registry.Register(AgentCursor, EventCommandEnd, hook2)
	
	ctx := context.Background()
	data := map[string]interface{}{}
	registry.Trigger(ctx, AgentCursor, EventCommandEnd, data)
	
	if callCount != 2 {
		t.Errorf("expected 2 hook calls, got %d", callCount)
	}
}

func TestHookRegistryErrorPropagation(t *testing.T) {
	registry := NewHookRegistry()
	
	expectedErr := errors.New("test error")
	hook := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		return expectedErr
	}
	
	registry.Register(AgentCopilot, EventError, hook)
	
	ctx := context.Background()
	data := map[string]interface{}{}
	errs := registry.Trigger(ctx, AgentCopilot, EventError, data)
	
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0] != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, errs[0])
	}
}

func TestHookRegistryNoHooks(t *testing.T) {
	registry := NewHookRegistry()
	
	ctx := context.Background()
	data := map[string]interface{}{}
	errs := registry.Trigger(ctx, AgentCline, EventCommandStart, data)
	
	if errs != nil {
		t.Errorf("expected nil errors, got %v", errs)
	}
}

func TestNewAgentHookSet(t *testing.T) {
	hookSet := NewAgentHookSet(AgentCodex)
	if hookSet == nil {
		t.Fatal("expected non-nil hook set")
	}
	if hookSet.Agent != AgentCodex {
		t.Errorf("expected agent %s, got %s", AgentCodex, hookSet.Agent)
	}
}

func TestAgentHookSetAddHooks(t *testing.T) {
	hookSet := NewAgentHookSet(AgentWindsurf)
	
	startCalled := false
	startHook := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		startCalled = true
		return nil
	}
	
	hookSet.AddStartHook(startHook)
	
	ctx := context.Background()
	errs := hookSet.TriggerStart(ctx, map[string]interface{}{})
	
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if !startCalled {
		t.Error("expected start hook to be called")
	}
}

func TestAgentHookSetTriggerError(t *testing.T) {
	hookSet := NewAgentHookSet(AgentClaudeCode)
	
	errCalled := false
	errHook := func(ctx context.Context, event HookEvent, data map[string]interface{}) error {
		errCalled = true
		return nil
	}
	
	hookSet.AddErrorHook(errHook)
	
	ctx := context.Background()
	errs := hookSet.TriggerError(ctx, map[string]interface{}{})
	
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if !errCalled {
		t.Error("expected error hook to be called")
	}
}

func TestDefaultHookConfig(t *testing.T) {
	config := DefaultHookConfig()
	
	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected Timeout to be 30s, got %v", config.Timeout)
	}
	if config.RetryCount != 3 {
		t.Errorf("expected RetryCount to be 3, got %d", config.RetryCount)
	}
	if config.Async {
		t.Error("expected Async to be false")
	}
}

func TestClaudeCodeHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewClaudeCodeHooks(config)
	
	ctx := context.Background()
	
	// Test PreCommandHook
	err := hooks.PreCommandHook(ctx, "ls -la")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Test PostCommandHook
	err = hooks.PostCommandHook(ctx, "ls -la", 0, "output")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Test ErrorHook
	err = hooks.ErrorHook(ctx, errors.New("test error"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Verify config
	if hooks.GetConfig().Enabled != config.Enabled {
		t.Error("config mismatch")
	}
}

func TestClaudeCodeHooksDisabled(t *testing.T) {
	config := HookConfig{Enabled: false}
	hooks := NewClaudeCodeHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCopilotHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewCopilotHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "git commit")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.PostCommandHook(ctx, "git commit", 0, "success")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.ErrorHook(ctx, errors.New("copilot error"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCursorHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewCursorHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "cursor command")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.PostCommandHook(ctx, "cursor command", 0, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.ErrorHook(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClineHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewClineHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "cline task")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.PostCommandHook(ctx, "cline task", 1, "failed")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCodexHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewCodexHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "codex analyze")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.PostCommandHook(ctx, "codex analyze", 0, "analyzed")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWindsurfHooks(t *testing.T) {
	config := DefaultHookConfig()
	hooks := NewWindsurfHooks(config)
	
	ctx := context.Background()
	
	err := hooks.PreCommandHook(ctx, "windsurf deploy")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	err = hooks.PostCommandHook(ctx, "windsurf deploy", 0, "deployed")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHookEventConstants(t *testing.T) {
	if EventCommandStart != "command.start" {
		t.Errorf("unexpected EventCommandStart: %s", EventCommandStart)
	}
	if EventCommandEnd != "command.end" {
		t.Errorf("unexpected EventCommandEnd: %s", EventCommandEnd)
	}
	if EventError != "error" {
		t.Errorf("unexpected EventError: %s", EventError)
	}
	if EventTokenUsage != "token.usage" {
		t.Errorf("unexpected EventTokenUsage: %s", EventTokenUsage)
	}
	if EventFileChange != "file.change" {
		t.Errorf("unexpected EventFileChange: %s", EventFileChange)
	}
}

func TestAgentTypeConstants(t *testing.T) {
	if AgentClaudeCode != "claude-code" {
		t.Errorf("unexpected AgentClaudeCode: %s", AgentClaudeCode)
	}
	if AgentCopilot != "copilot" {
		t.Errorf("unexpected AgentCopilot: %s", AgentCopilot)
	}
	if AgentCursor != "cursor" {
		t.Errorf("unexpected AgentCursor: %s", AgentCursor)
	}
	if AgentCline != "cline" {
		t.Errorf("unexpected AgentCline: %s", AgentCline)
	}
	if AgentCodex != "codex" {
		t.Errorf("unexpected AgentCodex: %s", AgentCodex)
	}
	if AgentWindsurf != "windsurf" {
		t.Errorf("unexpected AgentWindsurf: %s", AgentWindsurf)
	}
}

func TestHookConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  HookConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultHookConfig(),
			wantErr: false,
		},
		{
			name:    "negative timeout",
			config:  HookConfig{Timeout: -1},
			wantErr: true,
		},
		{
			name:    "negative retry count",
			config:  HookConfig{RetryCount: -1},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHookConfigMerge(t *testing.T) {
	base := DefaultHookConfig()
	override := HookConfig{
		Enabled:    true, // explicitly set to true to override
		Timeout:    60 * time.Second,
		RetryCount: 5,
		Async:      true,
	}
	
	merged := base.Merge(override)
	
	if merged.Enabled != true {
		t.Error("Enabled should be true after merge")
	}
	if merged.Timeout != 60*time.Second {
		t.Error("Timeout should be 60s after merge")
	}
	if merged.RetryCount != 5 {
		t.Error("RetryCount should be 5 after merge")
	}
	if !merged.Async {
		t.Error("Async should be true after merge")
	}
}

func TestDefaultAgentConfig(t *testing.T) {
	config := DefaultAgentConfig(AgentCursor)
	
	if config.Agent != AgentCursor {
		t.Errorf("expected Agent %s, got %s", AgentCursor, config.Agent)
	}
	if !config.Enabled {
		t.Error("expected Enabled to be true")
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected Timeout to be 30s, got %v", config.Timeout)
	}
	if config.RetryCount != 3 {
		t.Errorf("expected RetryCount to be 3, got %d", config.RetryCount)
	}
	if config.MaxLogSize != 10*1024*1024 {
		t.Errorf("expected MaxLogSize to be 10MB, got %d", config.MaxLogSize)
	}
}

func TestEnvVars(t *testing.T) {
	envs := EnvVars()
	
	expectedVars := []string{
		"RTK_HOOKS_ENABLED",
		"RTK_HOOKS_TIMEOUT",
		"RTK_HOOKS_RETRY_COUNT",
		"RTK_HOOKS_ASYNC",
	}
	
	for _, varName := range expectedVars {
		if _, ok := envs[varName]; !ok {
			t.Errorf("expected %s to be in EnvVars", varName)
		}
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "Timeout",
		Message: "timeout cannot be negative",
	}
	
	expected := "invalid config: Timeout - timeout cannot be negative"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}
