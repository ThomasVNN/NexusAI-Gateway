package discovery

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// CommandPattern represents a discovered command pattern
type CommandPattern struct {
	ID            string         `json:"id"`
	Pattern       string         `json:"pattern"`
	CommandType   string         `json:"command_type"`
	Frequency     int            `json:"frequency"`
	LastSeen      time.Time      `json:"last_seen"`
	Opportunities []Opportunity  `json:"opportunities"`
}

// Opportunity represents an optimization opportunity
type Opportunity struct {
	ID              string `json:"id"`
	PatternID       string `json:"pattern_id"`
	Command         string `json:"command"`
	PotentialSavings int   `json:"potential_savings"`
	Confidence      float64 `json:"confidence"`
	Type            string `json:"type"` // "verbose_flag", "repeated", "inefficient", "cached"
}

// TokenType represents lexer token types
type TokenType int

const (
	TokenCommand TokenType = iota
	TokenSubcommand
	TokenFlag
	TokenValue
	TokenSeparator
)

// LexToken represents a lexer token
type LexToken struct {
	Type  TokenType
	Value string
	Pos   int
}

// CommandLexer tokenizes and classifies commands
type CommandLexer struct {
	patterns map[string]*regexp.Regexp
}

// NewCommandLexer creates a new command lexer
func NewCommandLexer() *CommandLexer {
	return &CommandLexer{
		patterns: map[string]*regexp.Regexp{
			"command":     regexp.MustCompile(`^([a-zA-Z0-9_-]+)`),
			"subcommand": regexp.MustCompile(`\s+([a-zA-Z0-9_-]+)`),
			"flag":        regexp.MustCompile(`(-{1,2}[a-zA-Z0-9_-]+)`),
			"value":       regexp.MustCompile(`=([^\s]+)|"([^"]+)"|'([^']+)'|(\S+)`),
			"separator":   regexp.MustCompile(`(\s+[|&;]\s+|\|\||&&|;)`),
		},
	}
}

// Lex parses a command string into tokens
func (l *CommandLexer) Lex(cmd string) []LexToken {
	var tokens []LexToken
	pos := 0

	for pos < len(cmd) {
		// Skip whitespace
		for pos < len(cmd) && cmd[pos] == ' ' {
			pos++
		}
		if pos >= len(cmd) {
			break
		}

		// Check for separator (|, &, ;)
		if cmd[pos] == '|' || cmd[pos] == '&' || cmd[pos] == ';' {
			end := pos + 1
			if end < len(cmd) && cmd[pos] == cmd[end] && cmd[pos] == '|' {
				end++
			}
			tokens = append(tokens, LexToken{
				Type:  TokenSeparator,
				Value: cmd[pos:end],
				Pos:   pos,
			})
			pos = end
			continue
		}

		// Check for flag (-- or -)
		if cmd[pos] == '-' {
			end := pos + 1
			if end < len(cmd) && cmd[end] == '-' {
				end++
			}
			// Capture the rest of the flag
			for end < len(cmd) && (cmd[end] == '-' || cmd[end] == '_' || (cmd[end] >= 'a' && cmd[end] <= 'z') || (cmd[end] >= 'A' && cmd[end] <= 'Z') || (cmd[end] >= '0' && cmd[end] <= '9')) {
				end++
			}
			flagValue := cmd[pos:end]
			tokens = append(tokens, LexToken{
				Type:  TokenFlag,
				Value: flagValue,
				Pos:   pos,
			})
			pos = end
			continue
		}

		// Check for quoted value
		if cmd[pos] == '"' || cmd[pos] == '\'' {
			quote := cmd[pos]
			end := pos + 1
			for end < len(cmd) && cmd[end] != quote {
				if cmd[end] == '\\' && end+1 < len(cmd) {
					end += 2
				} else {
					end++
				}
			}
			if end < len(cmd) {
				end++ // Include closing quote
			}
			tokens = append(tokens, LexToken{
				Type:  TokenValue,
				Value: cmd[pos:end],
				Pos:   pos,
			})
			pos = end
			continue
		}

		// Check for command/subcommand
		if (cmd[pos] >= 'a' && cmd[pos] <= 'z') || (cmd[pos] >= 'A' && cmd[pos] <= 'Z') || cmd[pos] == '_' {
			end := pos
			for end < len(cmd) && (cmd[end] == '_' || cmd[end] == '-' || (cmd[end] >= 'a' && cmd[end] <= 'z') || (cmd[end] >= 'A' && cmd[end] <= 'Z') || (cmd[end] >= '0' && cmd[end] <= '9')) {
				end++
			}
			value := cmd[pos:end]

			// Determine if this is a command or subcommand based on position
			tokenType := TokenCommand
			if len(tokens) > 0 && tokens[len(tokens)-1].Type == TokenCommand {
				tokenType = TokenSubcommand
			}

			tokens = append(tokens, LexToken{
				Type:  tokenType,
				Value: value,
				Pos:   pos,
			})
			pos = end
			continue
		}

		// Unquoted value
		end := pos
		for end < len(cmd) && cmd[end] != ' ' && cmd[end] != '|' && cmd[end] != '&' && cmd[end] != ';' {
			end++
		}
		if end > pos {
			tokens = append(tokens, LexToken{
				Type:  TokenValue,
				Value: cmd[pos:end],
				Pos:   pos,
			})
			pos = end
			continue
		}

		// Skip unknown character
		pos++
	}

	return tokens
}

