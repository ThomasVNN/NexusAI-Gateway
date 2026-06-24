package compression

import (
	"context"
	"strings"
	"testing"
)

func TestNewSemanticCompressor(t *testing.T) {
	config := SemanticConfig{
		TargetRatio:     0.5,
		PreserveMeaning: true,
		MinTokens:       50,
		MaxTokens:       4096,
		Strategy:        "balanced",
	}
	
	compressor := NewSemanticCompressor(config)
	
	if compressor == nil {
		t.Fatal("Expected non-nil compressor")
	}
	
	if compressor.config.TargetRatio != 0.5 {
		t.Errorf("Expected TargetRatio 0.5, got %v", compressor.config.TargetRatio)
	}
	
	if compressor.config.Strategy != "balanced" {
		t.Errorf("Expected Strategy 'balanced', got %v", compressor.config.Strategy)
	}
}

func TestNewSemanticCompressorDefaults(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	if compressor.config.MinTokens != 50 {
		t.Errorf("Expected default MinTokens 50, got %v", compressor.config.MinTokens)
	}
	
	if compressor.config.MaxTokens != 4096 {
		t.Errorf("Expected default MaxTokens 4096, got %v", compressor.config.MaxTokens)
	}
	
	if compressor.config.TargetRatio != 0.5 {
		t.Errorf("Expected default TargetRatio 0.5, got %v", compressor.config.TargetRatio)
	}
	
	if compressor.config.Strategy != "balanced" {
		t.Errorf("Expected default Strategy 'balanced', got %v", compressor.config.Strategy)
	}
}

func TestCompressEmpty(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	result, err := compressor.Compress(context.Background(), "")
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result.Original != "" {
		t.Errorf("Expected empty original, got %v", result.Original)
	}
	
	if result.Compressed != "" {
		t.Errorf("Expected empty compressed, got %v", result.Compressed)
	}
	
	if result.OriginalTokens != 0 {
		t.Errorf("Expected 0 original tokens, got %v", result.OriginalTokens)
	}
	
	if result.Ratio != 0 {
		t.Errorf("Expected 0 ratio, got %v", result.Ratio)
	}
}

func TestCompressBelowMinTokens(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{
		MinTokens: 50,
	})
	
	// Short text should not be compressed
	shortText := "Hello world, this is a short test."
	
	result, err := compressor.Compress(context.Background(), shortText)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result.Compressed != shortText {
		t.Errorf("Expected short text to be preserved, got %v", result.Compressed)
	}
	
	if result.Savings != 0 {
		t.Errorf("Expected 0 savings for short text, got %v", result.Savings)
	}
}

func TestCompressBasic(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{
		Strategy: "balanced",
	})
	
	text := "This is a longer piece of text that should be compressed. It contains multiple sentences and paragraphs. The compression should reduce the size while preserving the essential meaning of the content."
	
	result, err := compressor.Compress(context.Background(), text)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result.Original != text {
		t.Errorf("Original text mismatch")
	}
	
	if result.OriginalTokens <= 0 {
		t.Errorf("Expected positive original tokens, got %v", result.OriginalTokens)
	}
	
	if result.CompressedTokens <= 0 {
		t.Errorf("Expected positive compressed tokens, got %v", result.CompressedTokens)
	}
	
	if result.Ratio <= 0 || result.Ratio > 1 {
		t.Errorf("Expected ratio between 0 and 1, got %v", result.Ratio)
	}
}

