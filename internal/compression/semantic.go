package compression

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// SemanticConfig contains compression configuration
type SemanticConfig struct {
	TargetRatio     float64 // Target compression ratio (0.0-1.0)
	PreserveMeaning bool    // Preserve semantic meaning
	MinTokens       int     // Minimum tokens to compress
	MaxTokens       int     // Maximum tokens after compression
	Strategy       string  // "aggressive", "balanced", "conservative"
}

// SemanticResult contains semantic compression results
type SemanticResult struct {
	Original         string
	Compressed       string
	OriginalTokens   int
	CompressedTokens int
	Ratio            float64
	Savings          int // bytes saved
	Preserved        bool
}

// SemanticCompressionResult is the return type for semantic compression
type SemanticCompressionResult = SemanticResult

// SemanticCompressor provides semantic compression
type SemanticCompressor struct {
	config            SemanticConfig
	preservedKeywords []string
	seenNgrams        map[string]int
	stats             *SemanticStats
}

// SemanticStats tracks compression statistics
type SemanticStats struct {
	TotalCompressed int64
	TotalOriginal   int64
	TotalSavings    int64
	Operations      int64
}

// NewSemanticCompressor creates a new semantic compressor
func NewSemanticCompressor(config SemanticConfig) *SemanticCompressor {
	if config.MinTokens == 0 {
		config.MinTokens = 50
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.TargetRatio == 0 {
		config.TargetRatio = 0.5
	}
	if config.Strategy == "" {
		config.Strategy = "balanced"
	}
	return &SemanticCompressor{
		config:            config,
		preservedKeywords: []string{},
		seenNgrams:        make(map[string]int),
		stats:             &SemanticStats{},
	}
}

// Compress compresses text while preserving meaning
func (c *SemanticCompressor) Compress(ctx context.Context, text string) (*SemanticResult, error) {
	if text == "" {
		return &SemanticResult{
			Original:        "",
			Compressed:      "",
			OriginalTokens:  0,
			Ratio:          0,
			Savings:        0,
			Preserved:      true,
		}, nil
	}

	originalTokens := c.EstimateTokens(text)
	
	// Skip compression if below minimum tokens
	if originalTokens < c.config.MinTokens {
		return &SemanticResult{
			Original:         text,
			Compressed:       text,
			OriginalTokens:   originalTokens,
			CompressedTokens: originalTokens,
			Ratio:           1.0,
			Savings:         0,
			Preserved:       true,
		}, nil
	}

	// Split into semantic chunks
	chunks := c.SplitByMeaning(text)
	
	// Remove redundancy
	filteredChunks := c.RemoveRedundancy(chunks)
	
	// Merge related chunks
	mergedChunks := c.MergeChunks(filteredChunks)
	
	// Build compressed text from chunks
	var compressed strings.Builder
	for _, chunk := range mergedChunks {
		// Check if this chunk contains preserved keywords
		isPreserved := c.shouldPreserve(chunk.Text)
		
		if isPreserved {
			compressed.WriteString(chunk.Text)
		} else {
			// Apply LZ-based compression for non-preserved chunks
			compressed.WriteString(c.compressChunk(chunk.Text))
		}
		compressed.WriteString(" ")
	}
	
	result := compressed.String()
	
	// Apply semantic-aware compression based on strategy
	result = c.applyStrategy(result)
	
	// If target ratio not met, apply additional compression
	if c.config.TargetRatio > 0 {
		currentRatio := float64(len(result)) / float64(len(text))
		if currentRatio > c.config.TargetRatio {
			result = c.applyTargetRatioCompression(result, text)
		}
	}
	
	compressedTokens := c.EstimateTokens(result)
	savings := len(text) - len(result)
	
	// Ensure compressed tokens don't exceed max
	if compressedTokens > c.config.MaxTokens {
		result = c.truncateToMaxTokens(result, compressedTokens)
		compressedTokens = c.EstimateTokens(result)
		savings = len(text) - len(result)
	}
	
	return &SemanticResult{
		Original:         text,
		Compressed:       result,
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		Ratio:           float64(len(result)) / float64(len(text)),
		Savings:         savings,
		Preserved:       c.config.PreserveMeaning && c.verifyMeaningPreserved(text, result),
	}, nil
}

// Decompress restores compressed text
func (c *SemanticCompressor) Decompress(ctx context.Context, compressed string) (string, error) {
	if compressed == "" {
		return "", nil
	}
	
	// Check if this is an LZ-encoded string (base64 prefix marker)
	if strings.HasPrefix(compressed, "LZ:") {
		encoded := strings.TrimPrefix(compressed, "LZ:")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", err
		}
		
		// Decompress using flate
		r := flate.NewReader(bytes.NewReader(decoded))
		defer r.Close()
		
		result := new(strings.Builder)
		buf := make([]byte, 256)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				result.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		return result.String(), nil
	}
	
	return compressed, nil
}

