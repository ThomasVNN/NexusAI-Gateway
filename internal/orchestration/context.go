package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ContextManager handles conversation context for multi-turn interactions
type ContextManager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	maxHistory  int
	ttl         time.Duration
	cleanupTick time.Duration
}

// Session represents a user conversation session
type Session struct {
	ID          string
	TenantID    string
	UserID      string
	Messages    []*Message
	Metadata    map[string]interface{}
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	IsActive    bool
}

// Message represents a single turn in a conversation
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	Tokens    int       `json:"tokens,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewContextManager creates a new context manager
func NewContextManager() *ContextManager {
	return &ContextManager{
		sessions:    make(map[string]*Session),
		maxHistory:  50,
		ttl:         24 * time.Hour,
		cleanupTick: 1 * time.Hour,
	}
}

// CreateSession creates a new conversation session
func (cm *ContextManager) CreateSession(sessionID, tenantID, userID string) *Session {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session := &Session{
		ID:        sessionID,
		TenantID:  tenantID,
		UserID:    userID,
		Messages:  make([]*Message, 0),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(cm.ttl),
		IsActive:  true,
	}

	cm.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID
func (cm *ContextManager) GetSession(sessionID string) (*Session, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, ok := cm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// AddMessage adds a message to a session
func (cm *ContextManager) AddMessage(sessionID string, msg *Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, ok := cm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	msg.ID = fmt.Sprintf("msg-%d", len(session.Messages)+1)
	msg.Timestamp = time.Now()

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(cm.ttl)

	// Trim history if needed
	if len(session.Messages) > cm.maxHistory {
		session.Messages = session.Messages[len(session.Messages)-cm.maxHistory:]
	}

	return nil
}

// GetMessages returns all messages in a session
func (cm *ContextManager) GetMessages(sessionID string) ([]*Message, error) {
	session, ok := cm.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session.Messages, nil
}

// GetContextForLLM formats messages for LLM consumption
func (cm *ContextManager) GetContextForLLM(sessionID string, maxTokens int) ([]*Message, int, error) {
	session, ok := cm.GetSession(sessionID)
	if !ok {
		return nil, 0, fmt.Errorf("session not found: %s", sessionID)
	}

	messages := session.Messages
	totalTokens := 0

	// Estimate tokens (rough: 4 chars = 1 token)
	for _, msg := range messages {
		msgTokens := len(msg.Content) / 4
		if totalTokens+msgTokens > maxTokens {
			// Truncate from oldest
			messages = messages[len(messages):]
			break
		}
		totalTokens += msgTokens
	}

	return messages, totalTokens, nil
}

// ClearSession removes all messages from a session
func (cm *ContextManager) ClearSession(sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, ok := cm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Messages = make([]*Message, 0)
	session.UpdatedAt = time.Now()

	return nil
}

// DeleteSession removes a session entirely
func (cm *ContextManager) DeleteSession(sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.sessions[sessionID]; !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(cm.sessions, sessionID)
	return nil
}

// SetSessionMetadata sets metadata on a session
func (cm *ContextManager) SetSessionMetadata(sessionID string, key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, ok := cm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Metadata[key] = value
	session.UpdatedAt = time.Now()

	return nil
}

// GetSessionMetadata retrieves metadata from a session
func (cm *ContextManager) GetSessionMetadata(sessionID string, key string) (interface{}, bool) {
	session, ok := cm.GetSession(sessionID)
	if !ok {
		return nil, false
	}

	val, ok := session.Metadata[key]
	return val, ok
}

// Cleanup removes expired sessions
func (cm *ContextManager) Cleanup() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, session := range cm.sessions {
		if now.After(session.ExpiresAt) {
			delete(cm.sessions, id)
			removed++
		}
	}

	return removed
}

// SessionCount returns the number of active sessions
func (cm *ContextManager) SessionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.sessions)
}

// SerializeContext creates a JSON representation of session context
func (cm *ContextManager) SerializeContext(sessionID string) (string, error) {
	session, ok := cm.GetSession(sessionID)
	if !ok {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	data := map[string]interface{}{
		"session_id": session.ID,
		"tenant_id":  session.TenantID,
		"user_id":    session.UserID,
		"messages":   session.Messages,
		"metadata":   session.Metadata,
		"created_at": session.CreatedAt,
		"updated_at": session.UpdatedAt,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize context: %w", err)
	}

	return string(bytes), nil
}

// DeserializeContext restores session context from JSON
func (cm *ContextManager) DeserializeContext(sessionID string, data string) error {
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(data), &ctx); err != nil {
		return fmt.Errorf("failed to deserialize context: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	session := &Session{
		ID:        sessionID,
		Messages:  make([]*Message, 0),
		Metadata:  make(map[string]interface{}),
		IsActive:  true,
	}

	if tenantID, ok := ctx["tenant_id"].(string); ok {
		session.TenantID = tenantID
	}
	if userID, ok := ctx["user_id"].(string); ok {
		session.UserID = userID
	}
	if metadata, ok := ctx["metadata"].(map[string]interface{}); ok {
		session.Metadata = metadata
	}

	cm.sessions[sessionID] = session
	return nil
}

// ChainContext carries orchestration data across the request pipeline
type ChainContext struct {
	mu           sync.RWMutex
	TenantID     string
	UserID       string
	SessionID    string
	ChainID      string
	StepID       string
	InputContext map[string]interface{}
	OutputData   map[string]interface{}
	Metadata     map[string]string
}

// NewChainContext creates a new chain context
func NewChainContext(tenantID, userID string) *ChainContext {
	return &ChainContext{
		TenantID:     tenantID,
		UserID:       userID,
		InputContext: make(map[string]interface{}),
		OutputData:   make(map[string]interface{}),
		Metadata:     make(map[string]string),
	}
}

// SetInput stores input data for the chain
func (cc *ChainContext) SetInput(key string, value interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.InputContext[key] = value
}

// GetInput retrieves input data from the chain
func (cc *ChainContext) GetInput(key string) (interface{}, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	v, ok := cc.InputContext[key]
	return v, ok
}

// SetOutput stores output data from a step
func (cc *ChainContext) SetOutput(key string, value interface{}) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.OutputData[key] = value
}

// GetOutput retrieves output data from the chain
func (cc *ChainContext) GetOutput(key string) (interface{}, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	v, ok := cc.OutputData[key]
	return v, ok
}

// SetMetadata stores string metadata
func (cc *ChainContext) SetMetadata(key, value string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.Metadata[key] = value
}

// GetMetadata retrieves string metadata
func (cc *ChainContext) GetMetadata(key string) (string, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	v, ok := cc.Metadata[key]
	return v, ok
}

// WithChainContext returns a context with chain data
func WithChainContext(ctx context.Context, cc *ChainContext) context.Context {
	return context.WithValue(ctx, chainContextKey{}, cc)
}

// GetChainContext retrieves chain context from context
func GetChainContext(ctx context.Context) (*ChainContext, bool) {
	cc, ok := ctx.Value(chainContextKey{}).(*ChainContext)
	return cc, ok
}

type chainContextKey struct{}
