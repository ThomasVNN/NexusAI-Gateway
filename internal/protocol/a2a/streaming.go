package a2a

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SSEHandler handles Server-Sent Events for task streaming
type SSEHandler struct {
	mu       sync.RWMutex
	subs     map[string]chan *SSEMessage
	server   *Server
	logger   *slog.Logger
}

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(server *Server) *SSEHandler {
	return &SSEHandler{
		subs:   make(map[string]chan *SSEMessage),
		server: server,
		logger: slog.Default(),
	}
}

// Subscribe subscribes to task updates
func (h *SSEHandler) Subscribe(taskID string) (<-chan *SSEMessage, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan *SSEMessage, 100)
	h.subs[taskID] = ch

	// Return unsubscribe function
	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		close(ch)
		delete(h.subs, taskID)
	}

	return ch, unsubscribe
}

// SubscribeAll subscribes to all task updates
func (h *SSEHandler) SubscribeAll() (<-chan *SSEMessage, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan *SSEMessage, 100)
	allID := fmt.Sprintf("all-%d", time.Now().UnixNano())
	h.subs[allID] = ch

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		close(ch)
		delete(h.subs, allID)
	}

	return ch, unsubscribe
}

// Notify sends a message to subscribers
func (h *SSEHandler) Notify(taskID string, event string, data map[string]interface{}) {
	msg := &SSEMessage{
		Event: event,
		Data:  data,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Send to task-specific subscribers
	if ch, ok := h.subs[taskID]; ok {
		select {
		case ch <- msg:
		default:
			h.logger.Warn("SSE channel full, dropping message", slog.String("task_id", taskID))
		}
	}

	// Send to "all" subscribers
	if ch, ok := h.subs["all"]; ok {
		select {
		case ch <- msg:
		default:
		}
	}
}

// NotifyTaskCreated notifies subscribers of a new task
func (h *SSEHandler) NotifyTaskCreated(task *A2ATask) {
	h.Notify(task.ID, "task.created", map[string]interface{}{
		"task_id":   task.ID,
		"agent_id":  task.AgentID,
		"status":   task.Status.String(),
		"created_at": task.CreatedAt.Format(time.RFC3339),
	})
}

// NotifyTaskUpdated notifies subscribers of a task update
func (h *SSEHandler) NotifyTaskUpdated(task *A2ATask) {
	h.Notify(task.ID, "task.updated", map[string]interface{}{
		"task_id":   task.ID,
		"status":   task.Status.String(),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
	})
}

// NotifyTaskCompleted notifies subscribers of a completed task
func (h *SSEHandler) NotifyTaskCompleted(task *A2ATask) {
	h.Notify(task.ID, "task.completed", map[string]interface{}{
		"task_id":      task.ID,
		"output":       task.Output,
		"completed_at": task.CompletedAt.Format(time.RFC3339),
	})
}

// NotifyTaskFailed notifies subscribers of a failed task
func (h *SSEHandler) NotifyTaskFailed(task *A2ATask) {
	h.Notify(task.ID, "task.failed", map[string]interface{}{
		"task_id":  task.ID,
		"error":    task.Error,
		"failed_at": task.UpdatedAt.Format(time.RFC3339),
	})
}

// NotifyProgress notifies subscribers of task progress
func (h *SSEHandler) NotifyProgress(taskID string, progress float64, message string) {
	h.Notify(taskID, "task.progress", map[string]interface{}{
		"task_id":  taskID,
		"progress": progress,
		"message":  message,
	})
}

// FormatSSE formats a message for SSE delivery
func FormatSSE(event string, data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(jsonData)), nil
}

// StreamingServer wraps A2A server with streaming capabilities
type StreamingServer struct {
	*Server
	sse *SSEHandler
}

// NewStreamingServer creates a new streaming A2A server
func NewStreamingServer() *StreamingServer {
	server := NewServer()
	return &StreamingServer{
		Server: server,
		sse:    NewSSEHandler(server),
	}
}

// CreateTaskWithStream creates a task and starts streaming updates
func (s *StreamingServer) CreateTaskWithStream(agentID string, input map[string]interface{}) (*A2ATask, <-chan *SSEMessage, func()) {
	task, _ := s.Server.CreateTask(agentID, input)
	s.sse.NotifyTaskCreated(task)

	ch, unsub := s.sse.Subscribe(task.ID)
	return task, ch, unsub
}

// NotifyTaskUpdated sends a task update event
func (s *StreamingServer) NotifyTaskUpdated(task *A2ATask) {
	s.Server.UpdateTaskStatus(task.ID, task.Status)
	s.sse.NotifyTaskUpdated(task)
}

// NotifyTaskCompleted sends a task completion event
func (s *StreamingServer) NotifyTaskCompleted(task *A2ATask) {
	s.Server.SetTaskOutput(task.ID, task.Output)
	s.sse.NotifyTaskCompleted(task)
}

// NotifyTaskFailed sends a task failure event
func (s *StreamingServer) NotifyTaskFailed(taskID string, errMsg string) {
	s.Server.SetTaskError(taskID, errMsg)
	task, _ := s.Server.GetTask(taskID)
	if task != nil {
		s.sse.NotifyTaskFailed(task)
	}
}