// SplitByMeaning splits text into semantic chunks
func (c *SemanticCompressor) SplitByMeaning(text string) []SemanticChunk {
	var chunks []SemanticChunk
	
	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")
	currentPos := 0
	
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			currentPos += 2
			continue
		}
		
		chunkType := c.detectChunkType(para)
		
		// Check for code blocks
		if strings.Contains(para, "```") || strings.Contains(para, "    ") {
			chunkType = "code"
		}
		
		// Check for lists
		if matched, _ := regexp.MatchString(`^[\-\*]\s`, para); matched {
			chunkType = "list"
		}
		
		chunks = append(chunks, SemanticChunk{
			Text:     para,
			Type:     chunkType,
			StartPos: currentPos,
			EndPos:   currentPos + len(para),
		})
		
		currentPos += len(para) + 2
	}
	
	// If no paragraphs found, split by sentences
	if len(chunks) == 0 {
		sentences := c.splitBySentences(text)
		currentPos = 0
		for _, sent := range sentences {
			if sent == "" {
				continue
			}
			chunks = append(chunks, SemanticChunk{
				Text:     sent,
				Type:     "sentence",
				StartPos: currentPos,
				EndPos:   currentPos + len(sent),
			})
			currentPos += len(sent) + 1
		}
	}
	
	return chunks
}

// SemanticChunk represents a semantically coherent text chunk
type SemanticChunk struct {
	Text     string
	Type     string // "sentence", "paragraph", "code", "list"
	StartPos int
	EndPos   int
}

// RemoveRedundancy removes redundant content
func (c *SemanticCompressor) RemoveRedundancy(chunks []SemanticChunk) []SemanticChunk {
	var result []SemanticChunk
	
	// Track seen trigrams for deduplication
	seenTrigrams := make(map[string]bool)
	
	for _, chunk := range chunks {
		// Always preserve code chunks
		if chunk.Type == "code" {
			result = append(result, chunk)
			continue
		}
		
		// Check for semantic redundancy using trigrams
		trigrams := c.extractTrigrams(chunk.Text)
		isRedundant := true
		
		for _, trigram := range trigrams {
			if !seenTrigrams[trigram] {
				isRedundant = false
				break
			}
		}
		
		if !isRedundant {
			result = append(result, chunk)
			for _, trigram := range trigrams {
				seenTrigrams[trigram] = true
			}
		}
	}
	
	return result
}