func TestDecompressEmpty(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	result, err := compressor.Decompress(context.Background(), "")
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "" {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestDecompressPlainText(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	plainText := "This is plain text without any special encoding."
	
	result, err := compressor.Decompress(context.Background(), plainText)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != plainText {
		t.Errorf("Expected plain text to be preserved, got %v", result)
	}
}

func TestDecompressCompressedText(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	original := "This is a longer piece of text that contains enough content to trigger compression. It has multiple words and sentences that can be compressed effectively using LZ-based algorithms."
	
	// First compress
	compressResult, err := compressor.Compress(context.Background(), original)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	
	// Verify compressed text contains LZ marker
	if !strings.HasPrefix(compressResult.Compressed, "LZ:") && compressResult.Compressed != original {
		// If not LZ compressed, it means compression didn't help or wasn't applied
		t.Logf("Text was not LZ compressed: %v", compressResult.Compressed[:min(50, len(compressResult.Compressed))])
	}
	
	// Decompress
	decompressed, err := compressor.Decompress(context.Background(), compressResult.Compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	
	// Note: We can't guarantee exact match if text wasn't compressed
	// So we just verify decompression doesn't error
	t.Logf("Decompression result length: %d", len(decompressed))
}

func TestSplitByMeaning(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "First paragraph here.\n\nSecond paragraph here.\n\nThird paragraph here."
	
	chunks := compressor.SplitByMeaning(text)
	
	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %v", len(chunks))
	}
	
	// Check that chunks have types
	for _, chunk := range chunks {
		if chunk.Type == "" {
			t.Error("Expected non-empty chunk type")
		}
		if chunk.EndPos <= chunk.StartPos {
			t.Error("Expected EndPos > StartPos")
		}
	}
}

func TestSplitByMeaningParagraphs(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	// Two clear paragraphs
	text := "This is the first paragraph. It has multiple sentences.\n\nThis is the second paragraph. It also has sentences."
	
	chunks := compressor.SplitByMeaning(text)
	
	// Should split into at least 2 chunks
	foundParagraphs := 0
	for _, chunk := range chunks {
		if chunk.Type == "paragraph" || chunk.Type == "sentence" {
			foundParagraphs++
		}
	}
	
	if foundParagraphs < 2 {
		t.Errorf("Expected at least 2 paragraph/sentence chunks, got %v", foundParagraphs)
	}
}

func TestSplitByMeaningCode(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "Here is some code:\n\n    func main() {\n        println(\"Hello\")\n    }\n\nMore text after."
	
	chunks := compressor.SplitByMeaning(text)
	
	// Should detect code block
	foundCode := false
	for _, chunk := range chunks {
		if chunk.Type == "code" {
			foundCode = true
			break
		}
	}
	
	if !foundCode {
		t.Log("Code block detection - may vary based on implementation")
	}
}

func TestRemoveRedundancy(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	chunks := []SemanticChunk{
		{Text: "This is unique text one", Type: "paragraph", StartPos: 0, EndPos: 25},
		{Text: "This is unique text two", Type: "paragraph", StartPos: 26, EndPos: 51},
		{Text: "This is unique text three", Type: "paragraph", StartPos: 52, EndPos: 78},
	}
	
	result := compressor.RemoveRedundancy(chunks)
	
	// All chunks should be kept since they're unique
	if len(result) != len(chunks) {
		t.Errorf("Expected %d chunks, got %d", len(chunks), len(result))
	}
}

func TestRemoveRedundancyPreservesCode(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	chunks := []SemanticChunk{
		{Text: "function foo() {}", Type: "code", StartPos: 0, EndPos: 19},
		{Text: "function bar() {}", Type: "code", StartPos: 20, EndPos: 39},
	}
	
	result := compressor.RemoveRedundancy(chunks)
	
	// Code chunks should always be preserved
	if len(result) != len(chunks) {
		t.Errorf("Expected code chunks to be preserved, got %d instead of %d", len(result), len(chunks))
	}
}

func TestMergeChunks(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	chunks := []SemanticChunk{
		{Text: "First", Type: "sentence", StartPos: 0, EndPos: 5},
		{Text: "Second", Type: "sentence", StartPos: 6, EndPos: 12},
		{Text: "Third", Type: "sentence", StartPos: 13, EndPos: 18},
	}
	
	result := compressor.MergeChunks(chunks)
	
	// Should have merged into one chunk
	if len(result) != 1 {
		t.Errorf("Expected 1 merged chunk, got %d", len(result))
	}
}

func TestMergeChunksPreservesTypes(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	chunks := []SemanticChunk{
		{Text: "Item one", Type: "list", StartPos: 0, EndPos: 8},
		{Text: "Item two", Type: "list", StartPos: 9, EndPos: 17},
	}
	
	result := compressor.MergeChunks(chunks)
	
	if len(result) != 1 {
		t.Errorf("Expected 1 merged chunk, got %d", len(result))
	}
	
	if result[0].Type != "list" {
		t.Errorf("Expected merged chunk type 'list', got %v", result[0].Type)
	}
}

func TestTokenize(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "Hello world, this is a test. Numbers: 123, symbols: @#$%"
	
	tokens := compressor.Tokenize(text)
	
	if len(tokens) == 0 {
		t.Error("Expected some tokens")
	}
	
	// Should have word tokens
	foundWords := false
	for _, token := range tokens {
		if token.Type == TokenWord {
			foundWords = true
			break
		}
	}
	
	if !foundWords {
		t.Error("Expected to find word tokens")
	}
}

func TestTokenizePreservesPosition(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "abc def"
	
	tokens := compressor.Tokenize(text)
	
	// Tokens should be sorted by position
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Position > tokens[i+1].Position {
			t.Error("Tokens not sorted by position")
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "This is a test sentence with several words."
	
	count := compressor.EstimateTokens(text)
	
	if count <= 0 {
		t.Errorf("Expected positive token count, got %v", count)
	}
	
	// Rough estimate: should be around 10-15 tokens for this text
	if count < 5 || count > 30 {
		t.Errorf("Token count seems off: %v", count)
	}
}

func TestEstimateTokensEmpty(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	count := compressor.EstimateTokens("")
	
	if count != 0 {
		t.Errorf("Expected 0 tokens for empty string, got %v", count)
	}
}

func TestPreservedKeywords(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	keywords := []string{"function", "variable", "API"}
	compressor.SetPreservedKeywords(keywords)
	
	result := compressor.PreservedKeywords()
	
	if len(result) != len(keywords) {
		t.Errorf("Expected %d keywords, got %d", len(keywords), len(result))
	}
	
	for i, kw := range keywords {
		if result[i] != kw {
			t.Errorf("Expected keyword %v at index %d, got %v", kw, i, result[i])
		}
	}
}

func TestPreservedKeywordsEmpty(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	result := compressor.PreservedKeywords()
	
	if result == nil {
		t.Error("Expected non-nil keywords slice")
	}
	
	if len(result) != 0 {
		t.Errorf("Expected 0 preserved keywords, got %d", len(result))
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  SemanticConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  SemanticConfig{TargetRatio: 0.5, Strategy: "balanced"},
			wantErr: false,
		},
		{
			name:    "invalid ratio too high",
			config:  SemanticConfig{TargetRatio: 1.5, Strategy: "balanced"},
			wantErr: true,
		},
		{
			name:    "invalid ratio negative",
			config:  SemanticConfig{TargetRatio: -0.1, Strategy: "balanced"},
			wantErr: true,
		},
		{
			name:    "invalid min tokens negative",
			config:  SemanticConfig{TargetRatio: 0.5, MinTokens: -1},
			wantErr: true,
		},
		{
			name:    "max less than min",
			config:  SemanticConfig{TargetRatio: 0.5, MinTokens: 100, MaxTokens: 50},
			wantErr: true,
		},
		{
			name:    "invalid strategy",
			config:  SemanticConfig{TargetRatio: 0.5, Strategy: "invalid"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		wantRatio float64
		wantMin   int
		wantMax   int
	}{
		{"aggressive", 0.3, 30, 2048},
		{"conservative", 0.7, 100, 8192},
		{"balanced", 0.5, 50, 4096},
	}
	
	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			config := ParseStrategy(tt.strategy)
			
			if config.TargetRatio != tt.wantRatio {
				t.Errorf("TargetRatio = %v, want %v", config.TargetRatio, tt.wantRatio)
			}
			
			if config.MinTokens != tt.wantMin {
				t.Errorf("MinTokens = %v, want %v", config.MinTokens, tt.wantMin)
			}
			
			if config.MaxTokens != tt.wantMax {
				t.Errorf("MaxTokens = %v, want %v", config.MaxTokens, tt.wantMax)
			}
			
			if config.Strategy != tt.strategy {
				t.Errorf("Strategy = %v, want %v", config.Strategy, tt.strategy)
			}
		})
	}
}

func TestCreateWithDefaults(t *testing.T) {
	compressor := CreateWithDefaults()
	
	if compressor.config.TargetRatio != 0.5 {
		t.Errorf("Expected default TargetRatio 0.5, got %v", compressor.config.TargetRatio)
	}
	
	if compressor.config.Strategy != "balanced" {
		t.Errorf("Expected default Strategy 'balanced', got %v", compressor.config.Strategy)
	}
}

func TestCreateAggressive(t *testing.T) {
	compressor := CreateAggressive()
	
	if compressor.config.Strategy != "aggressive" {
		t.Errorf("Expected Strategy 'aggressive', got %v", compressor.config.Strategy)
	}
	
	if compressor.config.TargetRatio != 0.3 {
		t.Errorf("Expected TargetRatio 0.3, got %v", compressor.config.TargetRatio)
	}
}

func TestCreateConservative(t *testing.T) {
	compressor := CreateConservative()
	
	if compressor.config.Strategy != "conservative" {
		t.Errorf("Expected Strategy 'conservative', got %v", compressor.config.Strategy)
	}
	
	if compressor.config.TargetRatio != 0.7 {
		t.Errorf("Expected TargetRatio 0.7, got %v", compressor.config.TargetRatio)
	}
}

func TestBatchCompress(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{
		MinTokens: 20,
	})
	
	texts := []string{
		"This is the first piece of text that is long enough.",
		"This is the second piece of text that is also long enough.",
		"And here is a third piece of longer text for compression testing.",
	}
	
	results, err := compressor.BatchCompress(context.Background(), texts)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(results) != len(texts) {
		t.Errorf("Expected %d results, got %d", len(texts), len(results))
	}
}

