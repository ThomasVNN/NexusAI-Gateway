package memory

import (
	"os"
	"testing"
	"time"
)

func TestMemoryTypes(t *testing.T) {
	mem := &Memory{
		ID:        "test-123",
		Type:      MemoryFactual,
		Content:   "Test content",
		SessionID: "session-1",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if mem.Type != MemoryFactual {
		t.Errorf("Expected MemoryFactual, got %s", mem.Type)
	}

	if mem.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", mem.ID)
	}
}

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()

	if opts.Limit != 10 {
		t.Errorf("Expected Limit 10, got %d", opts.Limit)
	}

	if opts.RRF_K != 60 {
		t.Errorf("Expected RRF_K 60, got %d", opts.RRF_K)
	}

	if !opts.Hybrid {
		t.Error("Expected Hybrid to be true")
	}

	if len(opts.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(opts.Sources))
	}
}

func TestMemoryMatch(t *testing.T) {
	mem := &Memory{
		ID:      "match-1",
		Type:    MemoryEpisodic,
		Content: "Test match",
	}

	match := &MemoryMatch{
		Memory: mem,
		Score:  0.95,
		Rank:   0,
		Source: "hybrid",
	}

	if match.Score != 0.95 {
		t.Errorf("Expected Score 0.95, got %f", match.Score)
	}

	if match.Source != "hybrid" {
		t.Errorf("Expected Source hybrid, got %s", match.Source)
	}
}

func TestMemoryConfig(t *testing.T) {
	config := &MemoryConfig{
		DefaultLimit: 20,
		RRF_K:       60,
		Tier0: FTS5Config{
			Enabled: true,
			Path:    "./test_fts5.db",
		},
		Tier1: VecConfig{
			Enabled:  true,
			Path:     "./test_vec.db",
			Model:    "all-MiniLM-L6-v2",
			Provider: "static",
		},
		Tier2: QdrantConfig{
			Enabled:    true,
			Endpoint:   "http://localhost:6333",
			Collection:  "test_memories",
			Dimension:  384,
		},
	}

	if config.DefaultLimit != 20 {
		t.Errorf("Expected DefaultLimit 20, got %d", config.DefaultLimit)
	}

	if !config.Tier0.Enabled {
		t.Error("Expected Tier0 to be enabled")
	}

	if config.Tier2.Dimension != 384 {
		t.Errorf("Expected Dimension 384, got %d", config.Tier2.Dimension)
	}
}

