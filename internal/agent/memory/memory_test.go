package memory

import (
	"context"
	"testing"
	"time"
)

func TestMemory_NewMemory(t *testing.T) {
	memory := NewMemory(MemoryTypeSemantic, "test content", []float32{0.1, 0.2, 0.3})

	if memory.ID == "" {
		t.Error("Expected memory ID to be generated")
	}
	if memory.Type != MemoryTypeSemantic {
		t.Errorf("Expected type %s, got %s", MemoryTypeSemantic, memory.Type)
	}
	if memory.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", memory.Content)
	}
	if len(memory.Embedding) != 3 {
		t.Errorf("Expected embedding length 3, got %d", len(memory.Embedding))
	}
}

func TestMemory_WithMetadata(t *testing.T) {
	memory := NewMemory(MemoryTypeEpisodic, "test", nil)
	memory.WithMetadata("key1", "value1")
	memory.WithMetadata("key2", 42)

	if memory.Metadata["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got '%v'", memory.Metadata["key1"])
	}
	if memory.Metadata["key2"] != 42 {
		t.Errorf("Expected key2 to be 42, got '%v'", memory.Metadata["key2"])
	}
}

func TestMemory_WithSessionID(t *testing.T) {
	memory := NewMemory(MemoryTypeEpisodic, "test", nil)
	memory.WithSessionID("session-123")

	if memory.Metadata["session_id"] != "session-123" {
		t.Errorf("Expected session_id to be 'session-123', got '%v'", memory.Metadata["session_id"])
	}
}

func TestMemory_WithTTL(t *testing.T) {
	memory := NewMemory(MemoryTypeSemantic, "test", nil)
	memory.WithTTL(24 * time.Hour)

	if memory.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}

	if memory.IsExpired() {
		t.Error("Memory should not be expired immediately after creation")
	}

	pastMemory := NewMemory(MemoryTypeSemantic, "test", nil)
	pastTime := time.Now().Add(-1 * time.Hour)
	pastMemory.ExpiresAt = &pastTime

	if !pastMemory.IsExpired() {
		t.Error("Memory should be expired when ExpiresAt is in the past")
	}
}

func TestStore_Store(t *testing.T) {
	store := NewStore(time.Hour)

	memory := NewMemory(MemoryTypeSemantic, "test content", []float32{0.1, 0.2, 0.3})
	err := store.Store(context.Background(), *memory)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	retrieved, err := store.GetByID(context.Background(), memory.ID)
	if err != nil {
		t.Errorf("Expected no error retrieving memory, got %v", err)
	}
	if retrieved.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", retrieved.Content)
	}
}

