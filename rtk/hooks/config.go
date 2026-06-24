package hooks

import (
	"os"
	"strconv"
	"time"
)

// ConfigLoader loads hook configuration from environment variables
type ConfigLoader struct{}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadFromEnv loads configuration from environment variables
func (l *ConfigLoader) LoadFromEnv() HookConfig {
	config := DefaultHookConfig()

	if enabled := os.Getenv("RTK_HOOKS_ENABLED"); enabled != "" {
		config.Enabled = enabled == "true" || enabled == "1"
	}

	if timeout := os.Getenv("RTK_HOOKS_TIMEOUT"); timeout != "" {
		if seconds, err := strconv.Atoi(timeout); err == nil {
			config.Timeout = time.Duration(seconds) * time.Second
		}
	}

	if retry := os.Getenv("RTK_HOOKS_RETRY_COUNT"); retry != "" {
		if count, err := strconv.Atoi(retry); err == nil {
			config.RetryCount = count
		}
	}

	if async := os.Getenv("RTK_HOOKS_ASYNC"); async != "" {
		config.Async = async == "true" || async == "1"
	}

	return config
}

// EnvConfigPrefix returns the environment variable prefix for hooks
const EnvConfigPrefix = "RTK_HOOKS"

// EnvVars returns all environment variables used for hook configuration
func EnvVars() map[string]string {
	return map[string]string{
		"RTK_HOOKS_ENABLED":     "Enable/disable hooks (true/false)",
		"RTK_HOOKS_TIMEOUT":     "Hook timeout in seconds",
		"RTK_HOOKS_RETRY_COUNT": "Number of retries for failed hooks",
		"RTK_HOOKS_ASYNC":       "Run hooks asynchronously (true/false)",
	}
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	Agent       AgentType
	Enabled     bool
	Timeout     time.Duration
	RetryCount  int
	Async       bool
	HooksDir    string
	MaxLogSize  int64
}

// DefaultAgentConfig returns default configuration for an agent
func DefaultAgentConfig(agent AgentType) AgentConfig {
	return AgentConfig{
		Agent:       agent,
		Enabled:     true,
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Async:       false,
		HooksDir:    "/var/lib/nexusai/hooks/" + string(agent),
		MaxLogSize:  10 * 1024 * 1024, // 10MB
	}
}

// LoadAgentConfigFromEnv loads agent-specific configuration from environment
func LoadAgentConfigFromEnv(agent AgentType) AgentConfig {
	config := DefaultAgentConfig(agent)
	prefix := EnvConfigPrefix + "_" + envAgentName(agent) + "_"

	if enabled := os.Getenv(prefix + "ENABLED"); enabled != "" {
		config.Enabled = enabled == "true" || enabled == "1"
	}

	if timeout := os.Getenv(prefix + "TIMEOUT"); timeout != "" {
		if seconds, err := strconv.Atoi(timeout); err == nil {
			config.Timeout = time.Duration(seconds) * time.Second
		}
	}

	if retry := os.Getenv(prefix + "RETRY_COUNT"); retry != "" {
		if count, err := strconv.Atoi(retry); err == nil {
			config.RetryCount = count
		}
	}

	if async := os.Getenv(prefix + "ASYNC"); async != "" {
		config.Async = async == "true" || async == "1"
	}

	if hooksDir := os.Getenv(prefix + "HOOKS_DIR"); hooksDir != "" {
		config.HooksDir = hooksDir
	}

	if maxLog := os.Getenv(prefix + "MAX_LOG_SIZE"); maxLog != "" {
		if size, err := strconv.ParseInt(maxLog, 10, 64); err == nil {
			config.MaxLogSize = size
		}
	}

	return config
}

// envAgentName converts agent type to environment variable name
func envAgentName(agent AgentType) string {
	switch agent {
	case AgentClaudeCode:
		return "CLAUDE_CODE"
	case AgentCopilot:
		return "COPILOT"
	case AgentCursor:
		return "CURSOR"
	case AgentCline:
		return "CLINE"
	case AgentCodex:
		return "CODEX"
	case AgentWindsurf:
		return "WINDSURF"
	default:
		return "UNKNOWN"
	}
}

// Validate checks if the configuration is valid
func (c HookConfig) Validate() error {
	if c.Timeout < 0 {
		return &ConfigError{Field: "Timeout", Message: "timeout cannot be negative"}
	}
	if c.RetryCount < 0 {
		return &ConfigError{Field: "RetryCount", Message: "retry count cannot be negative"}
	}
	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "invalid config: " + e.Field + " - " + e.Message
}

// Merge combines two configurations, with other taking precedence
func (c HookConfig) Merge(other HookConfig) HookConfig {
	if other.Enabled {
		c.Enabled = other.Enabled
	}
	if other.Timeout != 0 {
		c.Timeout = other.Timeout
	}
	if other.RetryCount != 0 {
		c.RetryCount = other.RetryCount
	}
	if other.Async {
		c.Async = other.Async
	}
	return c
}