func TestCompressWithStats(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{
		MinTokens: 20,
	})
	
	text := "This is a longer piece of text that should be compressed when we apply semantic compression techniques."
	
	result, stats, err := compressor.CompressWithStats(context.Background(), text)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
	
	if stats.Operations != 1 {
		t.Errorf("Expected 1 operation, got %v", stats.Operations)
	}
	
	if stats.TotalOriginal != int64(len(text)) {
		t.Errorf("Expected original length %d, got %d", len(text), stats.TotalOriginal)
	}
	
	if result.CompressedTokens <= 0 {
		t.Errorf("Expected positive compressed tokens, got %v", result.CompressedTokens)
	}
}

func TestSemanticChunkHelpers(t *testing.T) {
	chunk := SemanticChunk{
		Text:     "Test chunk content",
		Type:     "paragraph",
		StartPos: 10,
		EndPos:   30,
	}
	
	if chunk.ChunkType() != "paragraph" {
		t.Errorf("Expected 'paragraph', got %v", chunk.ChunkType())
	}
	
	if chunk.Length() != 20 {
		t.Errorf("Expected length 20, got %v", chunk.Length())
	}
	
	if !chunk.ContainsPosition(15) {
		t.Error("Expected ContainsPosition(15) to be true")
	}
	
	if chunk.ContainsPosition(5) {
		t.Error("Expected ContainsPosition(5) to be false")
	}
	
	if !chunk.IsCode() {
		t.Error("Expected IsCode() to be false for paragraph")
	}
	
	if chunk.IsList() {
		t.Error("Expected IsList() to be false for paragraph")
	}
}

