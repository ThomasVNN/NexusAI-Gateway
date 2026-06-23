package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// QdrantConfig holds Qdrant tier configuration
type QdrantConfig struct {
	Enabled    bool   `json:"enabled"`
	Endpoint   string `json:"endpoint"` // http://localhost:6333
	APIKey     string `json:"api_key"`
	Collection string `json:"collection"`
	Dimension  int    `json:"dimension"`
}

// QdrantMemory implements vector search using Qdrant HTTP API
type QdrantMemory struct {
	client   *qdrantClient
	config   *QdrantConfig
	embedder *EmbeddingModel
	logger   *slog.Logger
}

// NewQdrantMemory creates a new Qdrant memory store
func NewQdrantMemory(config *QdrantConfig) (*QdrantMemory, error) {
	if config == nil {
		config = &QdrantConfig{}
	}
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:6333"
	}
	if config.Collection == "" {
		config.Collection = "nexusai_memories"
	}
	if config.Dimension == 0 {
		config.Dimension = 384
	}

	m := &QdrantMemory{
		client:   newQdrantClient(config.Endpoint, config.APIKey),
		config:   config,
		logger:   slog.Default(),
	}

	// Initialize embedding model
	embedder, err := NewEmbeddingModel("auto", "", config.APIKey)
	if err != nil {
		m.logger.Warn("embedding model init failed", "error", err)
		embedder = &EmbeddingModel{}
	}
	m.embedder = embedder

	// Ensure collection exists
	if err := m.client.EnsureCollection(config.Collection, config.Dimension); err != nil {
		m.logger.Warn("failed to ensure collection", "error", err)
	}

	return m, nil
}

// Name returns the tier name
func (m *QdrantMemory) Name() string {
	return "qdrant"
}

// Search finds similar memories using vector search
func (m *QdrantMemory) Search(ctx interface{}, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	if query == "" {
		return nil, nil
	}

	if opts == nil {
		opts = DefaultSearchOptions()
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Generate embedding for query
	embedding, err := m.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return m.client.Search(m.config.Collection, embedding, opts.Limit*2, opts)
}

// Add stores a new memory
func (m *QdrantMemory) Add(ctx interface{}, memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now().UTC()
	}
	if memory.UpdatedAt.IsZero() {
		memory.UpdatedAt = time.Now().UTC()
	}

	// Generate embedding if not present
	if len(memory.Embedding) == 0 && memory.Content != "" {
		embedding, err := m.embedder.Embed(memory.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
		memory.Embedding = embedding
	}

	payload := map[string]interface{}{
		"type":        string(memory.Type),
		"content":     memory.Content,
		"session_id":  memory.SessionID,
		"user_id":     memory.UserID,
		"created_at":  memory.CreatedAt.Format(time.RFC3339),
		"updated_at":  memory.UpdatedAt.Format(time.RFC3339),
		"metadata":    memory.Metadata,
	}

	return m.client.UpsertPoint(m.config.Collection, memory.ID, memory.Embedding, payload)
}

// Delete removes a memory by ID
func (m *QdrantMemory) Delete(ctx interface{}, id string) error {
	return m.client.DeletePoint(m.config.Collection, id)
}

// Get retrieves a memory by ID
func (m *QdrantMemory) Get(ctx interface{}, id string) (*Memory, error) {
	result, err := m.client.GetPoint(m.config.Collection, id)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	return resultToMemory(result), nil
}

// List returns all memories for a session
func (m *QdrantMemory) List(ctx interface{}, sessionID string, opts *SearchOptions) ([]*Memory, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	filter := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key":   "session_id",
				"match": map[string]interface{}{"value": sessionID},
			},
		},
	}

	results, _, err := m.client.Scroll(m.config.Collection, opts.Limit, "", filter)
	if err != nil {
		return nil, err
	}

	memories := make([]*Memory, 0, len(results))
	for _, result := range results {
		memories = append(memories, resultToMemory(&result))
	}

	return memories, nil
}

// qdrantClient wraps the Qdrant HTTP API
type qdrantClient struct {
	endpoint  string
	apiKey    string
	transport *http.Client
}

