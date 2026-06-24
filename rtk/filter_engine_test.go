package rtk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexer_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []*Token
	}{
		{
			name:  "simple command",
			input: "git status",
			expected: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "status"},
			},
		},
		{
			name:  "command with flag",
			input: "git clone --depth 1 repo",
			expected: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "clone"},
				{Type: TokenFlag, Value: "--depth"},
				{Type: TokenArgument, Value: "1"},
				{Type: TokenArgument, Value: "repo"},
			},
		},
		{
			name:  "command with short flag",
			input: "git push -u origin main",
			expected: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "push"},
				{Type: TokenFlag, Value: "-u"},
				{Type: TokenArgument, Value: "origin"},
				{Type: TokenArgument, Value: "main"},
			},
		},
		{
			name:  "empty input",
			input: "",
			expected: nil,
		},
		{
			name:  "whitespace only",
			input: "   ",
			expected: nil,
		},
		{
			name:  "multiple flags",
			input: "npm install --save-dev --prefer-offline",
			expected: []*Token{
				{Type: TokenCommand, Value: "npm"},
				{Type: TokenArgument, Value: "install"},
				{Type: TokenFlag, Value: "--save-dev"},
				{Type: TokenFlag, Value: "--prefer-offline"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestLexer_NextToken(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectToken *Token
		expectError bool
	}{
		{
			name:        "command token",
			input:       "git",
			expectToken: &Token{Type: TokenCommand, Value: "git"},
		},
		{
			name:        "flag token",
			input:       "--depth",
			expectToken: &Token{Type: TokenFlag, Value: "--depth"},
		},
		{
			name:        "empty input",
			input:       "",
			expectToken: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token, err := lexer.NextToken()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectToken, token)
			}
		})
	}
}

func TestDetectCommandType(t *testing.T) {
	tests := []struct {
		command string
		expected string
	}{
		{"git clone https://github.com/repo", "git_clone"},
		{"git push origin main", "git_push"},
		{"git pull origin main", "git_pull"},
		{"git status", "git_status"},
		{"git log -n 10", "git_log"},
		{"cargo build", "cargo_build"},
		{"cargo test", "cargo_test"},
		{"cargo clippy", "cargo_clippy"},
		{"npm install", "npm_install"},
		{"npm run build", "npm_run"},
		{"npm test", "npm_test"},
		{"docker build -t myapp .", "docker_build"},
		{"docker run -d nginx", "docker_run"},
		{"docker ps", "docker_ps"},
		{"docker images", "docker_images"},
		{"go build", "go_build"},
		{"go test ./...", "go_test"},
		{"go fmt", "go_fmt"},
		{"go vet", "go_vet"},
		{"unknown command", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := DetectCommandType(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEngine_FilterGit(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		shouldOpt  bool
		optType    string
		confidence float64
	}{
		{
			name:       "git clone without optimization",
			command:    "git clone https://github.com/repo",
			shouldOpt:  true,
			optType:    "git_clone",
			confidence: 0.85,
		},
		{
			name:       "git clone with depth",
			command:    "git clone --depth 1 https://github.com/repo",
			shouldOpt:  true,
			optType:    "git_clone",
			confidence: 0.85,
		},
		{
			name:       "git status",
			command:    "git status",
			shouldOpt:  true,
			optType:    "git_status",
			confidence: 0.95,
		},
		{
			name:       "git status short",
			command:    "git status -s",
			shouldOpt:  false,
			optType:    "git_status",
			confidence: 0.95,
		},
		{
			name:       "git log",
			command:    "git log",
			shouldOpt:  true,
			optType:    "git_log",
			confidence: 0.9,
		},
		{
			name:       "git push",
			command:    "git push origin main",
			shouldOpt:  true,
			optType:    "git_push",
			confidence: 0.8,
		},
		{
			name:       "git pull",
			command:    "git pull origin main",
			shouldOpt:  true,
			optType:    "git_pull",
			confidence: 0.75,
		},
		{
			name:       "git diff",
			command:    "git diff",
			shouldOpt:  true,
			optType:    "git_diff",
			confidence: 0.85,
		},
		{
			name:       "empty command",
			command:    "",
			shouldOpt:  false,
			optType:    "",
			confidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FilterGit(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldOpt, result.ShouldOptimize)
			assert.Equal(t, tt.optType, result.OptimizationType)
			assert.Equal(t, tt.confidence, result.Confidence)
			assert.Equal(t, tt.command, result.OriginalCommand)
		})
	}
}

func TestFilterEngine_FilterCargo(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		shouldOpt  bool
		optType    string
	}{
		{
			name:      "cargo build",
			command:   "cargo build",
			shouldOpt: true,
			optType:   "cargo_build",
		},
		{
			name:      "cargo test",
			command:   "cargo test",
			shouldOpt: true,
			optType:   "cargo_test",
		},
		{
			name:      "cargo check",
			command:   "cargo check",
			shouldOpt: true,
			optType:   "cargo_check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FilterCargo(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldOpt, result.ShouldOptimize)
			assert.Equal(t, tt.optType, result.OptimizationType)
		})
	}
}

