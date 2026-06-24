package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageType defines A2A message types
type MessageType string

const (
	// Core message types
	MessageTypeTaskSend      MessageType = "message/send"
	MessageTypeTaskResend     MessageType = "message/resend"
	MessageTypeTaskCancel     MessageType = "message/cancel"
	MessageTypeTaskAccept     MessageType = "task/accept"
	MessageTypeTaskSubmit     MessageType = "task/submit"
	MessageTypeTaskReject     MessageType = "task/reject"
	MessageTypeTaskGet        MessageType = "tasks/get"
	MessageTypeTasksList      MessageType = "tasks/list"
	MessageTypeTasksCancel    MessageType = "tasks/cancel"
	MessageTypeAgentDiscovery MessageType = "agent/discovery"
	MessageTypeAgentRegister  MessageType = "agent/register"
	MessageTypeAgentCardGet   MessageType = "agent/card"
)

// StreamEventType defines streaming event types
type StreamEventType string

const (
	StreamEventTaskStarted   StreamEventType = "task_started"
	StreamEventTaskProgress  StreamEventType = "task_progress"
	StreamEventTaskCompleted StreamEventType = "task_completed"
	StreamEventTaskFailed    StreamEventType = "task_failed"
	StreamEventPing         StreamEventType = "ping"
)

// A2AMessage represents an A2A protocol message
type A2AMessage struct {
	ID        string                 `json:"id"`
	Type      MessageType            `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	SessionID string                 `json:"session_id,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// AgentCard represents agent discovery information
type AgentCard struct {
	AgentID       string                 `json:"agent_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Version       string                 `json:"version"`
	Capabilities  []string               `json:"capabilities"`
	Skills       []string               `json:"skills"`
	Endpoints    map[string]string      `json:"endpoints"` // "http", "stream", "sse"
	Auth         AgentAuth              `json:"auth"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// AgentAuth defines agent authentication requirements
type AgentAuth struct {
	Type        string   `json:"type"` // "none", "api_key", "bearer", "oauth2"
	Scopes      []string `json:"scopes,omitempty"`
	Required    bool     `json:"required"`
}

// ExtendedA2AServer extends the base A2A server with full protocol support
type ExtendedA2AServer struct {
	*Server
	mu           sync.RWMutex
	agents       map[string]*AgentCard
	sessions     map[string]*Session
	messageQueue chan *A2AMessage
	eventHandler *EventHandler
}

// Session represents an A2A session
type Session struct {
	ID         string           `json:"id"`
	AgentID    string           `json:"agent_id"`
	CreatedAt  time.Time        `json:"created_at"`
	LastActive time.Time        `json:"last_active"`
	State      map[string]interface{} `json:"state"`
}

// EventHandler handles streaming events
type EventHandler struct {
	mu       sync.RWMutex
	sessions map[string]chan *StreamEvent
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type      StreamEventType          `json:"type"`
	TaskID    string                  `json:"task_id,omitempty"`
	SessionID string                  `json:"session_id,omitempty"`
	Data      map[string]interface{}  `json:"data,omitempty"`
	Timestamp time.Time               `json:"timestamp"`
}

// NewExtendedServer creates a new extended A2A server
func NewExtendedServer() *ExtendedA2AServer {
	s := &ExtendedA2AServer{
		Server:       NewServer(),
		agents:      make(map[string]*AgentCard),
		sessions:    make(map[string]*Session),
		messageQueue: make(chan *A2AMessage, 1000),
		eventHandler: &EventHandler{
			sessions: make(map[string]chan *StreamEvent),
		},
	}

	s.registerExtendedHandlers()
	go s.processMessageQueue()

	return s
}

// registerExtendedHandlers registers extended protocol handlers
func (s *ExtendedA2AServer) registerExtendedHandlers() {
	s.handlers["agent/discovery"] = s.handleAgentDiscovery
	s.handlers["agent/register"] = s.handleAgentRegister
	s.handlers["agent/card"] = s.handleAgentCard
}

// processMessageQueue processes incoming messages
func (s *ExtendedA2AServer) processMessageQueue() {
	for msg := range s.messageQueue {
		s.processMessage(msg)
	}
}

// processMessage processes a single message
func (s *ExtendedA2AServer) processMessage(msg *A2AMessage) {
	slog.DebugContext(context.Background(), "Processing A2A message",
		slog.String("id", msg.ID),
		slog.String("type", string(msg.Type)),
		slog.String("from", msg.From),
		slog.String("to", msg.To),
	)
}

// RegisterAgent registers an agent with the server
func (s *ExtendedA2AServer) RegisterAgent(ctx context.Context, card *AgentCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if card.AgentID == "" {
		card.AgentID = uuid.New().String()
	}
	card.Timestamp = time.Now()

	s.agents[card.AgentID] = card

	slog.InfoContext(ctx, "Agent registered",
		slog.String("agent_id", card.AgentID),
		slog.String("name", card.Name),
	)

	return nil
}

// GetAgent returns an agent by ID
func (s *ExtendedA2AServer) GetAgent(agentID string) (*AgentCard, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	card, exists := s.agents[agentID]
	return card, exists
}

// ListAgents returns all registered agents
func (s *ExtendedA2AServer) ListAgents(capabilities []string) []*AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*AgentCard
	for _, agent := range s.agents {
		if len(capabilities) == 0 {
			result = append(result, agent)
			continue
		}
		// Filter by capabilities
		hasAll := true
		for _, cap := range capabilities {
			found := false
			for _, aCap := range agent.Capabilities {
				if aCap == cap {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, agent)
		}
	}
	return result
}

// CreateSession creates a new session
func (s *ExtendedA2AServer) CreateSession(agentID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:         uuid.New().String(),
		AgentID:    agentID,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		State:      make(map[string]interface{}),
	}

	s.sessions[session.ID] = session
	s.eventHandler.sessions[session.ID] = make(chan *StreamEvent, 100)

	return session, nil
}

// GetSession returns a session by ID
func (s *ExtendedA2AServer) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[sessionID]
	return session, exists
}

