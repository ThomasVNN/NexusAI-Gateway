package planner

import (
	"context"
	"errors"
	"time"
)

// Goal represents a high-level goal for the agent
type Goal struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Constraints []Constraint           `json:"constraints"`
	Objectives  []Objective            `json:"objectives"`
	Status      GoalStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GoalStatus represents the status of a goal
type GoalStatus string

const (
	GoalStatusPending   GoalStatus = "pending"
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusFailed    GoalStatus = "failed"
	GoalStatusCancelled GoalStatus = "cancelled"
)

// Constraint represents a constraint on a goal
type Constraint struct {
	Type     ConstraintType `json:"type"`
	Weight   float32        `json:"weight"`
	Priority int            `json:"priority"`
}

// ConstraintType represents the type of constraint
type ConstraintType string

const (
	ConstraintTypeTime    ConstraintType = "time"
	ConstraintTypeQuality ConstraintType = "quality"
	ConstraintTypeCost    ConstraintType = "cost"
	ConstraintTypeRisk    ConstraintType = "risk"
)

// Objective represents a specific objective within a goal
type Objective struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Completed   bool     `json:"completed"`
	Priority    int      `json:"priority"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

// Task represents an actionable task derived from a goal
type Task struct {
	ID           string                 `json:"id"`
	GoalID       string                 `json:"goal_id"`
	Description  string                 `json:"description"`
	Status       TaskStatus             `json:"status"`
	Priority     int                    `json:"priority"`
	Dependencies []string               `json:"dependencies,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	Timeout      time.Duration          `json:"timeout"`
	Result       *TaskResult            `json:"result,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusReady     TaskStatus = "ready"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusRetrying  TaskStatus = "retrying"
	TaskStatusSkipped   TaskStatus = "skipped"
)

// TaskResult represents the result of a task execution
type TaskResult struct {
	Success bool               `json:"success"`
	Output  interface{}        `json:"output,omitempty"`
	Error   string             `json:"error,omitempty"`
	Metrics map[string]float64 `json:"metrics,omitempty"`
}

// Planner defines the interface for planning operations
type Planner interface {
	DecomposeGoal(ctx context.Context, goal Goal) ([]Task, error)
	PrioritizeTasks(ctx context.Context, tasks []Task) ([]Task, error)
	ExecuteWithCorrection(ctx context.Context, task Task) (TaskResult, error)
	MonitorExecution(ctx context.Context, taskID string) (TaskStatus, error)
	AdaptPlan(ctx context.Context, goalID string, feedback Feedback) ([]Task, error)
}

// Feedback represents feedback for plan adaptation
type Feedback struct {
	Type    FeedbackType       `json:"type"`
	Message string             `json:"message"`
	Metrics map[string]float64 `json:"metrics,omitempty"`
}

// FeedbackType represents the type of feedback
type FeedbackType string

const (
	FeedbackTypeSuccess      FeedbackType = "success"
	FeedbackTypeFailure      FeedbackType = "failure"
	FeedbackTypeSlow         FeedbackType = "slow"
	FeedbackTypeQualityIssue FeedbackType = "quality_issue"
	FeedbackTypeUserInput    FeedbackType = "user_input"
)

// NewGoal creates a new goal
func NewGoal(description string) *Goal {
	now := time.Now()
	return &Goal{
		ID:          generateGoalID(),
		Description: description,
		Status:      GoalStatusPending,
		CreatedAt:   now,
		Constraints: make([]Constraint, 0),
		Objectives:  make([]Objective, 0),
		Metadata:    make(map[string]interface{}),
	}
}

// WithConstraint adds a constraint to the goal
func (g *Goal) WithConstraint(constraintType ConstraintType, weight float32) *Goal {
	g.Constraints = append(g.Constraints, Constraint{
		Type:   constraintType,
		Weight: weight,
	})
	return g
}

// WithObjective adds an objective to the goal
func (g *Goal) WithObjective(description string, priority int) *Goal {
	objective := Objective{
		ID:          generateObjectiveID(),
		Description: description,
		Completed:   false,
		Priority:    priority,
	}
	g.Objectives = append(g.Objectives, objective)
	return g
}

// WithDeadline sets a deadline for the goal
func (g *Goal) WithDeadline(deadline time.Time) *Goal {
	g.Deadline = &deadline
	return g
}

// AddMetadata adds metadata to the goal
func (g *Goal) AddMetadata(key string, value interface{}) {
	g.Metadata[key] = value
}

// NewTask creates a new task
func NewTask(goalID, description string) *Task {
	now := time.Now()
	return &Task{
		ID:          generateTaskID(),
		GoalID:      goalID,
		Description: description,
		Status:      TaskStatusPending,
		Priority:    0,
		RetryCount:  0,
		MaxRetries:  3,
		Timeout:     5 * time.Minute,
		CreatedAt:   now,
		Metadata:    make(map[string]interface{}),
	}
}

// WithPriority sets the priority of a task
func (t *Task) WithPriority(priority int) *Task {
	t.Priority = priority
	return t
}

// WithDependencies adds dependencies to a task
func (t *Task) WithDependencies(deps ...string) *Task {
	t.Dependencies = append(t.Dependencies, deps...)
	return t
}

// WithTimeout sets the timeout for a task
func (t *Task) WithTimeout(timeout time.Duration) *Task {
	t.Timeout = timeout
	return t
}

// WithMaxRetries sets the maximum retries for a task
func (t *Task) WithMaxRetries(maxRetries int) *Task {
	t.MaxRetries = maxRetries
	return t
}

// AddMetadata adds metadata to a task
func (t *Task) AddMetadata(key string, value interface{}) {
	t.Metadata[key] = value
}

// CanExecute checks if a task can be executed based on dependencies
func (t *Task) CanExecute(completedTasks map[string]bool) bool {
	for _, dep := range t.Dependencies {
		if !completedTasks[dep] {
			return false
		}
	}
	return true
}

// IsDeadlineMissed checks if the goal deadline has been missed
func (g *Goal) IsDeadlineMissed() bool {
	if g.Deadline == nil {
		return false
	}
	return time.Now().After(*g.Deadline)
}

// IsComplete checks if all objectives are complete
func (g *Goal) IsComplete() bool {
	for _, obj := range g.Objectives {
		if !obj.Completed {
			return false
		}
	}
	return true
}

func generateGoalID() string {
	return "goal-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

func generateObjectiveID() string {
	return "obj-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

func generateTaskID() string {
	return "task-" + time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

// Common errors
var (
	ErrGoalNotFound       = errors.New("goal not found")
	ErrTaskNotFound       = errors.New("task not found")
	ErrInvalidGoal        = errors.New("invalid goal")
	ErrInvalidTask        = errors.New("invalid task")
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
	ErrDeadlineMissed     = errors.New("deadline missed")
)