func TestFilterEngine_FilterNPM(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		shouldOpt  bool
		optType    string
	}{
		{
			name:      "npm install",
			command:   "npm install lodash",
			shouldOpt: true,
			optType:   "npm_install",
		},
		{
			name:      "npm run",
			command:   "npm run build",
			shouldOpt: true,
			optType:   "npm_run",
		},
		{
			name:      "npm test",
			command:   "npm test",
			shouldOpt: true,
			optType:   "npm_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FilterNPM(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldOpt, result.ShouldOptimize)
			assert.Equal(t, tt.optType, result.OptimizationType)
		})
	}
}

func TestFilterEngine_FilterDocker(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		shouldOpt  bool
		optType    string
	}{
		{
			name:      "docker build",
			command:   "docker build .",
			shouldOpt: true,
			optType:   "docker_build",
		},
		{
			name:      "docker run",
			command:   "docker run nginx",
			shouldOpt: true,
			optType:   "docker_run",
		},
		{
			name:      "docker ps",
			command:   "docker ps",
			shouldOpt: true,
			optType:   "docker_ps",
		},
		{
			name:      "docker images",
			command:   "docker images",
			shouldOpt: true,
			optType:   "docker_images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FilterDocker(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldOpt, result.ShouldOptimize)
			assert.Equal(t, tt.optType, result.OptimizationType)
		})
	}
}

func TestFilterEngine_FilterGo(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		shouldOpt  bool
		optType    string
	}{
		{
			name:      "go build",
			command:   "go build ./...",
			shouldOpt: true,
			optType:   "go_build",
		},
		{
			name:      "go test",
			command:   "go test ./...",
			shouldOpt: true,
			optType:   "go_test",
		},
		{
			name:      "go fmt",
			command:   "go fmt ./...",
			shouldOpt: true,
			optType:   "go_fmt",
		},
		{
			name:      "go vet",
			command:   "go vet ./...",
			shouldOpt: true,
			optType:   "go_vet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FilterGo(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldOpt, result.ShouldOptimize)
			assert.Equal(t, tt.optType, result.OptimizationType)
		})
	}
}

func TestFilterEngine_Filter_AutoDetect(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name       string
		command    string
		optType    string
		confidence float64
	}{
		{"git command", "git status", "git_status", 0.95},
		{"cargo command", "cargo build", "cargo_build", 0.9},
		{"npm command", "npm install", "npm_install", 0.9},
		{"docker command", "docker ps", "docker_ps", 0.95},
		{"go command", "go build", "go_build", 0.9},
		{"generic command", "echo hello", "generic_shell", 0.3},
		{"empty command", "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Filter(tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.optType, result.OptimizationType)
			assert.Equal(t, tt.confidence, result.Confidence)
		})
	}
}