// newQdrantClient creates a new Qdrant HTTP client
func newQdrantClient(endpoint, apiKey string) *qdrantClient {
	if endpoint == "" {
		endpoint = "http://localhost:6333"
	}

	return &qdrantClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		transport: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureCollection creates collection if it doesn't exist
func (c *qdrantClient) EnsureCollection(name string, dimension int) error {
	// Check if collection exists
	url := c.endpoint + "/collections/" + name
	resp, err := c.transport.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Collection exists
	}

	// Create collection
	createURL := c.endpoint + "/collections"
	body := map[string]interface{}{
		"name": name,
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	return c.put(createURL, body)
}

// Search performs vector search
func (c *qdrantClient) Search(collection string, vector []float32, limit int, opts *SearchOptions) ([]*MemoryMatch, error) {
	url := c.endpoint + "/collections/" + collection + "/points/search"

	searchReq := map[string]interface{}{
		"vector":      vector,
		"limit":       limit,
		"with_payload": true,
	}

	if opts != nil && opts.Threshold > 0 {
		searchReq["score_threshold"] = opts.Threshold
	}

	bodyBytes, _ := json.Marshal(searchReq)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)

	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search error: %d - %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result []qdrantPoint `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	matches := make([]*MemoryMatch, 0, len(result.Result))
	for rank, point := range result.Result {
		matches = append(matches, &MemoryMatch{
			Memory: resultToMemory(&point),
			Score:  point.Score,
			Rank:   rank,
			Source: "qdrant",
		})
	}

	return matches, nil
}

// qdrantPoint represents a Qdrant search result
type qdrantPoint struct {
	ID      string                 `json:"id"`
	Score   float64               `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// UpsertPoint inserts or updates a point
func (c *qdrantClient) UpsertPoint(collection string, id string, vector []float32, payload map[string]interface{}) error {
	url := c.endpoint + "/collections/" + collection + "/points"

	upsertReq := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"vector":  vector,
				"payload": payload,
			},
		},
	}

	return c.put(url, upsertReq)
}

// GetPoint retrieves a point by ID
func (c *qdrantClient) GetPoint(collection, id string) (*qdrantPoint, error) {
	url := c.endpoint + "/collections/" + collection + "/points/" + id

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)
	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result qdrantPoint `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &result.Result, nil
}

// DeletePoint deletes a point by ID
func (c *qdrantClient) DeletePoint(collection, id string) error {
	url := c.endpoint + "/collections/" + collection + "/points/" + id

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	c.setHeaders(req)
	resp, err := c.transport.Do(req)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// Scroll retrieves points with pagination
func (c *qdrantClient) Scroll(collection string, limit int, offset string, filter map[string]interface{}) ([]qdrantPoint, string, error) {
	url := c.endpoint + "/collections/" + collection + "/points/scroll"

	body := map[string]interface{}{
		"limit": limit,
	}

	if offset != "" {
		body["offset"] = offset
	}

	if filter != nil {
		body["filter"] = filter
	}

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}

	c.setHeaders(req)
	resp, err := c.transport.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("scroll failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("scroll error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result struct {
			Points          []qdrantPoint `json:"points"`
			NextPageOffset string         `json:"next_page_offset"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("decode failed: %w", err)
	}

	return result.Result.Points, result.Result.NextPageOffset, nil
}

// resultToMemory converts qdrantPoint to Memory
func resultToMemory(result *qdrantPoint) *Memory {
	if result == nil {
		return nil
	}

	mem := &Memory{
		ID:      result.ID,
		Content: fmt.Sprintf("%v", result.Payload["content"]),
	}

	if v, ok := result.Payload["type"]; ok {
		mem.Type = MemoryType(fmt.Sprintf("%v", v))
	}
	if v, ok := result.Payload["session_id"]; ok {
		mem.SessionID = fmt.Sprintf("%v", v)
	}
	if v, ok := result.Payload["user_id"]; ok {
		mem.UserID = fmt.Sprintf("%v", v)
	}
	if created, ok := result.Payload["created_at"].(string); ok {
		mem.CreatedAt, _ = time.Parse(time.RFC3339, created)
	}
	if updated, ok := result.Payload["updated_at"].(string); ok {
		mem.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	}
	if metadata, ok := result.Payload["metadata"].(map[string]interface{}); ok {
		mem.Metadata = metadata
	}

	return mem
}

// setHeaders sets common headers
func (c *qdrantClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
}

// put sends a PUT request
func (c *qdrantClient) put(url string, body map[string]interface{}) error {
	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	c.setHeaders(req)
	resp, err := c.transport.Do(req)
	if err != nil {
		return fmt.Errorf("PUT failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT error: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}