// MergeChunks merges related chunks
func (c *SemanticCompressor) MergeChunks(chunks []SemanticChunk) []SemanticChunk {
	if len(chunks) <= 1 {
		return chunks
	}
	
	var result []SemanticChunk
	var currentGroup []SemanticChunk
	
	for _, chunk := range chunks {
		// Start a new group
		if len(currentGroup) == 0 {
			currentGroup = append(currentGroup, chunk)
			continue
		}
		
		// Check if chunk should merge with current group
		lastChunk := currentGroup[len(currentGroup)-1]
		
		// Merge if same type and within size limit
		mergedLen := 0
		for _, g := range currentGroup {
			mergedLen += len(g.Text)
		}
		
		shouldMerge := chunk.Type == lastChunk.Type && 
			mergedLen+len(chunk.Text) < 500 &&
			c.areRelated(chunk, lastChunk)
		
		if shouldMerge {
			currentGroup = append(currentGroup, chunk)
		} else {
			// Finalize current group
			result = append(result, c.mergeGroup(currentGroup)...)
			currentGroup = []SemanticChunk{chunk}
		}
	}
	
	// Don't forget the last group
	if len(currentGroup) > 0 {
		result = append(result, c.mergeGroup(currentGroup)...)
	}
	
	return result
}

// Tokenize splits text into tokens
func (c *SemanticCompressor) Tokenize(text string) []Token {
	var tokens []Token
	position := 0
	
	// Regex patterns for different token types
	wordPattern := regexp.MustCompile(`[a-zA-Z0-9_]+`)
	codePattern := regexp.MustCompile(`[\{\}\[\]\(\)\.<>]+|[a-zA-Z_][a-zA-Z0-9_]*\([^\)]*\)`)
	punctPattern := regexp.MustCompile(`[^\s\w]`)
	spacePattern := regexp.MustCompile(`\s+`)
	
	// Track what's been matched
	matched := make(map[int]bool)
	
	// Match words
	for _, m := range wordPattern.FindAllStringIndex(text, -1) {
		for i := m[0]; i < m[1]; i++ {
			matched[i] = true
		}
		tokens = append(tokens, Token{
			Text:     text[m[0]:m[1]],
			Type:     TokenWord,
			Position: m[0],
		})
	}
	
	// Match code elements
	for _, m := range codePattern.FindAllStringIndex(text, -1) {
		for i := m[0]; i < m[1]; i++ {
			matched[i] = true
		}
		tokens = append(tokens, Token{
			Text:     text[m[0]:m[1]],
			Type:     TokenCode,
			Position: m[0],
		})
	}
	
	// Match punctuation
	for _, m := range punctPattern.FindAllStringIndex(text, -1) {
		// Skip if already matched
		if matched[m[0]] {
			continue
		}
		tokens = append(tokens, Token{
			Text:     text[m[0]:m[1]],
			Type:     TokenPunctuation,
			Position: m[0],
		})
	}
	
	// Match whitespace
	for _, m := range spacePattern.FindAllStringIndex(text, -1) {
		for i := m[0]; i < m[1]; i++ {
			matched[i] = true
		}
		tokens = append(tokens, Token{
			Text:     text[m[0]:m[1]],
			Type:     TokenWhitespace,
			Position: m[0],
		})
	}
	
	// Sort tokens by position
	sortTokensByPosition(tokens)
	
	_ = position // suppress unused warning
	
	return tokens
}

// Token represents a text token
type Token struct {
	Text     string
	Type     TokenType
	Position int
}

// TokenType represents token types
type TokenType int

const (
	TokenWord TokenType = iota
	TokenPunctuation
	TokenWhitespace
	TokenCode
)

// EstimateTokens estimates token count
func (c *SemanticCompressor) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	
	// Simple tokenizer-based estimation
	tokens := c.Tokenize(text)
	
	// Filter out whitespace-only tokens
	count := 0
	for _, t := range tokens {
		if t.Type != TokenWhitespace && strings.TrimSpace(t.Text) != "" {
			count++
		}
	}
	
	// Approximate: 4 chars per token on average
	roughEstimate := len(text) / 4
	if roughEstimate < count {
		return count
	}
	return roughEstimate
}

// PreservedKeywords returns keywords that must be preserved
func (c *SemanticCompressor) PreservedKeywords() []string {
	return c.preservedKeywords
}

