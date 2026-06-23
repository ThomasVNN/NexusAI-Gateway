package a2a

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// TaskStatus represents the status of an A2A task
type TaskStatus int

const (
	TaskSubmitted  TaskStatus = iota
	TaskWorking
	TaskCompleted
	TaskFailed
	TaskCancelled
)

// String returns the string representation of TaskStatus
func (s TaskStatus) String() string {
	switch s {
	case TaskSubmitted:
		return "submitted"
	case TaskWorking:
		return "working"
	case TaskCompleted:
		return "completed"
	case TaskFailed:
		return "failed"
	case TaskCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// A2ATask represents an A2A task
type A2ATask struct {
	ID          string                 `json:"id"`
	Status      TaskStatus             `json:"status"`
	AgentID     string                 `json:"agent_id"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Server represents the A2A server
type Server struct {
	mu       sync.RWMutex
	tasks    map[string]*A2ATask
	skills   map[string]*A2ASkill
	handlers map[string]TaskHandler

	streaming   *SSEHandler
	logger      *slog.Logger
}

// TaskHandler is the function signature for task handlers
type TaskHandler func(ctx interface{}, task *A2ATask) (*A2ATask, error)

// A2ASkill represents a built-in A2A skill
type A2ASkill struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Parameters  []string               `json:"parameters,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// NewServer creates a new A2A server
func NewServer() *Server {
	s := &Server{
		tasks:    make(map[string]*A2ATask),
		skills:   make(map[string]*A2ASkill),
		handlers: make(map[string]TaskHandler),
		logger:   slog.Default(),
	}

	s.registerBuiltInSkills()
	s.registerBuiltInHandlers()
	return s
}

// registerBuiltInSkills registers the built-in skills
func (s *Server) registerBuiltInSkills() {
	s.skills["smartRouting"] = &A2ASkill{
		Name:        "smartRouting",
		Description: "Intelligent routing based on context",
		Category:    "routing",
		Parameters:  []string{"request", "context"},
	}
	s.skills["quotaManagement"] = &A2ASkill{
		Name:        "quotaManagement",
		Description: "Manage quota and budgets",
		Category:    "budget",
		Parameters:  []string{"action", "params"},
	}
	s.skills["providerDiscovery"] = &A2ASkill{
		Name:        "providerDiscovery",
		Description: "Discover available providers",
		Category:    "provider",
		Parameters:  []string{"criteria"},
	}
	s.skills["costAnalysis"] = &A2ASkill{
		Name:        "costAnalysis",
		Description: "Analyze cost patterns",
		Category:    "budget",
		Parameters:  []string{"period", "dimensions"},
	}
	s.skills["healthReport"] = &A2ASkill{
		Name:        "healthReport",
		Description: "Generate health reports",
		Category:    "system",
		Parameters:  []string{"format", "components"},
	}
}

// registerBuiltInHandlers registers built-in task handlers
func (s *Server) registerBuiltInHandlers() {
	s.handlers["smartRouting"] = s.handleSmartRouting
	s.handlers["quotaManagement"] = s.handleQuotaManagement
	s.handlers["providerDiscovery"] = s.handleProviderDiscovery
	s.handlers["costAnalysis"] = s.handleCostAnalysis
	s.handlers["healthReport"] = s.handleHealthReport
}

// CreateTask creates a new task
func (s *Server) CreateTask(agentID string, input map[string]interface{}) (*A2ATask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &A2ATask{
		ID:        fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Status:    TaskSubmitted,
		AgentID:  agentID,
		Input:    input,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.tasks[task.ID] = task
	s.logger.Info("Task created", slog.String("task_id", task.ID), slog.String("agent_id", agentID))

	return task, nil
}

// GetTask retrieves a task by ID
func (s *Server) GetTask(taskID string) (*A2ATask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus updates the status of a task
func (s *Server) UpdateTaskStatus(taskID string, status TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = status
	task.UpdatedAt = time.Now()

	if status == TaskCompleted || status == TaskFailed || status == TaskCancelled {
		now := time.Now()
		task.CompletedAt = &now
	}

	s.logger.Info("Task status updated", slog.String("task_id", taskID), slog.String("status", status.String()))

	return nil
}

// SetTaskOutput sets the output of a completed task
func (s *Server) SetTaskOutput(taskID string, output map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Output = output
	task.Status = TaskCompleted
	task.UpdatedAt = time.Now()
	now := time.Now()
	task.CompletedAt = &now

	return nil
}

// SetTaskError sets the error of a failed task
func (s *Server) SetTaskError(taskID string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Error = errMsg
	task.Status = TaskFailed
	task.UpdatedAt = time.Now()
	now := time.Now()
	task.CompletedAt = &now

	return nil
}

// ListTasks returns all tasks, optionally filtered by status
func (s *Server) ListTasks(status *TaskStatus) []*A2ATask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*A2ATask
	for _, task := range s.tasks {
		if status == nil || task.Status == *status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// CancelTask cancels a task
func (s *Server) CancelTask(taskID string) error {
	return s.UpdateTaskStatus(taskID, TaskCancelled)
}

// DeleteTask removes a task
func (s *Server) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[taskID]; !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	delete(s.tasks, taskID)
	s.logger.Info("Task deleted", slog.String("task_id", taskID))

	return nil
}

// RegisterHandler registers a task handler
func (s *Server) RegisterHandler(skillName string, handler TaskHandler) {
	s.handlers[skillName] = handler
}

// ExecuteTask executes a task
func (s *Server) ExecuteTask(taskID string) error {
	s.mu.Lock()
	task, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Status = TaskWorking
	task.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Execute asynchronously
	go func() {
		if handler, ok := s.handlers[task.AgentID]; ok {
			result, err := handler(nil, task)
			if err != nil {
				s.SetTaskError(taskID, err.Error())
			} else if result != nil {
				s.SetTaskOutput(taskID, result.Output)
			}
		} else {
			// Default handler
			s.SetTaskOutput(taskID, map[string]interface{}{
				"result": "Task executed",
				"task_id": taskID,
			})
		}
	}()

	return nil
}

// ListSkills returns all skills
func (s *Server) ListSkills() []*A2ASkill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*A2ASkill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}
	return skills
}

// GetSkill retrieves a skill by name
func (s *Server) GetSkill(name string) (*A2ASkill, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, exists := s.skills[name]
	return skill, exists
}

// Built-in task handlers
func (s *Server) handleSmartRouting(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	task.Output = map[string]interface{}{
		"provider":   "openai",
		"model":      "gpt-4o",
		"confidence": 0.95,
		"reasoning": "Optimal for request type",
	}
	return task, nil
}

func (s *Server) handleQuotaManagement(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	task.Output = map[string]interface{}{
		"used":      50000,
		"limit":     100000,
		"remaining": 50000,
	}
	return task, nil
}

func (s *Server) handleProviderDiscovery(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	task.Output = map[string]interface{}{
		"providers": []map[string]string{
			{"id": "openai", "name": "OpenAI"},
			{"id": "anthropic", "name": "Anthropic"},
			{"id": "google", "name": "Google AI"},
		},
	}
	return task, nil
}

func (s *Server) handleCostAnalysis(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	task.Output = map[string]interface{}{
		"total_cost":    1250.50,
		"period":        "30d",
		"cost_by_model": map[string]float64{
			"gpt-4o":    500.00,
			"claude-3-5": 450.50,
			"gemini-2":  300.00,
		},
	}
	return task, nil
}

func (s *Server) handleHealthReport(ctx interface{}, task *A2ATask) (*A2ATask, error) {
	task.Output = map[string]interface{}{
		"healthy":     true,
		"uptime":       "99.9%",
		"components": map[string]string{
			"api":        "healthy",
			"database":   "healthy",
			"providers":  "healthy",
		},
	}
	return task, nil
}

// ToJSON converts a task to JSON
func (t *A2ATask) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// TaskFromJSON creates a task from JSON
func TaskFromJSON(data []byte) (*A2ATask, error) {
	var task A2ATask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}
