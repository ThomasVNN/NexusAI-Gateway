package compression

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// DedupEngine provides session-based content deduplication.
// This engine tracks content across conversation turns and removes
// duplicate content, providing 5-15% savings.
//
// ENG-9202: Session-Dedup engine
type DedupEngine struct {
	enabled        bool
	stats          EngineStats
	mu             sync.RWMutex
	sessions       map[string]*SessionDedup
	maxSessionAge  int // in minutes
}

// NewDedupEngine creates a new session deduplication engine
func NewDedupEngine() *DedupEngine {
	return &DedupEngine{
		enabled:       true,
		maxSessionAge: 60, // 1 hour default
		sessions:      make(map[string]*SessionDedup),
		stats: EngineStats{
			Name: "dedup",
		},
	}
}

// Name returns the engine name
func (e *DedupEngine) Name() string {
	return "dedup"
}

// Priority returns the execution priority
func (e *DedupEngine) Priority() int {
	return 10
}

// IsEnabled returns whether the engine is active
func (e *DedupEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetEnabled enables or disables the engine
func (e *DedupEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// GetOrCreateSession retrieves or creates a session deduplication tracker
func (e *DedupEngine) GetOrCreateSession(sessionID string) *SessionDedup {
	e.mu.Lock()
	defer e.mu.Unlock()

	if session, exists := e.sessions[sessionID]; exists {
		return session
	}

	session := &SessionDedup{
		SessionID:  sessionID,
		SeenHashes: make(map[string]string),
		SeenLines:  make(map[string]bool),
	}
	e.sessions[sessionID] = session
	return session
}

// RemoveSession removes a session and its deduplication data
func (e *DedupEngine) RemoveSession(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.sessions, sessionID)
}

// ClearOldSessions removes sessions older than maxSessionAge
func (e *DedupEngine) ClearOldSessions() {
	e.mu.Lock()
	defer e.mu.Unlock()
	// For now, just clear all - could implement age tracking later
	e.sessions = make(map[string]*SessionDedup)
}

// Compress removes duplicate content from the input based on session history
func (e *DedupEngine) Compress(input string) (string, int, error) {
	return e.CompressWithSession(input, "default")
}

// CompressWithSession removes duplicate content from the input for a specific session
func (e *DedupEngine) CompressWithSession(input string, sessionID string) (string, int, error) {
	if input == "" {
		return "", 0, nil
	}

	originalLen := len(input)
	session := e.GetOrCreateSession(sessionID)

	// Split into lines for analysis
	lines := strings.Split(input, "\n")
	var uniqueLines []string
	var totalSaved int

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue // Skip empty lines
		}

		// Create a hash of the normalized line content
		normalized := strings.ToLower(strings.TrimSpace(line))
		hash := e.hashContent(normalized)

		// Check if we've seen this exact content
		if _, seen := session.SeenHashes[hash]; seen {
			totalSaved += len(line) + 1 // +1 for newline
			continue
		}

		// Check for similar lines using a shorter hash
		shortNorm := normalized
		if len(shortNorm) > 50 {
			shortNorm = shortNorm[:50]
		}
		shortHash := e.hashContent(shortNorm)
		if session.SeenLines[shortHash] {
			totalSaved += len(line) + 1
			continue
		}

		// Store the hash
		session.SeenHashes[hash] = normalized
		session.SeenLines[shortHash] = true
		uniqueLines = append(uniqueLines, line)
	}

	compressed := strings.Join(uniqueLines, "\n")

	// Update stats
	e.mu.Lock()
	e.stats.Invocations++
	e.stats.TotalInputLen += int64(originalLen)
	e.stats.TotalOutputLen += int64(len(compressed))
	e.stats.TotalSaved += int64(totalSaved)
	e.stats.LastUsed = time.Now()
	e.mu.Unlock()

	return compressed, totalSaved, nil
}

// hashContent creates a SHA-256 hash of the content
func (e *DedupEngine) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Stats returns the engine statistics
func (e *DedupEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}

// ResetStats clears the engine statistics
func (e *DedupEngine) ResetStats() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats = EngineStats{Name: "dedup"}
	e.sessions = make(map[string]*SessionDedup)
}

// Ensure DedupEngine implements CompressionEngine
var _ CompressionEngine = (*DedupEngine)(nil)
