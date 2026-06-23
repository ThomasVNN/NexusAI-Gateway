package memory

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FTS5Config holds FTS5 tier configuration
type FTS5Config struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"` // SQLite DB path
}

// FTS5Memory implements the MemoryStore interface using FTS5 full-text search
type FTS5Memory struct {
	db     *sql.DB
	config *FTS5Config
	logger *slog.Logger
}

// NewFTS5Memory creates a new FTS5 memory store
func NewFTS5Memory(config *FTS5Config) (*FTS5Memory, error) {
	if config == nil {
		config = &FTS5Config{}
	}
	if config.Path == "" {
		config.Path = "./memory_fts5.db"
	}

	db, err := sql.Open("sqlite3", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite DB: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	m := &FTS5Memory{
		db:     db,
		config: config,
		logger: slog.Default(),
	}

	// Initialize schema
	if err := m.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return m, nil
}

// initSchema creates the FTS5 virtual table and memory table
func (m *FTS5Memory) initSchema() error {
	// Create memories table
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		session_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id);
	CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
	CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at);
	`
	
	if _, err := m.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Create FTS5 virtual table
	ftsSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		content,
		content='memories',
		content_rowid='rowid',
		tokenize='porter unicode61'
	);
	`
	
	if _, err := m.db.Exec(ftsSchema); err != nil {
		return fmt.Errorf("failed to create FTS5 table: %w", err)
	}

	// Create triggers to keep FTS in sync
	triggers := `
	CREATE TRIGGER IF NOT EXISTS memories_fts_insert AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, content) VALUES (new.rowid, new.content);
	END;
	
	CREATE TRIGGER IF NOT EXISTS memories_fts_delete AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content) VALUES('delete', old.rowid, old.content);
	END;
	
	CREATE TRIGGER IF NOT EXISTS memories_fts_update AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content) VALUES('delete', old.rowid, old.content);
		INSERT INTO memories_fts(rowid, content) VALUES (new.rowid, new.content);
	END;
	`

	if _, err := m.db.Exec(triggers); err != nil {
		m.logger.Warn("failed to create FTS triggers (may already exist)", "error", err)
	}

	return nil
}

// Name returns the tier name
func (m *FTS5Memory) Name() string {
	return "fts5"
}

// Search finds memories matching an FTS5 query
func (m *FTS5Memory) Search(ctx interface{}, query string, opts *SearchOptions) ([]*MemoryMatch, error) {
	if query == "" {
		return nil, nil
	}

	if opts == nil {
		opts = DefaultSearchOptions()
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Escape FTS5 special characters and prepare query
	ftsQuery := m.prepareFTSQuery(query)

	sqlQuery := `
		SELECT m.id, m.type, m.content, m.metadata, m.session_id, m.user_id, 
			   m.created_at, m.updated_at, bm25(memories_fts) as score
		FROM memories_fts
		JOIN memories m ON memories_fts.rowid = m.rowid
		WHERE memories_fts MATCH ?
		ORDER BY score
		LIMIT ?
	`

	args := []interface{}{ftsQuery, opts.Limit * 2} // Fetch extra for filtering

	// Add filters
	whereClause := ""
	if opts.MemoryType != "" {
		whereClause += " AND m.type = ?"
		args = append(args, string(opts.MemoryType))
	}
	if opts.SessionID != "" {
		whereClause += " AND m.session_id = ?"
		args = append(args, opts.SessionID)
	}
	if opts.UserID != "" {
		whereClause += " AND m.user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.DateRange != nil {
		whereClause += " AND m.created_at BETWEEN ? AND ?"
		args = append(args, opts.DateRange.Start, opts.DateRange.End)
	}

	if whereClause != "" {
		sqlQuery = strings.Replace(sqlQuery, "WHERE memories_fts MATCH", "WHERE memories_fts MATCH"+whereClause, 1)
	}

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("FTS5 search failed: %w", err)
	}
	defer rows.Close()

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

		// Parse metadata
		if metadata != "" {
			parseMetadata(metadata, mem)
		}

		// Normalize score (BM25 is negative, lower is better)
		normalizedScore := 1.0 / (1.0 + score)
		
		matches = append(matches, &MemoryMatch{
			Memory: mem,
			Score:  normalizedScore,
			Rank:   rank,
			Source: "fts5",
		})
		rank++
	}

	return matches, nil
}

// prepareFTSQuery sanitizes and prepares an FTS5 query
func (m *FTS5Memory) prepareFTSQuery(query string) string {
	// Remove FTS5 special characters that could break the query
	replacer := strings.NewReplacer(
		"\"", "", "'", "", "*", "", "(", "", ")", "",
		":", " ", "^", "", "-", " ", "+", "",
	)

	query = replacer.Replace(query)
	query = strings.TrimSpace(query)

	// Split into words and join with OR for flexible matching
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	// Create phrase query if multiple words
	ftsQuery := "\"" + strings.Join(words, "\" \"") + "\""
	return ftsQuery
}