func TestMemorySerialization(t *testing.T) {
	mem := &Memory{
		ID:        "serialize-test",
		Type:      MemorySemantic,
		Content:   "Test serialization",
		Metadata:  map[string]interface{}{"key": "value"},
		SessionID: "sess-1",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	json := serializeMetadata(mem.Metadata)
	if json == "" {
		t.Error("serializeMetadata returned empty string")
	}

	mem2 := &Memory{}
	parseMetadata(json, mem2)
	if mem2.Metadata["key"] != "value" {
		t.Error("parseMetadata failed to parse key correctly")
	}

	mem3 := &Memory{}
	parseMetadata("", mem3)
	if mem3.Metadata == nil {
		t.Error("parseMetadata should initialize empty map")
	}

	parseMetadata("{}", mem3)
	if len(mem3.Metadata) != 0 {
		t.Error("parseMetadata should handle empty object")
	}
}

func TestGenerateMemoryID(t *testing.T) {
	id1 := generateMemoryID()
	id2 := generateMemoryID()

	if id1 == "" {
		t.Error("generateMemoryID should not return empty string")
	}

	if id1 == id2 {
		t.Error("generateMemoryID should generate unique IDs")
	}

	if len(id1) < 10 {
		t.Errorf("Memory ID seems too short: %s", id1)
	}
}

func TestMemoryToJSON(t *testing.T) {
	mem := &Memory{
		ID:      "json-test",
		Type:    MemoryFactual,
		Content: "Test JSON",
	}

	json := MemoryToJSON(mem)
	if json == "{}" {
		t.Error("MemoryToJSON returned empty object")
	}
}

func TestJSONToMemory(t *testing.T) {
	mem := &Memory{
		ID:      "roundtrip-test",
		Type:    MemoryEpisodic,
		Content: "Roundtrip test",
	}

	json := MemoryToJSON(mem)
	parsed, err := JSONToMemory(json)

	if err != nil {
		t.Errorf("JSONToMemory failed: %v", err)
	}

	if parsed.ID != mem.ID {
		t.Errorf("Expected ID %s, got %s", mem.ID, parsed.ID)
	}

	if parsed.Type != mem.Type {
		t.Errorf("Expected Type %s, got %s", mem.Type, parsed.Type)
	}
}

func TestEmbedderCreation(t *testing.T) {
	embedder, err := NewEmbeddingModel("auto", "", "")
	if err != nil {
		t.Logf("Embedding model init (expected to potentially fail without ONNX): %v", err)
	}

	if embedder != nil {
		if embedder.Source() != EmbeddingStatic && embedder.Source() != EmbeddingRemote && embedder.Source() != EmbeddingONNXLocal {
			t.Errorf("Expected auto to resolve, got %s", embedder.Source())
		}
	}
}

func TestEmbedStatic(t *testing.T) {
	embedder, _ := NewEmbeddingModel("static", "test-model", "")

	embedding, err := embedder.Embed("hello world")
	if err != nil {
		t.Errorf("Static embed failed: %v", err)
	}

	if len(embedding) != embedder.Dimension() {
		t.Errorf("Expected dimension %d, got %d", embedder.Dimension(), len(embedding))
	}

	embedding2, _ := embedder.Embed("hello world")
	for i := range embedding {
		if embedding[i] != embedding2[i] {
			t.Error("Static embeddings should be deterministic")
			break
		}
	}
}

func TestHashString(t *testing.T) {
	h1 := hashString("test string")
	h2 := hashString("test string")
	h3 := hashString("different string")

	if h1 != h2 {
		t.Error("Same string should produce same hash")
	}

	if h1 == h3 {
		t.Error("Different strings should produce different hashes")
	}
}

func TestRetrievalEngine(t *testing.T) {
	engine := NewRetrievalEngine(0)

	if engine.rrfK != 60 {
		t.Errorf("Expected default RRF_K of 60, got %d", engine.rrfK)
	}

	engine2 := NewRetrievalEngine(100)
	if engine2.rrfK != 100 {
		t.Errorf("Expected RRF_K of 100, got %d", engine2.rrfK)
	}

	// Note: FTS5 module may not be available in test environment
	// This test verifies the retrieval engine works with mock tiers
	engine.RegisterTier(&mockTier{name: "test"})

	tiers := engine.Tiers()
	if len(tiers) != 1 || tiers[0] != "test" {
		t.Errorf("Expected [test], got %v", tiers)
	}
}

func TestExtractionEngine(t *testing.T) {
	engine := NewExtractionEngine()

	content := "The capital of France is Paris. To make coffee, first boil water, then add grounds."
	memories := engine.ExtractMemories(content, "test-session", "test-user")

	t.Logf("Extracted %d memories", len(memories))
}

func TestExtractionEngineCustomPattern(t *testing.T) {
	engine := NewExtractionEngine()

	err := engine.AddPattern(MemoryFactual, `custom:\s*([^\n]+)`, 1.0)
	if err != nil {
		t.Errorf("Failed to add custom pattern: %v", err)
	}

	content := "Remember that custom: the sky is blue"
	memories := engine.ExtractMemories(content, "test-session", "test-user")

	t.Logf("Extracted %d memories with custom pattern", len(memories))
}

func TestMain(m *testing.M) {
	code := m.Run()

	os.Remove("./test_fts5.db")
	os.Remove("./test_vec.db")
	os.Remove("./test_tiers.db")

	os.Exit(code)
}
