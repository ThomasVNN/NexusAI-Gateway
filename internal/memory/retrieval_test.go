package memory

import (
	"testing"
)

func TestRetrievalEngineRRF(t *testing.T) {
	engine := NewRetrievalEngine(60)

	// Create mock tier
	mock := &mockTier{name: "test"}
	engine.RegisterTier(mock)

	// Test hybrid search
	results, err := engine.HybridSearch("test query", &SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	t.Logf("Hybrid search returned %d results", len(results))
}

func TestRetrievalEngineSingleTier(t *testing.T) {
	engine := NewRetrievalEngine(60)

	mock := &mockTier{name: "single-test"}
	engine.RegisterTier(mock)

	results, err := engine.SearchSingleTier("single-test", "query", &SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Single tier search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result from single tier, got %d", len(results))
	}
}

func TestRetrievalEngineTierNotFound(t *testing.T) {
	engine := NewRetrievalEngine(60)

	results, err := engine.SearchSingleTier("nonexistent", "query", &SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Should not error for nonexistent tier: %v", err)
	}

	if results != nil {
		t.Error("Should return nil for nonexistent tier")
	}
}

func TestRetrievalEngineSourcesFilter(t *testing.T) {
	engine := NewRetrievalEngine(60)

	fts5 := &mockTier{name: "fts5"}
	vec := &mockTier{name: "vec"}
	engine.RegisterTier(fts5)
	engine.RegisterTier(vec)

	// Search with sources filter
	results, err := engine.HybridSearch("query", &SearchOptions{
		Limit:   10,
		Sources: []string{"fts5"},
	})
	if err != nil {
		t.Fatalf("Filtered search failed: %v", err)
	}

	// Only fts5 tier should return results
	for _, r := range results {
		if r.Source != "fts5" {
			t.Errorf("Expected source 'fts5', got '%s'", r.Source)
		}
	}
}

// mockTier implements MemoryStore for testing
type mockTier struct {
	name string
}

func (m *mockTier) Search(ctx interface{}, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	return []*MemoryMatch{
		{
			Memory: &Memory{ID: "mock-1", Content: query},
			Score:  0.9,
			Rank:   0,
			Source: m.name,
		},
	}, nil
}

func (m *mockTier) Add(ctx interface{}, memory *Memory) error {
	return nil
}

func (m *mockTier) Delete(ctx interface{}, id string) error {
	return nil
}

func (m *mockTier) Get(ctx interface{}, id string) (*Memory, error) {
	return &Memory{ID: id}, nil
}

func (m *mockTier) List(ctx interface{}, sessionID string, opts *SearchOptions) ([]*Memory, error) {
	return []*Memory{}, nil
}

func (m *mockTier) Name() string {
	return m.name
}