func TestFilterEngine_Filter_DisabledCommands(t *testing.T) {
	config := &FilterConfig{
		GitEnabled:    false,
		CargoEnabled:  false,
		NPMEnabled:    false,
		DockerEnabled: false,
		GoEnabled:     false,
	}
	engine := NewFilterEngineWithConfig(config)

	result, err := engine.Filter("git status")
	require.NoError(t, err)
	assert.Equal(t, "generic_shell", result.OptimizationType)
	assert.Equal(t, 0.3, result.Confidence)
}

func TestParseFilterConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *FilterConfig
	}{
		{
			name: "default values",
			input: `
git_enabled = true
cargo_enabled = true
npm_enabled = true
docker_enabled = true
go_enabled = true
min_output_lines = 100
max_command_age_days = 7
`,
			expected: &FilterConfig{
				GitEnabled:        true,
				CargoEnabled:      true,
				NPMEnabled:        true,
				DockerEnabled:     true,
				GoEnabled:         true,
				MinOutputLines:    100,
				MaxCommandAgeDays: 7,
			},
		},
		{
			name: "all disabled",
			input: `
git_enabled = false
cargo_enabled = false
npm_enabled = false
docker_enabled = false
go_enabled = false
`,
			expected: &FilterConfig{
				GitEnabled:        false,
				CargoEnabled:      false,
				NPMEnabled:        false,
				DockerEnabled:     false,
				GoEnabled:         false,
				MinOutputLines:    100,
				MaxCommandAgeDays: 7,
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: DefaultFilterConfig(),
		},
		{
			name: "comments only",
			input: `
# This is a comment
# Another comment
`,
			expected: DefaultFilterConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFilterConfig([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected.GitEnabled, result.GitEnabled)
			assert.Equal(t, tt.expected.CargoEnabled, result.CargoEnabled)
			assert.Equal(t, tt.expected.NPMEnabled, result.NPMEnabled)
			assert.Equal(t, tt.expected.DockerEnabled, result.DockerEnabled)
			assert.Equal(t, tt.expected.GoEnabled, result.GoEnabled)
			assert.Equal(t, tt.expected.MinOutputLines, result.MinOutputLines)
			assert.Equal(t, tt.expected.MaxCommandAgeDays, result.MaxCommandAgeDays)
		})
	}
}

func TestGetOptimizationSuggestions(t *testing.T) {
	tests := []struct {
		commandType string
		expectEmpty bool
	}{
		{"git_clone", false},
		{"git_status", false},
		{"cargo_build", false},
		{"npm_install", false},
		{"docker_run", false},
		{"go_test", false},
		{"unknown_type", true}, // Test for truly unknown types returns generic suggestions
	}

	for _, tt := range tests {
		t.Run(tt.commandType, func(t *testing.T) {
			suggestions := GetOptimizationSuggestions(tt.commandType)
			if tt.expectEmpty {
				assert.Empty(t, suggestions)
			} else {
				assert.NotEmpty(t, suggestions)
			}
		})
	}
}

func TestAnalyzeCommandPattern(t *testing.T) {
	tests := []struct {
		command        string
		expectVerbose  bool
		expectNoCache  bool
		expectForce    bool
	}{
		{"git clone -v repo", true, false, false},
		{"npm install --no-cache", false, true, false},
		{"docker build --force", false, false, true},
		{"cargo build", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := AnalyzeCommandPattern(tt.command)
			assert.Equal(t, tt.expectVerbose, result["verbose_output"])
			assert.Equal(t, tt.expectNoCache, result["no_cache"])
			assert.Equal(t, tt.expectForce, result["force_flag"])
		})
	}
}

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *FilterResult
		contains string
	}{
		{
			name: "optimized result",
			result: &FilterResult{
				ShouldOptimize:   true,
				OptimizationType: "git_clone",
				OriginalCommand:  "git clone repo",
				OptimizedCommand: "git clone repo --depth 1",
				EstimatedSavings: 50,
				Confidence:       0.85,
			},
			contains: "git clone repo",
		},
		{
			name: "not optimized result",
			result: &FilterResult{
				ShouldOptimize:   false,
				OptimizationType: "git_status",
				OriginalCommand:  "git status -s",
				OptimizedCommand: "git status -s",
				Confidence:       0.95,
			},
			contains: "No optimization suggested",
		},
		{
			name: "unknown command",
			result: &FilterResult{
				OriginalCommand: "",
				Confidence:       0,
			},
			contains: "Unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatResult(tt.result)
			assert.Contains(t, output, tt.contains)
		})
	}
}

