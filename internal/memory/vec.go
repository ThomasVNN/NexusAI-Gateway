package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// VecConfig holds sqlite-vec tier configuration
type VecConfig struct {
	Enabled  bool   `json:"enabled"`
	Path     string `json:"path"` // SQLite DB path
	Model    string `json:"model"` // ONNX model path
	Provider string `json:"provider"` // "remote", "static", "transformers", "auto"
	APIKey   string `json:"api_key"` // For remote embedding
	Dimension int   `json:"dimension"` // Vector dimension
}

// VecMemory implements vector similarity search using sqlite-vec
type VecMemory struct {
	db       *sql.DB
	config   *VecConfig
	embedder *EmbeddingModel
	logger   *slog.Logger
}

// NewVecMemory creates a new VecMemory store
func NewVecMemory(config *VecConfig) (*VecMemory, error) {
	if config == nil {
		config = &VecConfig{}
	}
	if config.Path == "" {
		config.Path = "./memory_vec.db"
	}
	if config.Dimension == 0 {
		config.Dimension = 384 // Default for all-MiniLM-L6-v2
	}

	db, err := sql.Open("sqlite3", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite DB: %w", err)
	}

	// Configure connection
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	m := &VecMemory{
		db:     db,
		config: config,
		logger: slog.Default(),
	}

	// Initialize embedding model
	embedder, err := NewEmbeddingModel(config.Provider, config.Model, config.APIKey)
	if err != nil {
		m.logger.Warn("embedding model init failed, using fallback", "error", err)
		embedder = &EmbeddingModel{}
	}
	m.embedder = embedder

	// Initialize schema
	if err := m.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return m, nil
}

// initSchema creates the sqlite-vec tables
func (m *VecMemory) initSchema() error {
	// Load sqlite-vec extension (if available)
	if _, err := m.db.Exec("SELECT load_extension('vec0')"); err != nil {
		// sqlite-vec may not be available, use alternative storage
		m.logger.Warn("vec0 extension not available, using alternative storage")
	}

	// Create memories table
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		embedding BLOB,
		session_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id);
	CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
	`

	if _, err := m.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Create vector table (if vec extension is available)
	vecSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS memories_vec USING vec0(
		embedding[float]{0},
		columns=1,
		metric=cosine
	);
	`

	if _, err := m.db.Exec(vecSchema); err != nil {
		m.logger.Debug("vec virtual table may already exist or extension not available")
	}

	return nil
}

// Name returns the tier name
func (m *VecMemory) Name() string {
	return "vec"
}

// Search finds similar memories using vector search
func (m *VecMemory) Search(ctx interface{}, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
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
		m.logger.Warn("embedding generation failed, falling back to content match", "error", err)
		return m.searchByContent(query, opts)
	}

	// Try vector search first
	matches, err := m.searchByVector(embedding, opts)
	if err != nil || len(matches) == 0 {
		// Fallback to content search
		return m.searchByContent(query, opts)
	}

	return matches, nil
}