// Add stores a new memory
func (m *FTS5Memory) Add(ctx interface{}, memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now().UTC()
	}
	if memory.UpdatedAt.IsZero() {
		memory.UpdatedAt = time.Now().UTC()
	}

	metadataJSON := serializeMetadata(memory.Metadata)

	_, err := m.db.Exec(`
		INSERT INTO memories (id, type, content, metadata, session_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		memory.ID,
		string(memory.Type),
		memory.Content,
		metadataJSON,
		memory.SessionID,
		memory.UserID,
		memory.CreatedAt,
		memory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add memory: %w", err)
	}

	return nil
}

// Delete removes a memory by ID
func (m *FTS5Memory) Delete(ctx interface{}, id string) error {
	_, err := m.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// Get retrieves a memory by ID
func (m *FTS5Memory) Get(ctx interface{}, id string) (*Memory, error) {
	row := m.db.QueryRow(`
		SELECT id, type, content, metadata, session_id, user_id, created_at, updated_at
		FROM memories WHERE id = ?
	`, id)

	mem := &Memory{}
	var metadata string

	err := row.Scan(
		&mem.ID, &mem.Type, &mem.Content, &metadata,
		&mem.SessionID, &mem.UserID, &mem.CreatedAt, &mem.UpdatedAt,
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

	return mem, nil
}

// List returns all memories for a session
func (m *FTS5Memory) List(ctx interface{}, sessionID string, opts *SearchOptions) ([]*Memory, error) {
	if opts == nil {
		opts = DefaultSearchOptions()
	}

	query := `
		SELECT id, type, content, metadata, session_id, user_id, created_at, updated_at
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
		var metadata string

		err := rows.Scan(
			&mem.ID, &mem.Type, &mem.Content, &metadata,
			&mem.SessionID, &mem.UserID, &mem.CreatedAt, &mem.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if metadata != "" {
			parseMetadata(metadata, mem)
		}

		memories = append(memories, mem)
	}

	return memories, nil
}

// Close closes the database connection
func (m *FTS5Memory) Close() error {
	return m.db.Close()
}

// Reindex rebuilds the FTS index
func (m *FTS5Memory) Reindex() error {
	_, err := m.db.Exec("INSERT INTO memories_fts(memories_fts) VALUES('rebuild')")
	return err
}

// Indexer handles background indexing for FTS5 tier
type Indexer struct {
	fts5      *FTS5Memory
	queue     chan *Memory
	batchSize int
	interval  time.Duration
	logger    *slog.Logger
	mu        sync.RWMutex
	running   bool
}

// NewIndexer creates a new background indexer
func NewIndexer(fts5 *FTS5Memory, batchSize int, interval time.Duration) *Indexer {
	if batchSize <= 0 {
		batchSize = 100
	}
	if interval <= 0 {
		interval = time.Second
	}

	return &Indexer{
		fts5:      fts5,
		queue:     make(chan *Memory, 1000),
		batchSize: batchSize,
		interval:  interval,
		logger:    slog.Default(),
	}
}

// Start begins background indexing
func (i *Indexer) Start() {
	i.mu.Lock()
	if i.running {
		i.mu.Unlock()
		return
	}
	i.running = true
	i.mu.Unlock()

	go i.run()
	i.logger.Info("indexer started", "batch_size", i.batchSize, "interval", i.interval)
}

// Stop stops background indexing
func (i *Indexer) Stop() {
	i.mu.Lock()
	i.running = false
	i.mu.Unlock()

	close(i.queue)
	i.logger.Info("indexer stopped")
}

// Enqueue adds a memory to the indexing queue
func (i *Indexer) Enqueue(mem *Memory) {
	select {
	case i.queue <- mem:
	default:
		i.logger.Warn("indexer queue full, dropping memory", "id", mem.ID)
	}
}

// EnqueueBatch adds multiple memories to the indexing queue
func (i *Indexer) EnqueueBatch(memories []*Memory) {
	for _, mem := range memories {
		i.Enqueue(mem)
	}
}

// run is the main indexing loop
func (i *Indexer) run() {
	batch := make([]*Memory, 0, i.batchSize)
	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()

	for {
		select {
		case mem, ok := <-i.queue:
			if !ok {
				// Process remaining batch
				if len(batch) > 0 {
					i.processBatch(batch)
				}
				return
			}

			batch = append(batch, mem)
			if len(batch) >= i.batchSize {
				i.processBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				i.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// processBatch indexes a batch of memories
func (i *Indexer) processBatch(memories []*Memory) {
	var indexed, failed int

	for _, mem := range memories {
		if err := i.fts5.Add(nil, mem); err != nil {
			failed++
			i.logger.Warn("failed to index memory", "id", mem.ID, "error", err)
		} else {
			indexed++
		}
	}

	if indexed > 0 || failed > 0 {
		i.logger.Debug("batch indexed", "indexed", indexed, "failed", failed)
	}
}

// QueueSize returns the current queue size
func (i *Indexer) QueueSize() int {
	return len(i.queue)
}

// IsRunning returns whether the indexer is running
func (i *Indexer) IsRunning() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.running
}
