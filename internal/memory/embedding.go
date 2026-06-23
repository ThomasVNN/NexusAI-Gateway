package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// EmbeddingSource specifies where embeddings come from
type EmbeddingSource string

const (
	EmbeddingRemote       EmbeddingSource = "remote"        // OpenAI/Cohere API
	EmbeddingStatic       EmbeddingSource = "static"        // Static lookup
	EmbeddingONNXLocal   EmbeddingSource = "onnx"          // Local ONNX model
	EmbeddingAuto         EmbeddingSource = "auto"          // Runtime resolution
)

// EmbeddingModel handles text embedding generation
type EmbeddingModel struct {
	source    EmbeddingSource
	apiKey    string
	model     string
	dimension int

	// For remote API
	httpClient *http.Client

	logger *slog.Logger
}

// NewEmbeddingModel creates a new embedding model
func NewEmbeddingModel(source, model, apiKey string) (*EmbeddingModel, error) {
	e := &EmbeddingModel{
		source:    EmbeddingSource(source),
		apiKey:    apiKey,
		model:     model,
		dimension: 384, // Default for all-MiniLM-L6-v2
		logger:    slog.Default(),
	}

	// Set default model if not specified
	if e.model == "" {
		e.model = "sentence-transformers/all-MiniLM-L6-v2"
	}

	// Determine source
	if e.source == EmbeddingAuto {
		e.source = e.detectSource()
	}

	switch e.source {
	case EmbeddingONNXLocal:
		// ONNX would require separate ONNX runtime library
		// For now, fall back to static
		e.logger.Warn("ONNX source selected but runtime not available, using static")
		e.source = EmbeddingStatic
	case EmbeddingRemote:
		e.httpClient = &http.Client{Timeout: 30 * time.Second}
		e.dimension = 1536 // Default for OpenAI
	}

	return e, nil
}

// detectSource determines the best embedding source
func (e *EmbeddingModel) detectSource() EmbeddingSource {
	// Check if ONNX model file is available
	if _, err := os.Stat(e.model); err == nil {
		return EmbeddingONNXLocal
	}

	// Check for environment variables for API
	if e.apiKey != "" || os.Getenv("OPENAI_API_KEY") != "" {
		return EmbeddingRemote
	}

	// Default to static (deterministic, no external dependencies)
	return EmbeddingStatic
}

// Embed generates an embedding for the given text
func (e *EmbeddingModel) Embed(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	switch e.source {
	case EmbeddingONNXLocal:
		return e.embedONNX(text)
	case EmbeddingRemote:
		return e.embedRemote(text)
	case EmbeddingStatic:
		return e.embedStatic(text)
	default:
		return e.embedStatic(text)
	}
}

// embedONNX generates embedding using ONNX model (placeholder)
func (e *EmbeddingModel) embedONNX(text string) ([]float32, error) {
	// In production, this would use onnxruntime-go or similar
	// For now, return static embedding with text hash
	return e.embedStatic(text)
}

// embedRemote generates embedding using remote API
func (e *EmbeddingModel) embedRemote(text string) ([]float32, error) {
	apiKey := e.apiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("no API key available")
	}

	// Prepare request
	url := "https://api.openai.com/v1/embeddings"
	reqBody := map[string]interface{}{
		"input": text,
		"model": e.model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	e.dimension = len(result.Data[0].Embedding)
	return result.Data[0].Embedding, nil
}

// embedStatic generates embedding using static lookup (for common phrases)
func (e *EmbeddingModel) embedStatic(text string) ([]float32, error) {
	// Simple hash-based embedding for static phrases
	// In production, this would use a lookup table
	text = strings.ToLower(strings.TrimSpace(text))

	embedding := make([]float32, e.dimension)

	// Generate deterministic "embedding" based on text hash
	hash := hashString(text)
	r := int64(hash)

	for i := 0; i < e.dimension; i++ {
		// Generate pseudo-random but deterministic values
		r = (r*1103515245 + 12345) & 0x7fffffff
		embedding[i] = float32(r%1000) / 1000.0
	}

	// Normalize
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(1.0) / float32(norm+0.0001)

	for i := range embedding {
		embedding[i] *= norm
	}

	return embedding, nil
}

// Dimension returns the embedding dimension
func (e *EmbeddingModel) Dimension() int {
	return e.dimension
}

// Source returns the embedding source
func (e *EmbeddingModel) Source() EmbeddingSource {
	return e.source
}

// Close releases resources
func (e *EmbeddingModel) Close() error {
	return nil
}

// BatchEmbed generates embeddings for multiple texts
func (e *EmbeddingModel) BatchEmbed(texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	var err error
	for i, text := range texts {
		embeddings[i], err = e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("batch embed failed at %d: %w", i, err)
		}
	}

	return embeddings, nil
}

// hashString computes a simple hash of a string
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}
