# Session Memory Platform

**Version:** 1.0
**Date:** June 23, 2026
**Owner:** Backend Agent
**EPIC:** EPIC-OMNI-04

A 3-tier memory system for persistent context across AI sessions without token waste.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Memory System                         │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │   Tier 0    │  │   Tier 1    │  │   Tier 2    │   │
│  │    FTS5     │  │  sqlite-vec │  │   Qdrant    │   │
│  │  (keyword)  │  │   (local)   │  │  (remote)   │   │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘   │
│         │                  │                  │          │
│         └──────────────────┼──────────────────┘          │
│                            ▼                             │
│              ┌─────────────────────────┐                │
│              │   Hybrid Retrieval      │                │
│              │   (RRF k=60)            │                │
│              └─────────────────────────┘                │
├─────────────────────────────────────────────────────────┤
│                  Extraction Engine                       │
│              (Non-blocking regex extraction)             │
└─────────────────────────────────────────────────────────┘
```

## Memory Types

| Type | Description | Use Case |
|------|-------------|----------|
| `factual` | Facts, statements | "X is Y", "The capital of..." |
| `episodic` | Events, experiences | "Yesterday I did...", "We discussed..." |
| `procedural` | How-to, steps | "To do X, first Y, then Z" |
| `semantic` | Concepts, meanings | "The concept of...", "Essentially means..." |

## Tiers

### Tier 0: FTS5 (Full-Text Search)
- **Purpose:** Fast keyword matching
- **Storage:** SQLite with FTS5 virtual table
- **Features:** Porter stemming, Unicode61 tokenization
- **Best for:** Exact phrase matches, boolean queries

### Tier 1: sqlite-vec (Local Vectors)
- **Purpose:** Semantic similarity without external API
- **Storage:** SQLite with vec0 extension
- **Embedding:** ONNX model (all-MiniLM-L6-v2) or static lookup
- **Best for:** Offline operation, privacy-sensitive data

### Tier 2: Qdrant (Remote Vectors)
- **Purpose:** Cloud-scale similarity search
- **Storage:** Qdrant vector database
- **Features:** HNSW indexing, cosine similarity
- **Best for:** High-volume, distributed deployments

## Usage

### Basic Operations

```go
import "github.com/ThomasVNN/NexusAI-Gateway/internal/memory"

// Configure memory system
config := &memory.MemoryConfig{
    DefaultLimit: 10,
    RRF_K:        60,
    Tier0: memory.FTS5Config{Enabled: true, Path: "./memory.db"},
    Tier1: memory.VecConfig{Enabled: true, Path: "./memory_vec.db"},
    Tier2: memory.QdrantConfig{
        Enabled:    true,
        Endpoint:   "http://localhost:6333",
        Collection: "nexusai_memories",
    },
}

// Create memory system
ms, err := memory.NewMemorySystem(config)
if err != nil {
    log.Fatal(err)
}
defer ms.Close()

// Add a memory
mem := &memory.Memory{
    Type:      memory.MemoryFactual,
    Content:   "The capital of France is Paris",
    SessionID: "session-123",
    UserID:    "user-456",
}
if err := ms.Add(ctx, mem); err != nil {
    log.Fatal(err)
}

// Search memories
results, err := ms.Search(ctx, "What is the capital of France?", nil)
if err != nil {
    log.Fatal(err)
}

for _, match := range results {
    fmt.Printf("Score: %.2f, Content: %s\n", match.Score, match.Memory.Content)
}
```

### Extraction

```go
// Extract and store memories from content
extracted, err := ms.ExtractAndStore(ctx, 
    "Remember that to make coffee, first boil water, then add grounds. The best coffee comes from Colombia.",
    "session-123",
    "user-456",
)
```

### Configuration

```go
// Search options
opts := &memory.SearchOptions{
    Limit:      10,
    MemoryType: memory.MemoryFactual,  // Filter by type
    SessionID:  "session-123",          // Filter by session
    Hybrid:     true,                   // Use all tiers
    RRF_K:      60,                     // RRF constant
}

// Or search a single tier
results, err := ms.SearchSingleTier("fts5", ctx, "query", opts)
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/mattn/go-sqlite3` | SQLite driver |
| `github.com/qdrant/go-client/v2` | Qdrant client |

## Testing

```bash
go test ./internal/memory/... -v
```

## Performance

- FTS5: ~1ms for 10K documents
- sqlite-vec: ~5ms for 10K vectors
- Qdrant: ~10ms for 100K vectors
- Hybrid search: ~15ms end-to-end

---

_Implementation: EPIC-OMNI-04 (Session Memory Platform)_
_Story Points: 47 | Effort: 8 weeks_