// Classify determines the command type based on lexer tokens
func (l *CommandLexer) Classify(cmd string) string {
	tokens := l.Lex(cmd)
	if len(tokens) == 0 {
		return "unknown"
	}

	// Get the command name
	var command string
	for _, token := range tokens {
		if token.Type == TokenCommand {
			command = token.Value
			break
		}
	}

	// Classify based on common command patterns
	switch command {
	case "git":
		return classifyGitSubcommand(tokens)
	case "docker", "d":
		return classifyDockerSubcommand(tokens)
	case "npm", "npx":
		return classifyNpmSubcommand(tokens)
	case "go":
		return classifyGoSubcommand(tokens)
	case "kubectl", "k":
		return classifyKubectlSubcommand(tokens)
	case "aws":
		return classifyAwsSubcommand(tokens)
	case "python", "python3", "py":
		return classifyPythonSubcommand(tokens)
	case "find":
		return "file_search"
	case "grep", "rg":
		return "text_search"
	case "curl", "wget":
		return "http_client"
	case "cat", "less", "more":
		return "file_reader"
	case "ls", "dir":
		return "directory_listing"
	case "cp", "mv", "rm", "mkdir":
		return "file_operation"
	default:
		return "general"
	}
}

// classifyGitSubcommand classifies git commands
func classifyGitSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "commit", "cm":
				return "git_commit"
			case "push", "ps":
				return "git_push"
			case "pull":
				return "git_pull"
			case "branch", "br":
				return "git_branch"
			case "checkout", "co":
				return "git_checkout"
			case "status", "st":
				return "git_status"
			case "diff":
				return "git_diff"
			case "log":
				return "git_log"
			case "merge":
				return "git_merge"
			case "rebase":
				return "git_rebase"
			case "stash":
				return "git_stash"
			case "fetch":
				return "git_fetch"
			case "clone":
				return "git_clone"
			case "init":
				return "git_init"
			default:
				return "git_other"
			}
		}
	}
	return "git"
}

// classifyDockerSubcommand classifies docker commands
func classifyDockerSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "run":
				return "docker_run"
			case "build":
				return "docker_build"
			case "ps":
				return "docker_ps"
			case "images":
				return "docker_images"
			case "pull":
				return "docker_pull"
			case "push":
				return "docker_push"
			case "exec":
				return "docker_exec"
			case "logs":
				return "docker_logs"
			case "stop":
				return "docker_stop"
			case "rm", "rmi":
				return "docker_remove"
			default:
				return "docker_other"
			}
		}
	}
	return "docker"
}

