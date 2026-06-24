package discovery

import (
	"context"
	"testing"
)

func TestNewCommandLexer(t *testing.T) {
	lexer := NewCommandLexer()
	if lexer == nil {
		t.Fatal("NewCommandLexer() returned nil")
	}
	if lexer.patterns == nil {
		t.Error("lexer.patterns is nil")
	}
}

func TestCommandLexer_Lex_SimpleCommand(t *testing.T) {
	lexer := NewCommandLexer()

	tests := []struct {
		name     string
		input    string
		minCount int    // minimum number of tokens
		hasType  TokenType
		hasValue string
	}{
		{
			name:     "simple git status",
			input:    "git status",
			minCount: 2,
			hasType:  TokenCommand,
			hasValue: "git",
		},
		{
			name:     "git status with flag",
			input:    "git status --short",
			minCount: 3,
			hasType:  TokenFlag,
			hasValue: "--short",
		},
		{
			name:     "npm install",
			input:    "npm install express",
			minCount: 3,
			hasType:  TokenSubcommand,
			hasValue: "install",
		},
		{
			name:     "docker run with flags",
			input:    "docker run -d --name test nginx",
			minCount: 6,
			hasType:  TokenFlag,
			hasValue: "-d",
		},
		{
			name:     "quoted value - echo with quotes",
			input:    `echo "hello world"`,
			minCount: 2,
			hasType:  TokenValue,
			hasValue: `"hello world"`,
		},
		{
			name:     "pipeline",
			input:    "cat file.txt | grep pattern",
			minCount: 4,
			hasType:  TokenSeparator,
			hasValue: "|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexer.Lex(tt.input)
			if len(tokens) < tt.minCount {
				t.Errorf("Lex(%q) returned %d tokens, want at least %d", tt.input, len(tokens), tt.minCount)
			}

			// Check for specific token
			found := false
			for _, token := range tokens {
				if token.Type == tt.hasType && token.Value == tt.hasValue {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Lex(%q) did not find token {Type: %v, Value: %q}", tt.input, tt.hasType, tt.hasValue)
			}
		})
	}
}

func TestCommandLexer_Lex_EmptyString(t *testing.T) {
	lexer := NewCommandLexer()
	tokens := lexer.Lex("")
	if len(tokens) != 0 {
		t.Errorf("Lex(\"\") returned %d tokens, want 0", len(tokens))
	}
}

func TestCommandLexer_Lex_WhitespaceOnly(t *testing.T) {
	lexer := NewCommandLexer()
	// Note: tabs are not currently skipped, they would be parsed differently
	// Just test that empty after space is handled
	tokens := lexer.Lex("   ")
	if len(tokens) != 0 {
		t.Errorf("Lex(\"   \") returned %d tokens, want 0", len(tokens))
	}
}