func TestDefaultFilterConfig(t *testing.T) {
	config := DefaultFilterConfig()

	assert.True(t, config.GitEnabled)
	assert.True(t, config.CargoEnabled)
	assert.True(t, config.NPMEnabled)
	assert.True(t, config.DockerEnabled)
	assert.True(t, config.GoEnabled)
	assert.Equal(t, 100, config.MinOutputLines)
	assert.Equal(t, 7, config.MaxCommandAgeDays)
}

func TestNewFilterEngine(t *testing.T) {
	engine := NewFilterEngine()
	require.NotNil(t, engine)
	require.NotNil(t, engine.config)
}

func TestNewFilterEngineWithConfig(t *testing.T) {
	config := &FilterConfig{
		GitEnabled:    false,
		CargoEnabled:  true,
		NPMEnabled:    true,
		DockerEnabled: true,
		GoEnabled:     true,
	}
	engine := NewFilterEngineWithConfig(config)

	require.NotNil(t, engine)
	assert.Equal(t, config, engine.config)
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []*Token
		flag     string
		expected bool
	}{
		{
			name: "has flag",
			tokens: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenFlag, Value: "--depth"},
				{Type: TokenArgument, Value: "1"},
			},
			flag:     "--depth",
			expected: true,
		},
		{
			name: "has short flag",
			tokens: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenFlag, Value: "-v"},
			},
			flag:     "-v",
			expected: true,
		},
		{
			name: "missing flag",
			tokens: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "status"},
			},
			flag:     "--depth",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasFlag(tt.tokens, tt.flag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJoinTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []*Token
		expected string
	}{
		{
			name: "simple join",
			tokens: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "status"},
			},
			expected: "git status",
		},
		{
			name:     "empty tokens",
			tokens:   []*Token{},
			expected: "",
		},
		{
			name: "with flags",
			tokens: []*Token{
				{Type: TokenCommand, Value: "git"},
				{Type: TokenArgument, Value: "clone"},
				{Type: TokenFlag, Value: "--depth"},
				{Type: TokenArgument, Value: "1"},
			},
			expected: "git clone --depth 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinTokens(tt.tokens)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterResult_EstimatedSavings(t *testing.T) {
	engine := NewFilterEngine()

	result, err := engine.FilterGit("git clone https://github.com/large/repo")
	require.NoError(t, err)
	assert.Greater(t, result.EstimatedSavings, 0)
	assert.True(t, result.ShouldOptimize)
}

func TestFilterResult_OptimizedCommand(t *testing.T) {
	engine := NewFilterEngine()

	result, err := engine.FilterGit("git clone https://github.com/repo")
	require.NoError(t, err)
	assert.Contains(t, result.OptimizedCommand, "--depth")
	assert.Contains(t, result.OptimizedCommand, "--single-branch")
}

func TestFilterEngine_DisabledFilters(t *testing.T) {
	config := &FilterConfig{
		GitEnabled:    false,
		CargoEnabled:  true,
		NPMEnabled:    true,
		DockerEnabled: true,
		GoEnabled:     true,
	}
	engine := NewFilterEngineWithConfig(config)

	// Git should be treated as generic shell command
	result, err := engine.Filter("git status")
	require.NoError(t, err)
	assert.Equal(t, "generic_shell", result.OptimizationType)
	assert.Equal(t, 0.3, result.Confidence)
}