// classifyNpmSubcommand classifies npm commands
func classifyNpmSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "install", "i":
				return "npm_install"
			case "run":
				return "npm_run"
			case "test":
				return "npm_test"
			case "start":
				return "npm_start"
			case "build":
				return "npm_build"
			case "dev":
				return "npm_dev"
			case "audit":
				return "npm_audit"
			case "update":
				return "npm_update"
			default:
				return "npm_other"
			}
		}
	}
	return "npm"
}

// classifyGoSubcommand classifies go commands
func classifyGoSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "build":
				return "go_build"
			case "run":
				return "go_run"
			case "test":
				return "go_test"
			case "get":
				return "go_get"
			case "install":
				return "go_install"
			case "mod":
				return "go_mod"
			case "fmt":
				return "go_fmt"
			case "vet":
				return "go_vet"
			case "clean":
				return "go_clean"
			default:
				return "go_other"
			}
		}
	}
	return "go"
}

// classifyKubectlSubcommand classifies kubectl commands
func classifyKubectlSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "get":
				return "kubectl_get"
			case "describe":
				return "kubectl_describe"
			case "apply":
				return "kubectl_apply"
			case "delete":
				return "kubectl_delete"
			case "create":
				return "kubectl_create"
			case "logs":
				return "kubectl_logs"
			case "exec":
				return "kubectl_exec"
			case "port-forward":
				return "kubectl_port_forward"
			default:
				return "kubectl_other"
			}
		}
	}
	return "kubectl"
}

// classifyAwsSubcommand classifies aws cli commands
func classifyAwsSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "s3":
				return "aws_s3"
			case "ec2":
				return "aws_ec2"
			case "lambda":
				return "aws_lambda"
			case "ecs":
				return "aws_ecs"
			case "rds":
				return "aws_rds"
			case "cloudformation":
				return "aws_cloudformation"
			default:
				return "aws_other"
			}
		}
	}
	return "aws"
}

// classifyPythonSubcommand classifies python commands
func classifyPythonSubcommand(tokens []LexToken) string {
	for _, token := range tokens {
		if token.Type == TokenSubcommand {
			switch token.Value {
			case "install":
				return "pip_install"
			case "uninstall":
				return "pip_uninstall"
			case "freeze":
				return "pip_freeze"
			default:
				return "python_other"
			}
		}
	}
	return "python"
}