// UpdateSessionState updates session state
func (s *ExtendedA2AServer) UpdateSessionState(sessionID string, state map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastActive = time.Now()
	session.State = state

	return nil
}

// SubscribeSession creates a subscription to session events
func (s *ExtendedA2AServer) SubscribeSession(sessionID string) (<-chan *StreamEvent, error) {
	s.mu.RLock()
	_, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	s.mu.Lock()
	ch := make(chan *StreamEvent, 100)
	s.eventHandler.sessions[sessionID] = ch
	s.mu.Unlock()

	return ch, nil
}

// UnsubscribeSession removes a session subscription
func (s *ExtendedA2AServer) UnsubscribeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.eventHandler.sessions[sessionID]; exists {
		close(ch)
		delete(s.eventHandler.sessions, sessionID)
	}
}

// SendMessage sends an A2A message
func (s *ExtendedA2AServer) SendMessage(ctx context.Context, msg *A2AMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Queue message for processing
	select {
	case s.messageQueue <- msg:
		return nil
	default:
		return fmt.Errorf("message queue full")
	}
}

// SendMessageSync sends a message and waits for response
func (s *ExtendedA2AServer) SendMessageSync(ctx context.Context, msg *A2AMessage, timeout time.Duration) (*A2AMessage, error) {
	// Subscribe to session
	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	
	subCh, err := s.SubscribeSession(sessionID)
	if err != nil {
		return nil, err
	}
	defer s.UnsubscribeSession(sessionID)

	// Send message
	if err := s.SendMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Wait for response or timeout
	select {
	case resp := <-subCh:
		return &A2AMessage{
			ID:        resp.TaskID,
			Type:      MessageTypeTaskSubmit,
			Payload:   resp.Data,
			Timestamp: resp.Timestamp,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// StreamTask streams task events
func (s *ExtendedA2AServer) StreamTask(ctx context.Context, taskID string) (<-chan *StreamEvent, error) {
	session, exists := s.GetSession(taskID)
	if !exists {
		session, _ = s.CreateSession(taskID)
	}

	return s.SubscribeSession(session.ID)
}

// PublishEvent publishes an event to subscribers
func (s *ExtendedA2AServer) PublishEvent(sessionID string, event *StreamEvent) {
	s.mu.RLock()
	ch, exists := s.eventHandler.sessions[sessionID]
	s.mu.RUnlock()

	if exists {
		select {
		case ch <- event:
		default:
			// Channel full, drop event
		}
	}
}

// Extended task handlers
func (s *ExtendedA2AServer) handleAgentDiscovery(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	result := s.ListAgents(nil)
	task.Output = map[string]interface{}{
		"agents": result,
		"count":  len(result),
	}
	return task, nil
}

func (s *ExtendedA2AServer) handleAgentRegister(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	card := &AgentCard{}
	if data, ok := task.Input["agent_card"]; ok {
		if jsonData, err := json.Marshal(data); err == nil {
			json.Unmarshal(jsonData, card)
		}
	}

	if err := s.RegisterAgent(context.Background(), card); err != nil {
		return task, err
	}

	task.Output = map[string]interface{}{
		"agent_id": card.AgentID,
		"status":   "registered",
	}
	return task, nil
}

func (s *ExtendedA2AServer) handleAgentCard(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	agentID, _ := task.Input["agent_id"].(string)
	card, exists := s.GetAgent(agentID)

	if !exists {
		task.Error = fmt.Sprintf("agent not found: %s", agentID)
		task.Status = TaskFailed
		return task, nil
	}

	task.Output = map[string]interface{}{
		"agent_card": card,
	}
	return task, nil
}

// MessageBuilder helps build A2A messages
type MessageBuilder struct {
	message *A2AMessage
}

// NewMessage creates a new message builder
func NewMessage(msgType MessageType, from, to string) *MessageBuilder {
	return &MessageBuilder{
		message: &A2AMessage{
			ID:        uuid.New().String(),
			Type:      msgType,
			From:      from,
			To:        to,
			Payload:   make(map[string]interface{}),
			Metadata:  make(map[string]string),
			Timestamp: time.Now(),
		},
	}
}

// WithSession sets the session ID
func (b *MessageBuilder) WithSession(sessionID string) *MessageBuilder {
	b.message.SessionID = sessionID
	return b
}

// WithTask sets the task ID
func (b *MessageBuilder) WithTask(taskID string) *MessageBuilder {
	b.message.TaskID = taskID
	return b
}

// WithPayload adds payload data
func (b *MessageBuilder) WithPayload(key string, value interface{}) *MessageBuilder {
	b.message.Payload[key] = value
	return b
}

// WithMetadata adds metadata
func (b *MessageBuilder) WithMetadata(key, value string) *MessageBuilder {
	b.message.Metadata[key] = value
	return b
}

// Build returns the final message
func (b *MessageBuilder) Build() *A2AMessage {
	return b.message
}

// A2AProtocolStats returns protocol statistics
type A2AProtocolStats struct {
	TotalAgents    int            `json:"total_agents"`
	TotalSessions  int            `json:"total_sessions"`
	TotalTasks     int            `json:"total_tasks"`
	MessageQueueSize int          `json:"message_queue_size"`
	ActiveStreams   int           `json:"active_streams"`
}

// GetStats returns A2A protocol statistics
func (s *ExtendedA2AServer) GetStats() A2AProtocolStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return A2AProtocolStats{
		TotalAgents:     len(s.agents),
		TotalSessions:  len(s.sessions),
		TotalTasks:     len(s.tasks),
		MessageQueueSize: len(s.messageQueue),
		ActiveStreams:   len(s.eventHandler.sessions),
	}
}
