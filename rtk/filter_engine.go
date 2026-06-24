package rtk

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// TokenType represents the type of token
type TokenType int

const (
	TokenCommand TokenType = iota
	TokenFlag
	TokenArgument
	TokenSubcommand
)

// Token represents a tokenized command component
type Token struct {
	Type  TokenType
	Value string
}

// Lexer tokenizes command strings
type Lexer struct {
	input string
	pos   int
}

// NewLexer creates a new Lexer instance
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: strings.TrimSpace(input),
		pos:   0,
	}
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() (*Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return nil, nil
	}

	// Check for flag
	if l.pos < len(l.input) && l.input[l.pos] == '-' {
		return l.scanFlag()
	}

	// Scan the next token (command, subcommand, or argument)
	return l.scanToken()
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func (l *Lexer) scanFlag() (*Token, error) {
	start := l.pos
	l.pos++ // skip '-'

	// Handle --flag
	if l.pos < len(l.input) && l.input[l.pos] == '-' {
		l.pos++
	}

	// Read flag name
	for l.pos < len(l.input) && !unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	return &Token{
		Type:  TokenFlag,
		Value: l.input[start:l.pos],
	}, nil
}

func (l *Lexer) scanToken() (*Token, error) {
	start := l.pos
	tokenType := TokenArgument

	for l.pos < len(l.input) && !unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	value := l.input[start:l.pos]

	// First token is always a command
	if start == 0 {
		tokenType = TokenCommand
	}

	return &Token{
		Type:  tokenType,
		Value: value,
	}, nil
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() ([]*Token, error) {
	var tokens []*Token

	for {
		token, err := l.NextToken()
		if err != nil {
			return nil, err
		}
		if token == nil {
			break
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// FilterConfig holds configuration for the filter engine
type FilterConfig struct {
	GitEnabled        bool `toml:"git_enabled"`
	CargoEnabled      bool `toml:"cargo_enabled"`
	NPMEnabled        bool `toml:"npm_enabled"`
	DockerEnabled     bool `toml:"docker_enabled"`
	GoEnabled         bool `toml:"go_enabled"`
	MinOutputLines    int  `toml:"min_output_lines"`
	MaxCommandAgeDays int  `toml:"max_command_age_days"`
}

// DefaultFilterConfig returns the default configuration
func DefaultFilterConfig() *FilterConfig {
	return &FilterConfig{
		GitEnabled:        true,
		CargoEnabled:      true,
		NPMEnabled:        true,
		DockerEnabled:     true,
		GoEnabled:         true,
		MinOutputLines:    100,
		MaxCommandAgeDays: 7,
	}
}

// FilterResult contains the result of filtering a command
type FilterResult struct {
	ShouldOptimize    bool
	OptimizationType  string
	OriginalCommand   string
	OptimizedCommand  string
	EstimatedSavings  int
	Confidence        float64
}

// FilterEngine analyzes and optimizes commands
type FilterEngine struct {
	config *FilterConfig
}

// NewFilterEngine creates a new FilterEngine with default config
func NewFilterEngine() *FilterEngine {
	return &FilterEngine{
		config: DefaultFilterConfig(),
	}
}

// NewFilterEngineWithConfig creates a new FilterEngine with custom config
func NewFilterEngineWithConfig(config *FilterConfig) *FilterEngine {
	return &FilterEngine{
		config: config,
	}
}

// ParseFilterConfig parses TOML configuration
func ParseFilterConfig(data []byte) (*FilterConfig, error) {
	cfg := DefaultFilterConfig()

	// Simple TOML parser for our config structure
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "git_enabled":
			cfg.GitEnabled = parseBool(value)
		case "cargo_enabled":
			cfg.CargoEnabled = parseBool(value)
		case "npm_enabled":
			cfg.NPMEnabled = parseBool(value)
		case "docker_enabled":
			cfg.DockerEnabled = parseBool(value)
		case "go_enabled":
			cfg.GoEnabled = parseBool(value)
		case "min_output_lines":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MinOutputLines = v
			}
		case "max_command_age_days":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxCommandAgeDays = v
			}
		}
	}

	return cfg, nil
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.Trim(s, "\""))
	return s == "true" || s == "yes" || s == "1"
}