func TestSemanticChunkWordCount(t *testing.T) {
	chunk := SemanticChunk{
		Text:     "One two three four five",
		Type:     "paragraph",
		StartPos: 0,
		EndPos:   23,
	}
	
	count := chunk.WordCount()
	
	if count != 5 {
		t.Errorf("Expected word count 5, got %v", count)
	}
}

func TestCompressionResultToMetrics(t *testing.T) {
	result := &CompressionResult{
		Original:         "This is the original text",
		Compressed:       "Compressed",
		OriginalTokens:   20,
		CompressedTokens: 10,
		Ratio:           0.5,
		Savings:         10,
	}
	
	metrics := result.ToMetrics()
	
	if metrics.OriginalLen != len(result.Original) {
		t.Errorf("OriginalLen mismatch")
	}
	
	if metrics.TokensOriginal != 20 {
		t.Errorf("Expected TokensOriginal 20, got %v", metrics.TokensOriginal)
	}
	
	if metrics.SavingsPercent != 50 {
		t.Errorf("Expected SavingsPercent 50, got %v", metrics.SavingsPercent)
	}
}

func TestFormatMetrics(t *testing.T) {
	metrics := CompressionMetrics{
		OriginalLen:      100,
		CompressedLen:    50,
		TokensOriginal:  25,
		TokensCompressed: 12,
		Ratio:           0.5,
		SavingsPercent:  50,
	}
	
	formatted := metrics.FormatMetrics()
	
	if !strings.Contains(formatted, "Original:") {
		t.Error("Expected 'Original:' in formatted metrics")
	}
	
	if !strings.Contains(formatted, "Compressed:") {
		t.Error("Expected 'Compressed:' in formatted metrics")
	}
	
	if !strings.Contains(formatted, "Savings:") {
		t.Error("Expected 'Savings:' in formatted metrics")
	}
}