// SetPreservedKeywords sets keywords to preserve during compression
func (c *SemanticCompressor) SetPreservedKeywords(keywords []string) {
	c.preservedKeywords = keywords
}

// Helper methods

func (c *SemanticCompressor) detectChunkType(text string) string {
	// Check for code patterns
	if strings.Contains(text, "```") || 
	   strings.Contains(text, "function ") ||
	   strings.Contains(text, "def ") ||
	   strings.Contains(text, "class ") ||
	   regexp.MustCompile(`^\s{4}`).MatchString(text) {
		return "code"
	}
	
	// Check for list patterns
	if matched, _ := regexp.MatchString(`^[\-\*\d]+\.?\s`, strings.TrimSpace(text)); matched {
		return "list"
	}
	
	// Check for sentence
	if matched, _ := regexp.MatchString(`[.!?]\s*$`, strings.TrimSpace(text)); matched {
		return "sentence"
	}
	
	return "paragraph"
}

func (c *SemanticCompressor) splitBySentences(text string) []string {
	// Split on sentence-ending punctuation followed by space or end
	pattern := regexp.MustCompile(`[^.!?]*[.!?]+\s*`)
	matches := pattern.FindAllStringIndex(text, -1)
	
	var sentences []string
	end := 0
	for _, m := range matches {
		if m[0] > end {
			sentences = append(sentences, strings.TrimSpace(text[end:m[0]]))
		}
		sentences = append(sentences, strings.TrimSpace(text[m[0]:m[1]]))
		end = m[1]
	}
	
	// Add remaining text
	if end < len(text) {
		remaining := strings.TrimSpace(text[end:])
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}
	
	return sentences
}

func (c *SemanticCompressor) extractTrigrams(text string) []string {
	words := regexp.MustCompile(`\s+`).Split(strings.ToLower(text), -1)
	words = filterEmpty(words)
	
	var trigrams []string
	for i := 0; i < len(words)-2; i++ {
		trigram := strings.Join(words[i:i+3], " ")
		trigrams = append(trigrams, trigram)
	}
	
	return trigrams
}