// DetectOpportunities finds optimization opportunities in a command
func (l *CommandLexer) DetectOpportunities(cmd string) []Opportunity {
	var opportunities []Opportunity
	tokens := l.Lex(cmd)

	// Check for verbose flags that have short equivalents
	verboseFlags := map[string]string{
		"--help":             "-h",
		"--verbose":          "-v",
		"--version":          "-V",
		"--output":           "-o",
		"--force":            "-f",
		"--quiet":            "-q",
		"--recursive":        "-r",
		"--all":              "-a",
		"--list":             "-l",
		"--json":             "-j",
		"--yaml":             "-y",
		"--dry-run":          "-n",
		"--no-cache":         "",
		"--parallel":         "-p",
		"--concurrency":      "-c",
		"--wait":             "-w",
		"--timeout":          "-t",
		"--max-time":         "-m",
		"--retry":            "",
		"--no-verify":        "",
		"--skip-ci":          "",
		"--follow":           "-f",
		"--reverse":          "-r",
		"--color":            "-C",
		"--patch":            "-p",
		"--stat":             "-s",
		"--oneline":          "--pretty=format:%h %s",
		"--no-deps":          "",
		"--save-dev":         "-D",
		"--save-prod":        "-P",
		"--save-exact":       "-E",
		"--short":            "-s", // git status --short
		"--porcelain":        "",   // suggestion to use --short instead
	}

	for _, token := range tokens {
		if token.Type == TokenFlag {
			if short, exists := verboseFlags[token.Value]; exists && short != "" {
				opportunities = append(opportunities, Opportunity{
					ID:              uuid.New().String(),
					PatternID:       "",
					Command:         cmd,
					PotentialSavings: len(token.Value) - len(short),
					Confidence:      0.9,
					Type:            "verbose_flag",
				})
			}
		}
	}

	// Check for inefficient patterns
	inefficientPatterns := []struct {
		pattern *regexp.Regexp
		savings int
		message string
	}{
		{regexp.MustCompile(`find\s+.*\|\s*xargs\s+grep`), 15, "Use xargs -0 or -P with grep"},
		{regexp.MustCompile(`grep\s+.*\|\s*grep`), 10, "Combine grep patterns"},
		{regexp.MustCompile(`cat\s+[^\s]+\s*\|`), 4, "Use input redirection"},
		{regexp.MustCompile(`\|\s*sort\s*\|\s*uniq`), 5, "Use sort -u instead"},
		{regexp.MustCompile(`ls\s+-l\s+.*\.log`), 3, "Use find with -name"},
		{regexp.MustCompile(`docker\s+ps\s+-a`), 2, "Use docker ps without -a for running containers"},
		{regexp.MustCompile(`git\s+status\s+--porcelain`), 3, "Consider git diff --stat for summary"},
		{regexp.MustCompile(`npm\s+list\s+--depth=0`), 5, "Omit --depth=0 as it's the default"},
	}

	for _, ip := range inefficientPatterns {
		if ip.pattern.MatchString(cmd) {
			opportunities = append(opportunities, Opportunity{
				ID:              uuid.New().String(),
				PatternID:       "",
				Command:         cmd,
				PotentialSavings: ip.savings,
				Confidence:      0.7,
				Type:            "inefficient",
			})
		}
	}

	// Check for commands that could be cached
	cacheableCommands := []string{
		"npm install", "pip install", "go get", "docker pull",
		"git fetch", "git clone", "aws s3 sync",
	}

	for _, cc := range cacheableCommands {
		if len(cmd) > len(cc) && cmd[:len(cc)] == cc {
			opportunities = append(opportunities, Opportunity{
				ID:              uuid.New().String(),
				PatternID:       "",
				Command:         cmd,
				PotentialSavings: 50,
				Confidence:      0.6,
				Type:            "cached",
			})
		}
	}

	return opportunities
}

// CommandType represents a command type
type CommandType struct {
	Name        string
	Patterns    []*regexp.Regexp
	Flags       []FlagInfo
	Subcommands []string
}

// FlagInfo contains flag information
type FlagInfo struct {
	Name        string
	Short       string
	Description string
	Verbose     string
}

// ClassificationRegistry stores command classifications
type ClassificationRegistry struct {
	commands map[string]CommandType
}