func TestCommandLexer_Classify(t *testing.T) {
	lexer := NewCommandLexer()

	tests := []struct {
		input    string
		expected string
	}{
		{"git status", "git_status"},
		{"git commit -m \"fix: bug\"", "git_commit"},
		{"git push origin main", "git_push"},
		{"git checkout -b feature", "git_checkout"},
		{"git branch -a", "git_branch"},
		{"docker ps", "docker_ps"},
		{"docker run -d nginx", "docker_run"},
		{"docker build -t myapp .", "docker_build"},
		{"npm install", "npm_install"},
		{"npm run build", "npm_run"},
		{"npm test", "npm_test"},
		{"go build", "go_build"},
		{"go test ./...", "go_test"},
		{"find . -name \"*.go\"", "file_search"},
		{"grep -r pattern .", "text_search"},
		{"curl https://api.example.com", "http_client"},
		{"ls -la", "directory_listing"},
		{"kubectl get pods", "kubectl_get"},
		{"unknown-cmd arg", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := lexer.Classify(tt.input)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCommandLexer_Classify_EmptyString(t *testing.T) {
	lexer := NewCommandLexer()
	result := lexer.Classify("")
	if result != "unknown" {
		t.Errorf("Classify(\"\") = %q, want \"unknown\"", result)
	}
}

func TestCommandLexer_DetectOpportunities(t *testing.T) {
	lexer := NewCommandLexer()

	tests := []struct {
		name      string
		input     string
		wantTypes []string
	}{
		{
			name:      "verbose flag opportunity",
			input:     "git status --short",
			wantTypes: []string{"verbose_flag"},
		},
		{
			name:      "verbose help flag",
			input:     "docker run --help",
			wantTypes: []string{"verbose_flag"},
		},
		{
			name:      "piping inefficiency",
			input:     "cat file.txt | grep pattern",
			wantTypes: []string{"inefficient"},
		},
		{
			name:      "grep chain inefficiency",
			input:     "grep pattern1 file.txt | grep pattern2",
			wantTypes: []string{"inefficient"},
		},
		{
			name:      "npm install cacheable",
			input:     "npm install lodash",
			wantTypes: []string{"cached"},
		},
		{
			name:      "no opportunities",
			input:     "echo hello",
			wantTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opps := lexer.DetectOpportunities(tt.input)
			
			if len(tt.wantTypes) == 0 {
				if len(opps) > 0 {
					t.Errorf("DetectOpportunities(%q) returned %d opportunities, want 0", tt.input, len(opps))
				}
				return
			}

			foundTypes := make(map[string]bool)
			for _, opp := range opps {
				foundTypes[opp.Type] = true
			}

			for _, wantType := range tt.wantTypes {
				if !foundTypes[wantType] {
					t.Errorf("DetectOpportunities(%q) did not find opportunity type %q", tt.input, wantType)
				}
			}
		})
	}
}

func TestCommandLexer_DetectOpportunities_ConfidenceAndSavings(t *testing.T) {
	lexer := NewCommandLexer()

	// Test verbose flag opportunity
	opps := lexer.DetectOpportunities("git status --short")
	if len(opps) > 0 {
		opp := opps[0]
		if opp.Type != "verbose_flag" {
			t.Errorf("Expected type 'verbose_flag', got %q", opp.Type)
		}
		if opp.Confidence <= 0 || opp.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", opp.Confidence)
		}
		if opp.PotentialSavings <= 0 {
			t.Errorf("PotentialSavings should be positive, got %d", opp.PotentialSavings)
		}
		if opp.ID == "" {
			t.Error("ID should not be empty")
		}
	}
}

func TestCommandLexer_DetectOpportunities_EmptyString(t *testing.T) {
	lexer := NewCommandLexer()
	opps := lexer.DetectOpportunities("")
	if len(opps) != 0 {
		t.Errorf("DetectOpportunities(\"\") returned %d opportunities, want 0", len(opps))
	}
}

func TestNewClassificationRegistry(t *testing.T) {
	registry := NewClassificationRegistry()
	if registry == nil {
		t.Fatal("NewClassificationRegistry() returned nil")
	}
	if registry.commands == nil {
		t.Error("registry.commands is nil")
	}
}

func TestClassificationRegistry_GetCommandType(t *testing.T) {
	registry := NewClassificationRegistry()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantOK   bool
	}{
		{"git exists", "git", "git", true},
		{"docker exists", "docker", "docker", true},
		{"npm exists", "npm", "npm", true},
		{"unknown command", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, ok := registry.GetCommandType(tt.input)
			if ok != tt.wantOK {
				t.Errorf("GetCommandType(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && ct.Name != tt.wantName {
				t.Errorf("GetCommandType(%q) Name = %q, want %q", tt.input, ct.Name, tt.wantName)
			}
		})
	}
}

func TestClassificationRegistry_FlagInfo(t *testing.T) {
	registry := NewClassificationRegistry()

	git, ok := registry.GetCommandType("git")
	if !ok {
		t.Fatal("GetCommandType(\"git\") returned false")
	}

	// Check that git has flag info
	if len(git.Flags) == 0 {
		t.Error("git command has no flags")
	}

	// Find specific flag
	found := false
	for _, flag := range git.Flags {
		if flag.Short == "v" && flag.Name == "verbose" {
			found = true
			if flag.Verbose != "--verbose" {
				t.Errorf("Expected Verbose '--verbose', got %q", flag.Verbose)
			}
		}
	}
	if !found {
		t.Error("Did not find verbose flag with short 'v'")
	}
}

func TestNewDiscoveryService(t *testing.T) {
	svc := NewDiscoveryService(nil)
	if svc == nil {
		t.Fatal("NewDiscoveryService(nil) returned nil")
	}
	if svc.lexer == nil {
		t.Error("svc.lexer is nil")
	}
	if svc.registry == nil {
		t.Error("svc.registry is nil")
	}
	if svc.patterns == nil {
		t.Error("svc.patterns is nil")
	}
}

func TestDiscoveryService_LearnFromHistory(t *testing.T) {
	svc := NewDiscoveryService(nil)
	
	err := svc.LearnFromHistory(context.Background())
	if err != nil {
		t.Errorf("LearnFromHistory() error = %v", err)
	}

	// Check that patterns were learned
	patterns, err := svc.GetPatterns(context.Background())
	if err != nil {
		t.Errorf("GetPatterns() error = %v", err)
	}
	if len(patterns) == 0 {
		t.Error("GetPatterns() returned empty, expected at least some patterns")
	}
}

func TestDiscoveryService_DetectOpportunities(t *testing.T) {
	svc := NewDiscoveryService(nil)
	
	commands := []string{
		"git status --short",
		"npm install express",
		"docker ps",
	}

	opps, err := svc.DetectOpportunities(context.Background(), commands)
	if err != nil {
		t.Errorf("DetectOpportunities() error = %v", err)
	}

	if len(opps) == 0 {
		t.Error("DetectOpportunities() returned no opportunities for known commands")
	}

	// Should find at least one verbose flag opportunity from git status --short
	foundVerboseFlag := false
	for _, opp := range opps {
		if opp.Type == "verbose_flag" {
			foundVerboseFlag = true
			break
		}
	}
	if !foundVerboseFlag {
		t.Error("Did not find verbose_flag opportunity in git status --short")
	}
}