func TestStore_StoreEpisodic(t *testing.T) {
	store := NewStore(time.Hour)

	memory := NewMemory(MemoryTypeEpisodic, "session event", nil)
	memory.WithSessionID("session-456")
	err := store.Store(context.Background(), *memory)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	events, err := store.GetSessionEvents(context.Background(), "session-456")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestStore_RetrieveByEmbedding(t *testing.T) {
	store := NewStore(time.Hour)

	embedding1 := []float32{1.0, 0.0, 0.0}
	embedding2 := []float32{0.0, 1.0, 0.0}
	embedding3 := []float32{0.9, 0.1, 0.0}

	memory1 := NewMemory(MemoryTypeSemantic, "content 1", embedding1)
	memory2 := NewMemory(MemoryTypeSemantic, "content 2", embedding2)
	memory3 := NewMemory(MemoryTypeSemantic, "content 3", embedding3)

	store.Store(context.Background(), *memory1)
	store.Store(context.Background(), *memory2)
	store.Store(context.Background(), *memory3)

	results, err := store.Retrieve(context.Background(), MemoryQuery{
		Embedding:     []float32{1.0, 0.0, 0.0},
		MinSimilarity: 0.5,
		Limit:         10,
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify top result has highest similarity (content 1)
	if len(results) > 0 && results[0].Similarity < 0.9 {
		t.Errorf("Expected top result to have similarity >= 0.9, got %f", results[0].Similarity)
	}
}

func TestStore_RetrieveByContent(t *testing.T) {
	store := NewStore(time.Hour)

	memory1 := NewMemory(MemoryTypeSemantic, "python programming language", nil)
	memory2 := NewMemory(MemoryTypeSemantic, "go programming language", nil)

	store.Store(context.Background(), *memory1)
	store.Store(context.Background(), *memory2)

	results, err := store.Retrieve(context.Background(), MemoryQuery{
		Query: "python",
		Limit: 10,
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Memory.Content != "python programming language" {
		t.Errorf("Expected 'python programming language', got '%s'", results[0].Memory.Content)
	}
}

func TestStore_RetrieveBySession(t *testing.T) {
	store := NewStore(time.Hour)

	memory1 := NewMemory(MemoryTypeEpisodic, "event 1", nil)
	memory1.WithSessionID("session-789")
	memory2 := NewMemory(MemoryTypeEpisodic, "event 2", nil)
	memory2.WithSessionID("session-789")
	memory3 := NewMemory(MemoryTypeEpisodic, "event 3", nil)
	memory3.WithSessionID("session-other")

	store.Store(context.Background(), *memory1)
	store.Store(context.Background(), *memory2)
	store.Store(context.Background(), *memory3)

	results, err := store.Retrieve(context.Background(), MemoryQuery{
		SessionID: "session-789",
		Limit:     10,
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewStore(time.Hour)

	memory := NewMemory(MemoryTypeSemantic, "to be deleted", nil)
	store.Store(context.Background(), *memory)

	err := store.Delete(context.Background(), memory.ID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, err = store.GetByID(context.Background(), memory.ID)
	if err != ErrMemoryNotFound {
		t.Errorf("Expected ErrMemoryNotFound, got %v", err)
	}
}

func TestStore_List(t *testing.T) {
	store := NewStore(time.Hour)

	for i := 0; i < 5; i++ {
		memory := NewMemory(MemoryTypeSemantic, "test content", nil)
		store.Store(context.Background(), *memory)
	}

	memories, err := store.List(context.Background(), MemoryTypeSemantic, 3)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(memories) != 3 {
		t.Errorf("Expected 3 memories, got %d", len(memories))
	}
}

func TestStore_Update(t *testing.T) {
	store := NewStore(time.Hour)

	memory := NewMemory(MemoryTypeSemantic, "original content", nil)
	store.Store(context.Background(), *memory)

	memory.Content = "updated content"
	err := store.Update(context.Background(), *memory)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updated, _ := store.GetByID(context.Background(), memory.ID)
	if updated.Content != "updated content" {
		t.Errorf("Expected 'updated content', got '%s'", updated.Content)
	}
}

func TestStore_Clear(t *testing.T) {
	store := NewStore(time.Hour)

	memory1 := NewMemory(MemoryTypeSemantic, "content 1", nil)
	memory2 := NewMemory(MemoryTypeSemantic, "content 2", nil)
	store.Store(context.Background(), *memory1)
	store.Store(context.Background(), *memory2)

	err := store.Clear(context.Background(), MemoryTypeSemantic)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	memories, _ := store.List(context.Background(), MemoryTypeSemantic, 10)
	if len(memories) != 0 {
		t.Errorf("Expected 0 memories after clear, got %d", len(memories))
	}
}

func TestStore_GetStats(t *testing.T) {
	store := NewStore(time.Hour)

	semantic := NewMemory(MemoryTypeSemantic, "semantic", nil)
	episodic := NewMemory(MemoryTypeEpisodic, "episodic", nil)
	procedural := NewMemory(MemoryTypeProcedural, "procedural", nil)

	store.Store(context.Background(), *semantic)
	store.Store(context.Background(), *episodic)
	store.Store(context.Background(), *procedural)

	stats, err := store.GetStats(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if stats.SemanticCount != 1 {
		t.Errorf("Expected 1 semantic, got %d", stats.SemanticCount)
	}
	if stats.EpisodicCount != 1 {
		t.Errorf("Expected 1 episodic, got %d", stats.EpisodicCount)
	}
	if stats.ProceduralCount != 1 {
		t.Errorf("Expected 1 procedural, got %d", stats.ProceduralCount)
	}
	if stats.TotalCount != 3 {
		t.Errorf("Expected 3 total, got %d", stats.TotalCount)
	}
}

func TestCosineSimilarity(t *testing.T) {
	vec1 := []float32{1.0, 0.0, 0.0}
	vec2 := []float32{1.0, 0.0, 0.0}

	similarity := cosineSimilarity(vec1, vec2)
	if similarity < 0.99 || similarity > 1.01 {
		t.Errorf("Expected similarity ~1.0, got %f", similarity)
	}

	vec3 := []float32{0.0, 1.0, 0.0}
	similarity2 := cosineSimilarity(vec1, vec3)
	if similarity2 > 0.1 {
		t.Errorf("Expected similarity ~0.0, got %f", similarity2)
	}

	vec4 := []float32{0.707, 0.707, 0.0}
	vec5 := []float32{0.707, 0.707, 0.0}
	similarity3 := cosineSimilarity(vec4, vec5)
	if similarity3 < 0.99 {
		t.Errorf("Expected similarity ~1.0, got %f", similarity3)
	}
}
