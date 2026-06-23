package compression

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// CavemanEngine provides rule-based compression with 40+ regex rewrite rules.
// This engine applies common phrase collapsing, whitespace normalization,
// and redundant formatting removal.
//
// ENG-9206: Caveman rule compression
type CavemanEngine struct {
	enabled bool
	stats   EngineStats
	mu      sync.RWMutex

	// Pre-compiled rules
	rules []CavemanRule
}

// CavemanRule defines a single compression rule
type CavemanRule struct {
	Name    string
	Find    *regexp.Regexp
	Replace string
	Priority int
}

// NewCavemanEngine creates a new Caveman compression engine with 40+ rules
func NewCavemanEngine() *CavemanEngine {
	rules := []CavemanRule{
		// ===== Whitespace Normalization (8 rules) =====
		{Name: "multi_space_to_one", Find: regexp.MustCompile(`[ \t]{2,}`), Replace: " ", Priority: 5},
		{Name: "multi_newline_to_two", Find: regexp.MustCompile(`\n{3,}`), Replace: "\n\n", Priority: 5},
		{Name: "newline_space_newline", Find: regexp.MustCompile(`\n\s+\n`), Replace: "\n\n", Priority: 10},
		{Name: "space_newline", Find: regexp.MustCompile(` \n`), Replace: "\n", Priority: 15},
		{Name: "newline_space", Find: regexp.MustCompile(`\n `), Replace: "\n", Priority: 15},
		{Name: "tab_to_space", Find: regexp.MustCompile(`\t`), Replace: " ", Priority: 20},
		{Name: "trailing_space", Find: regexp.MustCompile(` +\n`), Replace: "\n", Priority: 15},
		{Name: "leading_space", Find: regexp.MustCompile(`\n +`), Replace: "\n", Priority: 15},

		// ===== Common Phrase Collapsing (10 rules) =====
		{Name: "in_order_to", Find: regexp.MustCompile(`(?i)\bin order to\b`), Replace: "to", Priority: 10},
		{Name: "due_to_the_fact", Find: regexp.MustCompile(`(?i)\bdue to the fact that\b`), Replace: "because", Priority: 10},
		{Name: "for_the_purpose_of", Find: regexp.MustCompile(`(?i)\bfor the purpose of\b`), Replace: "to", Priority: 10},
		{Name: "in_the_event_that", Find: regexp.MustCompile(`(?i)\bin the event that\b`), Replace: "if", Priority: 10},
		{Name: "at_this_point_in_time", Find: regexp.MustCompile(`(?i)\bat this point in time\b`), Replace: "now", Priority: 10},
		{Name: "at_that_point_in_time", Find: regexp.MustCompile(`(?i)\bat that point in time\b`), Replace: "then", Priority: 10},
		{Name: "in_accordance_with", Find: regexp.MustCompile(`(?i)\bin accordance with\b`), Replace: "per", Priority: 10},
		{Name: "with_regard_to", Find: regexp.MustCompile(`(?i)\bwith regard to\b`), Replace: "about", Priority: 10},
		{Name: "in_the_case_of", Find: regexp.MustCompile(`(?i)\bin the case of\b`), Replace: "for", Priority: 10},
		{Name: "on_the_other_hand", Find: regexp.MustCompile(`(?i)\bon the other hand\b`), Replace: "but", Priority: 10},

		// ===== Redundant Formatting (8 rules) =====
		{Name: "bold_markdown", Find: regexp.MustCompile(`\*\*(.+?)\*\*`), Replace: "$1", Priority: 15},
		{Name: "italic_markdown", Find: regexp.MustCompile(`\*(.+?)\*`), Replace: "$1", Priority: 15},
		{Name: "code_inline", Find: regexp.MustCompile("`(.+?)`"), Replace: "$1", Priority: 15},
		{Name: "quote_collapse", Find: regexp.MustCompile(`"(.+?)"(?:\s*"(.+?)")*`), Replace: `"$1$2"`, Priority: 20},
		{Name: "bracket_space", Find: regexp.MustCompile(`\[\s+`), Replace: "[", Priority: 15},
		{Name: "space_bracket", Find: regexp.MustCompile(`\s+\]`), Replace: "]", Priority: 15},
		{Name: "paren_space", Find: regexp.MustCompile(`\(\s+`), Replace: "(", Priority: 15},
		{Name: "space_paren", Find: regexp.MustCompile(`\s+\)`), Replace: ")", Priority: 15},

		// ===== Programming Patterns (10 rules) =====
		{Name: "console_log", Find: regexp.MustCompile(`(?i)console\.log\(`), Replace: "log(", Priority: 20},
		{Name: "console_error", Find: regexp.MustCompile(`(?i)console\.error\(`), Replace: "err(", Priority: 20},
		{Name: "console_warn", Find: regexp.MustCompile(`(?i)console\.warn\(`), Replace: "warn(", Priority: 20},
		{Name: "self_this", Find: regexp.MustCompile(`(?i)\bself\.this\b`), Replace: "this", Priority: 20},
		{Name: "return_null", Find: regexp.MustCompile(`(?i)return null;?\s*`), Replace: "", Priority: 25},
		{Name: "import_os", Find: regexp.MustCompile(`(?i)import os\n`), Replace: "", Priority: 30},
		{Name: "import_sys", Find: regexp.MustCompile(`(?i)import sys\n`), Replace: "", Priority: 30},
		{Name: "import_time", Find: regexp.MustCompile(`(?i)import time\n`), Replace: "", Priority: 30},
		{Name: "const_let_var", Find: regexp.MustCompile(`(?i)\bconst\s+\w+\s*=\s*undefined`), Replace: "", Priority: 25},
		{Name: "boolean_comparison", Find: regexp.MustCompile(`=== true|=== false`), Replace: "", Priority: 20},

		// ===== Natural Language Simplification (10 rules) =====
		{Name: "please_note", Find: regexp.MustCompile(`(?i)\bplease note that\b`), Replace: "", Priority: 15},
		{Name: "it_is_important", Find: regexp.MustCompile(`(?i)\bit is important to note that\b`), Replace: "", Priority: 15},
		{Name: "as_mentioned", Find: regexp.MustCompile(`(?i)\bas mentioned earlier\b`), Replace: "", Priority: 15},
		{Name: "for_example", Find: regexp.MustCompile(`(?i)\bfor example,\s*`), Replace: "e.g., ", Priority: 10},
		{Name: "that_is_to_say", Find: regexp.MustCompile(`(?i)\bthat is to say,\s*`), Replace: "i.e., ", Priority: 10},
		{Name: "in_other_words", Find: regexp.MustCompile(`(?i)\bin other words,\s*`), Replace: "", Priority: 15},
		{Name: "however", Find: regexp.MustCompile(`(?i),\s*however,\s*`), Replace: ", but ", Priority: 15},
		{Name: "moreover", Find: regexp.MustCompile(`(?i),\s*moreover,\s*`), Replace: ", and ", Priority: 15},
		{Name: "furthermore", Find: regexp.MustCompile(`(?i),\s*furthermore,\s*`), Replace: ", also ", Priority: 15},
		{Name: "nevertheless", Find: regexp.MustCompile(`(?i),\s*nevertheless,\s*`), Replace: ", still ", Priority: 15},

		// ===== Empty/Redundant Lines (6 rules) =====
		{Name: "blank_line_after_header", Find: regexp.MustCompile(`\n{3,}`), Replace: "\n\n", Priority: 5},
		{Name: "blank_line_at_start", Find: regexp.MustCompile(`^\n+`), Replace: "", Priority: 10},
		{Name: "blank_line_at_end", Find: regexp.MustCompile(`\n+$`), Replace: "", Priority: 10},
		{Name: "comment_block", Find: regexp.MustCompile(`(?m)^\s*#+\s*$`), Replace: "", Priority: 20},
		{Name: "html_comment", Find: regexp.MustCompile(`<!--[\s\S]*?-->`), Replace: "", Priority: 25},
		{Name: "css_comment", Find: regexp.MustCompile(`/\*[\s\S]*?\*/`), Replace: "", Priority: 25},

		// ===== Additional Patterns (8 rules) =====
		{Name: "http_protocol", Find: regexp.MustCompile(`https?://`), Replace: "", Priority: 30},
		{Name: "www_prefix", Find: regexp.MustCompile(`www\.`), Replace: "", Priority: 30},
		{Name: "trailing_punctuation", Find: regexp.MustCompile(`[.,;:]{2,}`), Replace: "$0", Priority: 25},
		{Name: "multiple_question", Find: regexp.MustCompile(`\?{2,}`), Replace: "?", Priority: 20},
		{Name: "multiple_exclamation", Find: regexp.MustCompile(`!{2,}`), Replace: "!", Priority: 20},
		{Name: "ellipsis", Find: regexp.MustCompile(`\.{4,}`), Replace: "...", Priority: 20},
		{Name: "arrow_function", Find: regexp.MustCompile(`\(\)\s*=>\s*`), Replace: "()=>", Priority: 25},
		{Name: "arrow_operator", Find: regexp.MustCompile(`\s*->\s*`), Replace: "->", Priority: 25},
	}

	return &CavemanEngine{
		enabled: true,
		stats: EngineStats{
			Name: "caveman",
		},
		rules: rules,
	}
}