// NewClassificationRegistry creates a new classification registry
func NewClassificationRegistry() *ClassificationRegistry {
	return &ClassificationRegistry{
		commands: map[string]CommandType{
			"git": {
				Name: "git",
				Flags: []FlagInfo{
					{Name: "verbose", Short: "v", Description: "Verbose output", Verbose: "--verbose"},
					{Name: "quiet", Short: "q", Description: "Quiet output", Verbose: "--quiet"},
					{Name: "force", Short: "f", Description: "Force operation", Verbose: "--force"},
					{Name: "dry-run", Short: "n", Description: "Dry run", Verbose: "--dry-run"},
					{Name: "all", Short: "a", Description: "All", Verbose: "--all"},
				},
				Subcommands: []string{"commit", "push", "pull", "branch", "checkout", "status", "diff", "log", "merge", "rebase", "stash", "fetch", "clone", "init", "add", "reset", "rebased"},
			},
			"docker": {
				Name: "docker",
				Flags: []FlagInfo{
					{Name: "help", Short: "h", Description: "Help", Verbose: "--help"},
					{Name: "version", Short: "v", Description: "Version", Verbose: "--version"},
					{Name: "rmi", Short: "", Description: "Remove image", Verbose: "--rmi"},
					{Name: "force", Short: "f", Description: "Force", Verbose: "--force"},
				},
				Subcommands: []string{"run", "build", "ps", "images", "pull", "push", "exec", "logs", "stop", "rm", "rmi", "start", "restart", "kill"},
			},
			"npm": {
				Name: "npm",
				Flags: []FlagInfo{
					{Name: "save-dev", Short: "D", Description: "Save as dev dependency", Verbose: "--save-dev"},
					{Name: "save-prod", Short: "P", Description: "Save as production dependency", Verbose: "--save-prod"},
					{Name: "save-exact", Short: "E", Description: "Save exact version", Verbose: "--save-exact"},
					{Name: "no-audit", Short: "", Description: "Skip audit", Verbose: "--no-audit"},
					{Name: "quiet", Short: "q", Description: "Quiet output", Verbose: "--quiet"},
				},
				Subcommands: []string{"install", "run", "test", "start", "build", "dev", "audit", "update", "uninstall", "publish"},
			},
		},
	}
}

// GetCommandType returns the command type for a given command name
func (r *ClassificationRegistry) GetCommandType(name string) (CommandType, bool) {
	ct, ok := r.commands[name]
	return ct, ok
}

// DiscoveryService manages command discovery
type DiscoveryService struct {
	lexer    *CommandLexer
	registry *ClassificationRegistry
	db       interface{}
	patterns map[string]*CommandPattern
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db interface{}) *DiscoveryService {
	return &DiscoveryService{
		lexer:    NewCommandLexer(),
		registry: NewClassificationRegistry(),
		db:       db,
		patterns: make(map[string]*CommandPattern),
	}
}

// LearnFromHistory learns patterns from command history (stub implementation)
func (s *DiscoveryService) LearnFromHistory(ctx context.Context) error {
	// In a real implementation, this would read from PostgreSQL or other storage
	// For now, we initialize with known patterns
	s.patterns["git_status"] = &CommandPattern{
		ID:          uuid.New().String(),
		Pattern:     "git status",
		CommandType: "git_status",
		Frequency:   100,
		LastSeen:    time.Now(),
	}
	s.patterns["npm_install"] = &CommandPattern{
		ID:          uuid.New().String(),
		Pattern:     "npm install",
		CommandType: "npm_install",
		Frequency:   80,
		LastSeen:    time.Now(),
	}
	s.patterns["docker_ps"] = &CommandPattern{
		ID:          uuid.New().String(),
		Pattern:     "docker ps",
		CommandType: "docker_ps",
		Frequency:   60,
		LastSeen:    time.Now(),
	}
	return nil
}

// DetectOpportunities finds all opportunities in commands
func (s *DiscoveryService) DetectOpportunities(ctx context.Context, commands []string) ([]Opportunity, error) {
	var allOpportunities []Opportunity

	for _, cmd := range commands {
		opps := s.lexer.DetectOpportunities(cmd)
		allOpportunities = append(allOpportunities, opps...)
	}

	return allOpportunities, nil
}

// GetTopOpportunities returns highest-value opportunities sorted by savings
func (s *DiscoveryService) GetTopOpportunities(ctx context.Context, limit int) ([]Opportunity, error) {
	// In real implementation, this would query from storage
	// Return empty slice for now
	return []Opportunity{}, nil
}

// GetPatterns returns discovered patterns
func (s *DiscoveryService) GetPatterns(ctx context.Context) ([]*CommandPattern, error) {
	var patterns []*CommandPattern
	for _, p := range s.patterns {
		patterns = append(patterns, p)
	}
	return patterns, nil
}