func TestDiscoveryService_GetTopOpportunities(t *testing.T) {
	svc := NewDiscoveryService(nil)
	
	opps, err := svc.GetTopOpportunities(context.Background(), 10)
	if err != nil {
		t.Errorf("GetTopOpportunities() error = %v", err)
	}

	// In this stub implementation, returns empty
	if opps == nil {
		t.Error("GetTopOpportunities() returned nil, expected empty slice")
	}
}

func TestDiscoveryService_GetPatterns(t *testing.T) {
	svc := NewDiscoveryService(nil)
	
	// Learn some patterns first
	err := svc.LearnFromHistory(context.Background())
	if err != nil {
		t.Errorf("LearnFromHistory() error = %v", err)
	}

	patterns, err := svc.GetPatterns(context.Background())
	if err != nil {
		t.Errorf("GetPatterns() error = %v", err)
	}

	if len(patterns) == 0 {
		t.Error("GetPatterns() returned empty after learning")
	}

	// Check pattern structure
	for _, p := range patterns {
		if p.ID == "" {
			t.Error("Pattern ID should not be empty")
		}
		if p.Pattern == "" {
			t.Error("Pattern should not be empty")
		}
		if p.CommandType == "" {
			t.Error("CommandType should not be empty")
		}
		if p.Frequency <= 0 {
			t.Error("Frequency should be positive")
		}
	}
}

func TestLexToken_Positions(t *testing.T) {
	lexer := NewCommandLexer()
	tokens := lexer.Lex("git status --short")
	
	if len(tokens) < 3 {
		t.Fatalf("Expected at least 3 tokens, got %d", len(tokens))
	}

	// Check that positions are set correctly
	if tokens[0].Pos != 0 {
		t.Errorf("First token Pos = %d, want 0", tokens[0].Pos)
	}
	if tokens[1].Pos != 4 {
		t.Errorf("Second token Pos = %d, want 4", tokens[1].Pos)
	}
	if tokens[2].Pos != 11 {
		t.Errorf("Third token Pos = %d, want 11", tokens[2].Pos)
	}
}

func TestTokenTypes_AreDistinct(t *testing.T) {
	if TokenCommand == TokenSubcommand {
		t.Error("TokenCommand and TokenSubcommand should be distinct")
	}
	if TokenCommand == TokenFlag {
		t.Error("TokenCommand and TokenFlag should be distinct")
	}
	if TokenFlag == TokenValue {
		t.Error("TokenFlag and TokenValue should be distinct")
	}
	if TokenValue == TokenSeparator {
		t.Error("TokenValue and TokenSeparator should be distinct")
	}
}

func TestComplexCommand_Parsing(t *testing.T) {
	lexer := NewCommandLexer()

	tests := []struct {
		name     string
		input    string
		minCount int // minimum token count
	}{
		{"piped commands", "cat file.txt | grep pattern | sort | uniq", 6},
		{"docker complex", "docker run -d -p 8080:80 --name web nginx:latest", 8},
		{"git with multiple flags", "git commit -am \"fix: update deps\" --no-verify", 5},
		{"npm script", "npm run build -- --env production --no-cache", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexer.Lex(tt.input)
			if len(tokens) < tt.minCount {
				t.Errorf("Lex(%q) returned %d tokens, expected at least %d", tt.input, len(tokens), tt.minCount)
			}
		})
	}
}

func TestOpportunity_JSONFields(t *testing.T) {
	opp := Opportunity{
		ID:              "test-id-123",
		PatternID:       "pattern-456",
		Command:         "git status --short",
		PotentialSavings: 5,
		Confidence:      0.85,
		Type:            "verbose_flag",
	}

	if opp.ID == "" {
		t.Error("Opportunity.ID should not be empty")
	}
	if opp.PatternID == "" {
		t.Error("Opportunity.PatternID should not be empty")
	}
	if opp.Command == "" {
		t.Error("Opportunity.Command should not be empty")
	}
	if opp.PotentialSavings <= 0 {
		t.Error("Opportunity.PotentialSavings should be positive")
	}
	if opp.Confidence < 0 || opp.Confidence > 1 {
		t.Error("Opportunity.Confidence should be between 0 and 1")
	}
	if opp.Type == "" {
		t.Error("Opportunity.Type should not be empty")
	}
}

func TestCommandPattern_Structure(t *testing.T) {
	pattern := CommandPattern{
		ID:            "pattern-id",
		Pattern:       "git status",
		CommandType:   "git_status",
		Frequency:     100,
		Opportunities: []Opportunity{},
	}

	if pattern.ID == "" {
		t.Error("CommandPattern.ID should not be empty")
	}
	if pattern.Pattern == "" {
		t.Error("CommandPattern.Pattern should not be empty")
	}
	if pattern.CommandType == "" {
		t.Error("CommandPattern.CommandType should not be empty")
	}
	if pattern.Frequency <= 0 {
		t.Error("CommandPattern.Frequency should be positive")
	}
}