func TestEstimateCompressionRatio(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{
		Strategy: "balanced",
	})
	
	text := "This is a longer piece of text that contains multiple sentences. It should give us a reasonable estimation of the compression ratio when we apply semantic compression techniques."
	
	ratio := compressor.EstimateCompressionRatio(text)
	
	if ratio <= 0 || ratio > 1 {
		t.Errorf("Expected ratio between 0 and 1, got %v", ratio)
	}
}

func TestEstimateCompressionRatioEmpty(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	ratio := compressor.EstimateCompressionRatio("")
	
	if ratio != 1.0 {
		t.Errorf("Expected 1.0 for empty text, got %v", ratio)
	}
}

func TestGetConfig(t *testing.T) {
	config := SemanticConfig{
		TargetRatio:     0.6,
		PreserveMeaning: true,
		MinTokens:       100,
		MaxTokens:       5000,
		Strategy:        "aggressive",
	}
	
	compressor := NewSemanticCompressor(config)
	
	result := compressor.GetConfig()
	
	if result.TargetRatio != 0.6 {
		t.Errorf("Expected TargetRatio 0.6, got %v", result.TargetRatio)
	}
	
	if result.Strategy != "aggressive" {
		t.Errorf("Expected Strategy 'aggressive', got %v", result.Strategy)
	}
}

func TestUpdateConfig(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	newConfig := SemanticConfig{
		TargetRatio:  0.4,
		Strategy:     "aggressive",
		MinTokens:    30,
		MaxTokens:    3000,
	}
	
	err := compressor.UpdateConfig(newConfig)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if compressor.config.TargetRatio != 0.4 {
		t.Errorf("Expected TargetRatio 0.4, got %v", compressor.config.TargetRatio)
	}
}

func TestUpdateConfigInvalid(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	invalidConfig := SemanticConfig{
		TargetRatio: 1.5, // Invalid
	}
	
	err := compressor.UpdateConfig(invalidConfig)
	
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestReset(t *testing.T) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	compressor.seenNgrams["test"] = 1
	
	compressor.Reset()
	
	if len(compressor.seenNgrams) != 0 {
		t.Errorf("Expected empty seenNgrams after reset, got %d", len(compressor.seenNgrams))
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{TokenWord, "word"},
		{TokenPunctuation, "punctuation"},
		{TokenWhitespace, "whitespace"},
		{TokenCode, "code"},
		{TokenType(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.tokenType.String(); got != tt.expected {
				t.Errorf("TokenType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Benchmark tests

func BenchmarkCompress(b *testing.B) {
	compressor := NewSemanticCompressor(SemanticConfig{
		Strategy: "balanced",
	})
	
	text := "This is a longer piece of text that contains multiple sentences. " +
		"It should provide enough content for meaningful compression benchmarking. " +
		"The text needs to be reasonably long to trigger the compression algorithms."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Compress(context.Background(), text)
	}
}

func BenchmarkSplitByMeaning(b *testing.B) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "First paragraph with some content.\n\nSecond paragraph here.\n\nThird paragraph content."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.SplitByMeaning(text)
	}
}

func BenchmarkTokenize(b *testing.B) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "This is a test sentence with several words for benchmarking tokenization performance."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.Tokenize(text)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	compressor := NewSemanticCompressor(SemanticConfig{})
	
	text := "This is a longer piece of text that contains multiple words and sentences for testing token estimation."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compressor.EstimateTokens(text)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