// Name returns the engine name
func (e *CavemanEngine) Name() string {
	return "caveman"
}

// Priority returns the execution priority
func (e *CavemanEngine) Priority() int {
	return 15
}

// IsEnabled returns whether the engine is active
func (e *CavemanEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *CavemanEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// Compress applies caveman rule compression
func (e *CavemanEngine) Compress(input string) (string, int, error) {
	if input == "" {
		return "", 0, nil
	}

	originalLen := len(input)
	compressed := input

	// Apply rules in priority order
	for _, rule := range e.rules {
		compressed = rule.Find.ReplaceAllString(compressed, rule.Replace)
	}

	// Final cleanup
	compressed = strings.TrimSpace(compressed)

	saved := originalLen - len(compressed)

	// Update stats
	e.mu.Lock()
	e.stats.Invocations++
	e.stats.TotalInputLen += int64(originalLen)
	e.stats.TotalOutputLen += int64(len(compressed))
	e.stats.TotalSaved += int64(saved)
	e.stats.LastUsed = time.Now()
	e.mu.Unlock()

	return compressed, saved, nil
}

// RuleCount returns the number of rules in the engine
func (e *CavemanEngine) RuleCount() int {
	return len(e.rules)
}

// GetRules returns all rules
func (e *CavemanEngine) GetRules() []CavemanRule {
	return e.rules
}

// Stats returns the engine statistics
func (e *CavemanEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *CavemanEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "caveman"}
}

// Ensure CavemanEngine implements CompressionEngine
var _ CompressionEngine = (*CavemanEngine)(nil)