// searchByVector searches using vector similarity
func (m *VecMemory) searchByVector(queryEmbedding []float32, opts *SearchOptions) ([]*MemoryMatch, error) {
	// Try sqlite-vec search
	sqlQuery := `
		SELECT m.id, m.type, m.content, m.metadata, m.session_id, m.user_id,
			   m.created_at, m.updated_at,
			   (1 - v.distance) as score
		FROM memories m
		JOIN memories_vec v ON m.id = v.id
		WHERE v.top_k = ?
		ORDER BY v.distance
	`

	rows, err := m.db.Query(sqlQuery, opts.Limit*2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanVectorResults(rows, opts)
}

// searchByContent searches by text content (fallback when vec not available)
func (m *VecMemory) searchByContent(query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	// Simple LIKE-based search with scoring
	sqlQuery := `
		SELECT id, type, content, metadata, session_id, user_id,
			   created_at, updated_at,
			   CAST(LENGTH(content) - LENGTH(REPLACE(LOWER(content), LOWER(?), '')) AS FLOAT) / 
			   NULLIF(LENGTH(?), 0) as score
		FROM memories
		WHERE LOWER(content) LIKE LOWER('%' || ? || '%')
		ORDER BY score DESC
		LIMIT ?
	`

	rows, err := m.db.Query(sqlQuery, query, query, query, opts.Limit*2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanVectorResults(rows, opts)
}

// scanVectorResults scans query results into MemoryMatch slices
func (m *VecMemory) scanVectorResults(rows *sql.Rows, opts *SearchOptions) ([]*MemoryMatch, error) {
	var matches []*MemoryMatch
	rank := 0

	for rows.Next() {
		mem := &Memory{}
		var metadata string
		var score float64

		err := rows.Scan(
			&mem.ID, &mem.Type, &mem.Content, &metadata,
			&mem.SessionID, &mem.UserID, &mem.CreatedAt, &mem.UpdatedAt, &score,
		)
		if err != nil {
			continue
		}

		if metadata != "" {
			parseMetadata(metadata, mem)
		}

		// Normalize score
		if score < 0 {
			score = 0
		}

		matches = append(matches, &MemoryMatch{
			Memory: mem,
			Score:  score,
			Rank:   rank,
			Source: "vec",
		})
		rank++
	}

	// Apply filters
	if opts.MemoryType != "" || opts.SessionID != "" || opts.UserID != "" || opts.DateRange != nil {
		matches = m.filterMatches(matches, opts)
	}

	return matches, nil
}

// filterMatches applies additional filters to matches
func (m *VecMemory) filterMatches(matches []*MemoryMatch, opts *SearchOptions) []*MemoryMatch {
	filtered := make([]*MemoryMatch, 0)

	for _, match := range matches {
		mem := match.Memory

		if opts.MemoryType != "" && mem.Type != opts.MemoryType {
			continue
		}
		if opts.SessionID != "" && mem.SessionID != opts.SessionID {
			continue
		}
		if opts.UserID != "" && mem.UserID != opts.UserID {
			continue
		}
		if opts.DateRange != nil {
			if mem.CreatedAt.Before(opts.DateRange.Start) || mem.CreatedAt.After(opts.DateRange.End) {
				continue
			}
		}

		filtered = append(filtered, match)
	}

	return filtered
}

// Add stores a new memory with embedding
func (m *VecMemory) Add(ctx interface{}, memory *Memory) error {
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
			m.logger.Warn("failed to generate embedding", "error", err)
		} else {
			memory.Embedding = embedding
		}
	}

	metadataJSON := serializeMetadata(memory.Metadata)

	// Store embedding as JSON blob
	var embeddingJSON []byte
	if len(memory.Embedding) > 0 {
		embeddingJSON, _ = json.Marshal(memory.Embedding)
	}

	_, err := m.db.Exec(`
		INSERT INTO memories (id, type, content, metadata, embedding, session_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		memory.ID,
		string(memory.Type),
		memory.Content,
		metadataJSON,
		embeddingJSON,
		memory.SessionID,
		memory.UserID,
		memory.CreatedAt,
		memory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add memory: %w", err)
	}

	// Try to add to vector table
	if len(memory.Embedding) > 0 {
		m.addToVector(memory)
	}

	return nil
}

// addToVector adds memory to vector table
func (m *VecMemory) addToVector(memory *Memory) {
	embeddingJSON, _ := json.Marshal(memory.Embedding)
	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO memories_vec (id, embedding)
		VALUES (?, ?)
	`, memory.ID, embeddingJSON)

	if err != nil {
		m.logger.Debug("failed to add to vector table", "error", err)
	}
}

// Delete removes a memory by ID
func (m *VecMemory) Delete(ctx interface{}, id string) error {
	// Delete from vector table first
	m.db.Exec("DELETE FROM memories_vec WHERE id = ?", id)

	_, err := m.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// Get retrieves a memory by ID
func (m *VecMemory) Get(ctx interface{}, id string) (*Memory, error) {
	row := m.db.QueryRow(`
		SELECT id, type, content, metadata, embedding, session_id, user_id, created_at, updated_at
		FROM memories WHERE id = ?
	`, id)

	mem := &Memory{}
	var metadata, embeddingJSON string

	err := row.Scan(
		&mem.ID, &mem.Type, &mem.Content, &metadata,
		&embeddingJSON, &mem.SessionID, &mem.UserID, &mem.CreatedAt, &mem.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if metadata != "" {
		parseMetadata(metadata, mem)
	}

	if embeddingJSON != "" {
		json.Unmarshal([]byte(embeddingJSON), &mem.Embedding)
	}

	return mem, nil
}

// List returns all memories for a session
func (m *VecMemory) List(ctx interface{}, sessionID string, opts *SearchOptions) ([]*Memory, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	query := `
		SELECT id, type, content, metadata, embedding, session_id, user_id, created_at, updated_at
		FROM memories
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	args := []interface{}{sessionID, opts.Limit}

	if opts.MemoryType != "" {
		query += " AND type = ?"
		args = append(args, string(opts.MemoryType))
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		mem := &Memory{}
		var metadata, embeddingJSON string

		err := rows.Scan(
			&mem.ID, &mem.Type, &mem.Content, &metadata,
			&embeddingJSON, &mem.SessionID, &mem.UserID, &mem.CreatedAt, &mem.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if metadata != "" {
			parseMetadata(metadata, mem)
		}
		if embeddingJSON != "" {
			json.Unmarshal([]byte(embeddingJSON), &mem.Embedding)
		}

		memories = append(memories, mem)
	}

	return memories, nil
}

// Close closes the database connection
func (m *VecMemory) Close() error {
	return m.db.Close()
}

// UpdateEmbedding updates the embedding for an existing memory
func (m *VecMemory) UpdateEmbedding(id string, content string) error {
	embedding, err := m.embedder.Embed(content)
	if err != nil {
		return err
	}

	embeddingJSON, _ := json.Marshal(embedding)

	// Update memories table
	_, err = m.db.Exec("UPDATE memories SET embedding = ?, updated_at = ? WHERE id = ?",
		embeddingJSON, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	// Update vector table
	m.addToVector(&Memory{ID: id, Embedding: embedding})
	return nil
}

// GetEmbedder returns the embedding model
func (m *VecMemory) GetEmbedder() *EmbeddingModel {
	return m.embedder
}