func filterEmpty(ss []string) []string {
	var result []string
	for _, s := range ss {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func (c *SemanticCompressor) areRelated(a, b SemanticChunk) bool {
	// Check for shared keywords or topic similarity
	aWords := c.extractKeywords(a.Text)
	bWords := c.extractKeywords(b.Text)
	
	shared := 0
	for _, aw := range aWords {
		for _, bw := range bWords {
			if aw == bw {
				shared++
				break
			}
		}
	}
	
	// Consider related if > 20% shared keywords
	threshold := len(aWords) / 5
	return shared >= threshold && shared > 0
}

func (c *SemanticCompressor) extractKeywords(text string) []string {
	words := regexp.MustCompile(`[a-zA-Z]{4,}`).FindAllString(strings.ToLower(text), -1)
	
	// Filter common stop words
	stopWords := map[string]bool{
		"that": true, "this": true, "with": true, "from": true,
		"have": true, "been": true, "were": true, "they": true,
		"which": true, "when": true, "what": true, "where": true,
	}
	
	var keywords []string
	for _, w := range words {
		if !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	
	return keywords
}

func (c *SemanticCompressor) mergeGroup(chunks []SemanticChunk) []SemanticChunk {
	if len(chunks) == 0 {
		return chunks
	}
	
	if len(chunks) == 1 {
		return chunks
	}
	
	// Merge text while preserving metadata
	var mergedText strings.Builder
	startPos := chunks[0].StartPos
	
	for i, chunk := range chunks {
		if i > 0 {
			if chunk.Type == "list" || chunks[i-1].Type == "list" {
				mergedText.WriteString("\n")
			} else {
				mergedText.WriteString(" ")
			}
		}
		mergedText.WriteString(chunk.Text)
	}
	
	return []SemanticChunk{{
		Text:     mergedText.String(),
		Type:     chunks[0].Type,
		StartPos: startPos,
		EndPos:   startPos + len(mergedText.String()),
	}}
}

func (c *SemanticCompressor) shouldPreserve(text string) bool {
	// Check preserved keywords
	for _, keyword := range c.preservedKeywords {
		if strings.Contains(strings.ToLower(text), strings.ToLower(keyword)) {
			return true
		}
	}
	
	// Preserve short texts
	if len(text) < 50 {
		return true
	}
	
	// Preserve code
	if strings.Contains(text, "```") || strings.Contains(text, "function ") {
		return true
	}
	
	return false
}

func (c *SemanticCompressor) compressChunk(text string) string {
	// Apply LZ-style compression using flate
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	
	_, err := w.Write([]byte(text))
	if err != nil {
		return text
	}
	
	w.Close()
	
	// Encode as base64 for safe storage
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "LZ:" + encoded
}

func (c *SemanticCompressor) applyStrategy(text string) string {
	switch c.config.Strategy {
	case "aggressive":
		return c.aggressiveCompress(text)
	case "conservative":
		return c.conservativeCompress(text)
	default: // balanced
		return c.balancedCompress(text)
	}
}

func (c *SemanticCompressor) aggressiveCompress(text string) string {
	// Remove all extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	// Remove common redundant phrases
	redundant := []string{
		"in order to", "due to the fact that", "for the purpose of",
		"in the event that", "at this point in time", "at the present time",
		"it is important to note", "it should be noted",
	}
	for _, phrase := range redundant {
		text = strings.ReplaceAll(text, phrase, "")
	}
	
	// Aggressive abbreviation for common words
	abbrev := map[string]string{
		"information": "info", "configuration": "config",
		"development": "dev", "application": "app",
		"because": "bc", "without": "w/o",
	}
	for k, v := range abbrev {
		// Only abbreviate if not part of preserved keyword
		if !c.shouldPreserve(k) {
			text = strings.ReplaceAll(text, k, v)
		}
	}
	
	return strings.TrimSpace(text)
}

func (c *SemanticCompressor) conservativeCompress(text string) string {
	// Only remove excessive whitespace
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(` {2,}`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (c *SemanticCompressor) balancedCompress(text string) string {
	// Remove excessive whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	// Remove some redundant phrases
	redundant := []string{
		"in order to", "due to the fact that", "for the purpose of",
	}
	for _, phrase := range redundant {
		text = strings.ReplaceAll(text, phrase, "")
	}
	
	return strings.TrimSpace(text)
}

func (c *SemanticCompressor) applyTargetRatioCompression(text, original string) string {
	targetLen := int(float64(len(original)) * c.config.TargetRatio)
	
	if len(text) <= targetLen {
		return text
	}
	
	// Progressive compression until target is met
	iterations := 0
	maxIterations := 5
	
	for len(text) > targetLen && iterations < maxIterations {
		text = c.aggressiveCompress(text)
		
		// Fallback to LZ if still too long
		if len(text) > targetLen {
			compressed := c.compressChunk(text)
			if len(compressed) < len(text) {
				text = compressed
			}
		}
		
		iterations++
	}
	
	return text
}

func (c *SemanticCompressor) truncateToMaxTokens(text string, currentTokens int) string {
	targetTokens := c.config.MaxTokens
	ratio := float64(targetTokens) / float64(currentTokens)
	
	if ratio >= 1.0 {
		return text
	}
	
	targetLen := int(float64(len(text)) * ratio)
	if targetLen < 10 {
		targetLen = 10
	}
	
	return text[:targetLen] + "..."
}

func (c *SemanticCompressor) verifyMeaningPreserved(original, compressed string) bool {
	// Basic check: extract key subjects/verbs from both
	originalKeywords := c.extractKeywords(original)
	compressedKeywords := c.extractKeywords(compressed)
	
	if len(compressedKeywords) == 0 && len(originalKeywords) > 0 {
		return false
	}
	
	// Check overlap
	originalSet := make(map[string]bool)
	for _, k := range originalKeywords {
		originalSet[k] = true
	}
	
	matches := 0
	for _, k := range compressedKeywords {
		if originalSet[k] {
			matches++
		}
	}
	
	// At least 50% of keywords should be preserved
	threshold := len(originalKeywords) / 2
	return matches >= threshold
}

func sortTokensByPosition(tokens []Token) {
	for i := 0; i < len(tokens)-1; i++ {
		for j := i + 1; j < len(tokens); j++ {
			if tokens[i].Position > tokens[j].Position {
				tokens[i], tokens[j] = tokens[j], tokens[i]
			}
		}
	}
}

// ValidateConfig validates the semantic configuration
func ValidateConfig(config SemanticConfig) error {
	if config.TargetRatio < 0 || config.TargetRatio > 1 {
		return errors.New("TargetRatio must be between 0 and 1")
	}
	if config.MinTokens < 0 {
		return errors.New("MinTokens must be non-negative")
	}
	if config.MaxTokens < config.MinTokens {
		return errors.New("MaxTokens must be >= MinTokens")
	}
	validStrategies := map[string]bool{
		"aggressive": true, "balanced": true, "conservative": true,
	}
	if !validStrategies[config.Strategy] {
		return errors.New("Strategy must be one of: aggressive, balanced, conservative")
	}
	return nil
}

// String helpers for TokenType
func (t TokenType) String() string {
	switch t {
	case TokenWord:
		return "word"
	case TokenPunctuation:
		return "punctuation"
	case TokenWhitespace:
		return "whitespace"
	case TokenCode:
		return "code"
	default:
		return "unknown"
	}
}

// ParseStrategy parses a strategy string to SemanticConfig
func ParseStrategy(strategy string) SemanticConfig {
	config := SemanticConfig{
		Strategy:       strategy,
		PreserveMeaning: true,
	}
	
	switch strategy {
	case "aggressive":
		config.TargetRatio = 0.3
		config.MinTokens = 30
		config.MaxTokens = 2048
	case "conservative":
		config.TargetRatio = 0.7
		config.MinTokens = 100
		config.MaxTokens = 8192
	default: // balanced
		config.TargetRatio = 0.5
		config.MinTokens = 50
		config.MaxTokens = 4096
	}
	
	return config
}

// Stats returns compression statistics
type Stats struct {
	TotalCompressed int64
	TotalOriginal   int64
	TotalSavings    int64
	Operations      int64
}

// CompressWithStats compresses and tracks statistics
func (c *SemanticCompressor) CompressWithStats(ctx context.Context, text string) (*SemanticResult, *SemanticStats, error) {
	result, err := c.Compress(ctx, text)
	if err != nil {
		return nil, nil, err
	}

	stats := &SemanticStats{
		TotalOriginal:   int64(len(text)),
		TotalCompressed: int64(len(result.Compressed)),
		TotalSavings:    int64(result.Savings),
		Operations:      1,
	}

	return result, stats, nil
}

// BatchCompress compresses multiple texts
func (c *SemanticCompressor) BatchCompress(ctx context.Context, texts []string) ([]*SemanticResult, error) {
	results := make([]*SemanticResult, 0, len(texts))

	for _, text := range texts {
		result, err := c.Compress(ctx, text)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	
	return results, nil
}

// EstimateCompressionRatio estimates the compression ratio without full compression
func (c *SemanticCompressor) EstimateCompressionRatio(text string) float64 {
	originalLen := len(text)
	
	// Count semantic redundancy
	chunks := c.SplitByMeaning(text)
	uniqueChunks := c.RemoveRedundancy(chunks)
	mergedChunks := c.MergeChunks(uniqueChunks)
	
	mergedLen := 0
	for _, chunk := range mergedChunks {
		mergedLen += len(chunk.Text)
	}
	
	// Factor in strategy
	strategyMultiplier := 1.0
	switch c.config.Strategy {
	case "aggressive":
		strategyMultiplier = 0.5
	case "conservative":
		strategyMultiplier = 0.9
	default:
		strategyMultiplier = 0.7
	}
	
	estimatedCompressed := float64(mergedLen) * strategyMultiplier
	
	if originalLen == 0 {
		return 1.0
	}
	
	return estimatedCompressed / float64(originalLen)
}

// GetConfig returns the current configuration
func (c *SemanticCompressor) GetConfig() SemanticConfig {
	return c.config
}

// UpdateConfig updates the configuration
func (c *SemanticCompressor) UpdateConfig(config SemanticConfig) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	c.config = config
	return nil
}

// Reset clears the compressor state
func (c *SemanticCompressor) Reset() {
	c.seenNgrams = make(map[string]int)
}

// CreateWithDefaults creates a compressor with default settings
func CreateWithDefaults() *SemanticCompressor {
	return NewSemanticCompressor(SemanticConfig{
		TargetRatio:     0.5,
		PreserveMeaning: true,
		MinTokens:       50,
		MaxTokens:       4096,
		Strategy:        "balanced",
	})
}

// CreateAggressive creates an aggressive compressor
func CreateAggressive() *SemanticCompressor {
	return NewSemanticCompressor(ParseStrategy("aggressive"))
}

// CreateConservative creates a conservative compressor
func CreateConservative() *SemanticCompressor {
	return NewSemanticCompressor(ParseStrategy("conservative"))
}

// ChunkType returns the type of a semantic chunk
func (chunk SemanticChunk) ChunkType() string {
	return chunk.Type
}

// Length returns the length of a semantic chunk
func (chunk SemanticChunk) Length() int {
	return chunk.EndPos - chunk.StartPos
}

// ContainsPosition checks if a chunk contains a position
func (chunk SemanticChunk) ContainsPosition(pos int) bool {
	return pos >= chunk.StartPos && pos <= chunk.EndPos
}

// WordCount returns the approximate word count of a chunk
func (chunk SemanticChunk) WordCount() int {
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(chunk.Text), -1)
	return len(filterEmpty(words))
}

// IsCode checks if a chunk is code
func (chunk SemanticChunk) IsCode() bool {
	return chunk.Type == "code"
}

// IsList checks if a chunk is a list
func (chunk SemanticChunk) IsList() bool {
	return chunk.Type == "list"
}

// CompressionMetrics holds compression metrics
type CompressionMetrics struct {
	OriginalLen     int
	CompressedLen   int
	TokensOriginal  int
	TokensCompressed int
	Ratio           float64
	SavingsPercent  float64
}

// ToMetrics converts a SemanticResult to CompressionMetrics
func (r *SemanticResult) ToMetrics() CompressionMetrics {
	return CompressionMetrics{
		OriginalLen:       len(r.Original),
		CompressedLen:     len(r.Compressed),
		TokensOriginal:    r.OriginalTokens,
		TokensCompressed:  r.CompressedTokens,
		Ratio:             r.Ratio,
		SavingsPercent:    float64(r.Savings) / float64(len(r.Original)) * 100,
	}
}

// FormatMetrics formats metrics as a string
func (m CompressionMetrics) FormatMetrics() string {
	return "Original: " + strconv.Itoa(m.OriginalLen) + 
		" bytes (" + strconv.Itoa(m.TokensOriginal) + " tokens), " +
		"Compressed: " + strconv.Itoa(m.CompressedLen) + 
		" bytes (" + strconv.Itoa(m.TokensCompressed) + " tokens), " +
		"Ratio: " + strconv.FormatFloat(m.Ratio, 'f', 2, 64) + 
		", Savings: " + strconv.FormatFloat(m.SavingsPercent, 'f', 1, 64) + "%"
}