// FilterGit analyzes git commands and suggests optimizations
func (e *FilterEngine) FilterGit(cmd string) (*FilterResult, error) {
	lexer := NewLexer(cmd)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	result := &FilterResult{
		OriginalCommand: cmd,
		Confidence:      0.9,
	}

	command := strings.ToLower(tokens[0].Value)
	subcommand := ""
	if len(tokens) > 1 {
		subcommand = strings.ToLower(tokens[1].Value)
	}

	switch command {
	case "git":
		switch subcommand {
		case "clone":
			result = e.optimizeClone(tokens)
		case "push":
			result = e.optimizePush(tokens)
		case "pull":
			result = e.optimizePull(tokens)
		case "log":
			result = e.optimizeLog(tokens)
		case "status":
			result = e.optimizeStatus(tokens)
		case "diff":
			result = e.optimizeDiff(tokens)
		default:
			result.OptimizationType = "generic_git"
			result.Confidence = 0.5
		}
	default:
		result.Confidence = 0
	}

	return result, nil
}

func (e *FilterEngine) optimizeClone(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_clone",
		Confidence:      0.85,
	}

	hasDepth := hasFlag(tokens, "--depth")
	hasSingleBranch := hasFlag(tokens, "--single-branch")

	optimizations := []string{}

	if !hasDepth {
		optimizations = append(optimizations, "--depth=1")
	}

	if !hasSingleBranch {
		optimizations = append(optimizations, "--single-branch")
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 50 // KB
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizePush(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_push",
		Confidence:      0.8,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "--quiet") && !hasFlag(tokens, "-q") {
		optimizations = append(optimizations, "--quiet")
	}

	if !hasFlag(tokens, "--tags") && !hasFlag(tokens, "-t") {
		// Check if tags might be needed - conservative approach
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = tokens[0].Value + " " + tokens[1].Value + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 5
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizePull(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_pull",
		Confidence:      0.75,
	}

	if !hasFlag(tokens, "--rebase") && !hasFlag(tokens, "-r") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --rebase"
		result.EstimatedSavings = 3
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeLog(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_log",
		Confidence:      0.9,
	}

	optimizations := []string{"--oneline"}

	if hasFlag(tokens, "-n") {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	} else if !hasFlag(tokens, "--oneline") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --oneline -20"
		result.EstimatedSavings = 80
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	_ = optimizations // suppress unused warning

	return result
}

func (e *FilterEngine) optimizeStatus(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_status",
		Confidence:      0.95,
	}

	if !hasFlag(tokens, "-s") && !hasFlag(tokens, "--short") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -s"
		result.EstimatedSavings = 60
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeDiff(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "git_diff",
		Confidence:      0.85,
	}

	if !hasFlag(tokens, "--stat") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --stat"
		result.EstimatedSavings = 40
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

// FilterCargo analyzes cargo commands
func (e *FilterEngine) FilterCargo(cmd string) (*FilterResult, error) {
	lexer := NewLexer(cmd)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	result := &FilterResult{
		OriginalCommand: cmd,
		Confidence:      0.85,
	}

	command := strings.ToLower(tokens[0].Value)
	subcommand := ""
	if len(tokens) > 1 {
		subcommand = strings.ToLower(tokens[1].Value)
	}

	switch command {
	case "cargo":
		switch subcommand {
		case "build":
			result = e.optimizeCargoBuild(tokens)
		case "test":
			result = e.optimizeCargoTest(tokens)
		case "check":
			result = e.optimizeCargoCheck(tokens)
		default:
			result.OptimizationType = "generic_cargo"
			result.Confidence = 0.5
		}
	default:
		result.Confidence = 0
	}

	return result, nil
}

func (e *FilterEngine) optimizeCargoBuild(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "cargo_build",
		Confidence:      0.9,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "--release") && !hasFlag(tokens, "-r") {
		// Check for debug vs release suggestion
	}

	if !hasFlag(tokens, "--jobs") && !hasFlag(tokens, "-j") {
		optimizations = append(optimizations, "-j 4")
	}

	if !hasFlag(tokens, "--locked") {
		optimizations = append(optimizations, "--locked")
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 100
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeCargoTest(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "cargo_test",
		Confidence:      0.85,
	}

	optimizations := []string{}

	if hasFlag(tokens, "--doc") {
		// Doc tests are expensive
		if !hasFlag(tokens, "--release") {
			optimizations = append(optimizations, "--release")
		}
	}

	if hasFlag(tokens, "--no-fail-fast") {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	} else {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --no-fail-fast"
		result.EstimatedSavings = 20
	}

	_ = optimizations

	return result
}

func (e *FilterEngine) optimizeCargoCheck(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "cargo_check",
		Confidence:      0.95,
	}

	if !hasFlag(tokens, "--all-targets") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --all-targets"
		result.EstimatedSavings = 30
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

// FilterNPM analyzes npm commands
func (e *FilterEngine) FilterNPM(cmd string) (*FilterResult, error) {
	lexer := NewLexer(cmd)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	result := &FilterResult{
		OriginalCommand: cmd,
		Confidence:      0.85,
	}

	command := strings.ToLower(tokens[0].Value)

	switch command {
	case "npm":
		if len(tokens) > 1 {
			subcommand := strings.ToLower(tokens[1].Value)
			switch subcommand {
			case "install":
				result = e.optimizeNPMInstall(tokens)
			case "run":
				result = e.optimizeNPMRun(tokens)
			case "test":
				result = e.optimizeNPMTest(tokens)
			case "build":
				result = e.optimizeNPMBuild(tokens)
			default:
				result.OptimizationType = "generic_npm"
				result.Confidence = 0.5
			}
		}
	default:
		result.Confidence = 0
	}

	return result, nil
}

func (e *FilterEngine) optimizeNPMInstall(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "npm_install",
		Confidence:      0.9,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "--prefer-offline") {
		optimizations = append(optimizations, "--prefer-offline")
	}

	if hasFlag(tokens, "--save-dev") || hasFlag(tokens, "-D") {
		if !hasFlag(tokens, "--ignore-scripts") {
			optimizations = append(optimizations, "--ignore-scripts")
		}
	}

	if !hasFlag(tokens, "--no-audit") {
		optimizations = append(optimizations, "--no-audit")
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 50
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeNPMRun(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "npm_run",
		Confidence:      0.8,
	}

	if !hasFlag(tokens, "--if-present") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --if-present"
		result.EstimatedSavings = 10
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeNPMTest(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "npm_test",
		Confidence:      0.85,
	}

	if !hasFlag(tokens, "--runInBand") && !hasFlag(tokens, "-r") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --runInBand"
		result.EstimatedSavings = 40
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeNPMBuild(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "npm_build",
		Confidence:      0.8,
	}

	if !hasFlag(tokens, "--ignore-scripts") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " --ignore-scripts"
		result.EstimatedSavings = 30
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

// FilterDocker analyzes docker commands
func (e *FilterEngine) FilterDocker(cmd string) (*FilterResult, error) {
	lexer := NewLexer(cmd)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	result := &FilterResult{
		OriginalCommand: cmd,
		Confidence:      0.85,
	}

	command := strings.ToLower(tokens[0].Value)
	subcommand := ""
	if len(tokens) > 1 {
		subcommand = strings.ToLower(tokens[1].Value)
	}

	switch command {
	case "docker":
		switch subcommand {
		case "build":
			result = e.optimizeDockerBuild(tokens)
		case "run":
			result = e.optimizeDockerRun(tokens)
		case "ps":
			result = e.optimizeDockerPS(tokens)
		case "images":
			result = e.optimizeDockerImages(tokens)
		default:
			result.OptimizationType = "generic_docker"
			result.Confidence = 0.5
		}
	default:
		result.Confidence = 0
	}

	return result, nil
}

func (e *FilterEngine) optimizeDockerBuild(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "docker_build",
		Confidence:      0.9,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "--no-cache") && !hasFlag(tokens, "--pull") {
		optimizations = append(optimizations, "--pull")
	}

	if !hasFlag(tokens, "--compress") {
		optimizations = append(optimizations, "--compress")
	}

	if !hasFlag(tokens, "-t") && !hasFlag(tokens, "--tag") {
		// Tag is important for identification
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 200
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeDockerRun(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "docker_run",
		Confidence:      0.85,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "-d") && !hasFlag(tokens, "--detach") {
		// Detach is often desired
	}

	if !hasFlag(tokens, "--rm") {
		optimizations = append(optimizations, "--rm")
	}

	if !hasFlag(tokens, "--network") {
		optimizations = append(optimizations, "--network host")
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 20
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeDockerPS(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "docker_ps",
		Confidence:      0.95,
	}

	if !hasFlag(tokens, "-q") && !hasFlag(tokens, "--quiet") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -q"
		result.EstimatedSavings = 70
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeDockerImages(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "docker_images",
		Confidence:      0.95,
	}

	if !hasFlag(tokens, "-q") && !hasFlag(tokens, "--quiet") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -q"
		result.EstimatedSavings = 60
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

// FilterGo analyzes go commands
func (e *FilterEngine) FilterGo(cmd string) (*FilterResult, error) {
	lexer := NewLexer(cmd)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	result := &FilterResult{
		OriginalCommand: cmd,
		Confidence:      0.85,
	}

	command := strings.ToLower(tokens[0].Value)

	switch command {
	case "go":
		if len(tokens) > 1 {
			subcommand := strings.ToLower(tokens[1].Value)
			switch subcommand {
			case "build":
				result = e.optimizeGoBuild(tokens)
			case "test":
				result = e.optimizeGoTest(tokens)
			case "fmt":
				result = e.optimizeGoFmt(tokens)
			case "vet":
				result = e.optimizeGoVet(tokens)
			default:
				result.OptimizationType = "generic_go"
				result.Confidence = 0.5
			}
		}
	default:
		result.Confidence = 0
	}

	return result, nil
}

func (e *FilterEngine) optimizeGoBuild(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "go_build",
		Confidence:      0.9,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "-ldflags") {
		// Could suggest optimization flags
	}

	if hasFlag(tokens, "-v") && !hasFlag(tokens, "-vv") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -v"
		result.EstimatedSavings = 50
	} else if !hasFlag(tokens, "-v") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -v"
		result.EstimatedSavings = 30
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	_ = optimizations

	return result
}

func (e *FilterEngine) optimizeGoTest(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "go_test",
		Confidence:      0.9,
	}

	optimizations := []string{}

	if !hasFlag(tokens, "-short") {
		optimizations = append(optimizations, "-short")
	}

	if !hasFlag(tokens, "-count=1") {
		optimizations = append(optimizations, "-count=1")
	}

	if !hasFlag(tokens, "-race") {
		// Suggest race detector
	}

	if len(optimizations) > 0 {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " " + strings.Join(optimizations, " ")
		result.EstimatedSavings = 40
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeGoFmt(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "go_fmt",
		Confidence:      0.95,
	}

	if !hasFlag(tokens, "-l") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -l"
		result.EstimatedSavings = 20
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

func (e *FilterEngine) optimizeGoVet(tokens []*Token) *FilterResult {
	result := &FilterResult{
		OriginalCommand:  joinTokens(tokens),
		OptimizationType: "go_vet",
		Confidence:      0.9,
	}

	if !hasFlag(tokens, "-v") {
		result.ShouldOptimize = true
		result.OptimizedCommand = result.OriginalCommand + " -v"
		result.EstimatedSavings = 25
	} else {
		result.ShouldOptimize = false
		result.OptimizedCommand = result.OriginalCommand
	}

	return result
}

// Filter auto-detects command type and applies appropriate filter
func (e *FilterEngine) Filter(cmd string) (*FilterResult, error) {
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		return &FilterResult{
			OriginalCommand: cmd,
			Confidence:      0,
		}, nil
	}

	// Auto-detect command type
	firstWord := strings.Split(cmd, " ")[0]
	firstWord = strings.ToLower(firstWord)

	// Check for git
	if firstWord == "git" && e.config.GitEnabled {
		return e.FilterGit(cmd)
	}

	// Check for cargo
	if firstWord == "cargo" && e.config.CargoEnabled {
		return e.FilterCargo(cmd)
	}

	// Check for npm
	if firstWord == "npm" && e.config.NPMEnabled {
		return e.FilterNPM(cmd)
	}

	// Check for docker
	if firstWord == "docker" && e.config.DockerEnabled {
		return e.FilterDocker(cmd)
	}

	// Check for go
	if firstWord == "go" && e.config.GoEnabled {
		return e.FilterGo(cmd)
	}

	// Generic shell command
	return &FilterResult{
		OriginalCommand: cmd,
		OptimizationType: "generic_shell",
		Confidence:      0.3,
		ShouldOptimize:  false,
	}, nil
}

// Helper functions

func hasFlag(tokens []*Token, flag string) bool {
	for _, token := range tokens {
		if token.Type == TokenFlag && strings.HasPrefix(token.Value, flag) {
			return true
		}
	}
	return false
}

func joinTokens(tokens []*Token) string {
	var parts []string
	for _, t := range tokens {
		parts = append(parts, t.Value)
	}
	return strings.Join(parts, " ")
}

// DetectCommandType returns the type of command
func DetectCommandType(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	parts := strings.SplitN(cmd, " ", 2)

	if len(parts) == 0 {
		return "unknown"
	}

	first := strings.ToLower(parts[0])
	second := ""
	if len(parts) > 1 {
		parts2 := strings.SplitN(parts[1], " ", 2)
		second = strings.ToLower(parts2[0])
	}

	commandMap := map[string]map[string]string{
		"git": {
			"clone": "git_clone",
			"push":  "git_push",
			"pull":  "git_pull",
			"commit": "git_commit",
			"branch": "git_branch",
			"merge": "git_merge",
			"log":   "git_log",
			"diff":  "git_diff",
			"status": "git_status",
			"fetch": "git_fetch",
			"checkout": "git_checkout",
			"rebase": "git_rebase",
			"stash": "git_stash",
		},
		"cargo": {
			"build":  "cargo_build",
			"test":   "cargo_test",
			"run":    "cargo_run",
			"clippy": "cargo_clippy",
			"check":  "cargo_check",
			"fmt":    "cargo_fmt",
			"publish": "cargo_publish",
		},
		"npm": {
			"install": "npm_install",
			"run":     "npm_run",
			"test":    "npm_test",
			"build":   "npm_build",
			"start":   "npm_start",
			"stop":    "npm_stop",
		},
		"docker": {
			"build":   "docker_build",
			"run":     "docker_run",
			"ps":      "docker_ps",
			"images":  "docker_images",
			"pull":    "docker_pull",
			"push":    "docker_push",
			"exec":    "docker_exec",
			"logs":    "docker_logs",
		},
		"go": {
			"build":  "go_build",
			"test":   "go_test",
			"run":    "go_run",
			"fmt":    "go_fmt",
			"vet":    "go_vet",
			"get":    "go_get",
			"mod":    "go_mod",
		},
	}

	if subCommands, ok := commandMap[first]; ok {
		if subCmd, ok := subCommands[second]; ok {
			return subCmd
		}
		if second == "" {
			return first + "_base"
		}
		return first
	}

	return "generic"
}

// Common patterns for optimization detection
var optimizationPatterns = map[string]*regexp.Regexp{
	"verbose_output": regexp.MustCompile(`(-v|--verbose|--debug|--trace)`),
	"no_cache":       regexp.MustCompile(`--no-cache|--no-cache-dir`),
	"force_flag":     regexp.MustCompile(`(-f|--force|-y|--yes|--assume-yes)`),
}

// AnalyzeCommandPattern analyzes command patterns for optimization opportunities
func AnalyzeCommandPattern(cmd string) map[string]bool {
	result := make(map[string]bool)

	for name, pattern := range optimizationPatterns {
		result[name] = pattern.MatchString(cmd)
	}

	return result
}

// GetOptimizationSuggestions returns optimization suggestions for a command
func GetOptimizationSuggestions(cmdType string) []string {
	suggestions := map[string][]string{
		"git_clone": {
			"Use --depth=1 for shallow clones",
			"Use --single-branch to clone only one branch",
			"Consider --jobs=N for parallel fetching",
		},
		"git_push": {
			"Use --quiet to reduce output",
			"Consider --atomic for atomic pushes",
		},
		"git_pull": {
			"Use --rebase for cleaner history",
			"Consider --ff-only for fast-forward only",
		},
		"git_log": {
			"Use --oneline for concise output",
			"Use -n to limit commit count",
			"Consider --graph for visual history",
		},
		"git_status": {
			"Use -s for short format",
			"Consider --short for compact output",
		},
		"git_diff": {
			"Use --stat for summary",
			"Consider --name-only for file list only",
		},
		"cargo_build": {
			"Use --release for optimized builds",
			"Consider --jobs=N for parallel compilation",
			"Use --locked for reproducible builds",
		},
		"cargo_test": {
			"Use --release for faster tests",
			"Consider --no-fail-fast to run all tests",
		},
		"cargo_check": {
			"Use --all-targets to check all targets",
			"Consider --message-format=json for machine parsing",
		},
		"npm_install": {
			"Use --prefer-offline for faster installs",
			"Consider --no-audit to skip security audit",
		},
		"npm_run": {
			"Use --if-present to avoid errors",
			"Consider --silent for less output",
		},
		"npm_test": {
			"Use --runInBand for sequential tests",
			"Consider --coverage for test coverage",
		},
		"docker_build": {
			"Use --pull for fresh base images",
			"Consider --compress for smaller context",
			"Use --no-cache when not needed",
		},
		"docker_run": {
			"Use --rm for automatic cleanup",
			"Consider --network host for host networking",
		},
		"docker_ps": {
			"Use -q for quiet mode (IDs only)",
			"Consider -a for all containers",
		},
		"docker_images": {
			"Use -q for quiet mode (IDs only)",
			"Consider --format for custom output",
		},
		"go_build": {
			"Use -ldflags for build info injection",
			"Consider -s -w for binary size reduction",
		},
		"go_test": {
			"Use -short for shorter test runs",
			"Consider -count=1 to disable test caching",
			"Use -race for race detection",
		},
		"go_fmt": {
			"Use -l to list unformatted files",
			"Consider -w to write changes",
		},
		"go_vet": {
			"Use -v for verbose output",
			"Consider running with go build first",
		},
	}

	if s, ok := suggestions[cmdType]; ok {
		return s
	}

	return []string{
		"Check command documentation for performance flags",
		"Consider parallel execution options",
		"Review output verbosity settings",
	}
}

// FormatResult formats a FilterResult for display
func FormatResult(result *FilterResult) string {
	if result.Confidence == 0 {
		return fmt.Sprintf("Unknown command: %s", result.OriginalCommand)
	}

	lines := []string{
		fmt.Sprintf("Command: %s", result.OriginalCommand),
		fmt.Sprintf("Type: %s", result.OptimizationType),
		fmt.Sprintf("Confidence: %.0f%%", result.Confidence*100),
	}

	if result.ShouldOptimize {
		lines = append(lines,
			fmt.Sprintf("Optimized: %s", result.OptimizedCommand),
			fmt.Sprintf("Estimated Savings: %d bytes", result.EstimatedSavings),
		)
	} else {
		lines = append(lines, "No optimization suggested")
	}

	return strings.Join(lines, "\n")
}
