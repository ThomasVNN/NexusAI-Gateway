package planner

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// Decomposer handles goal decomposition
type Decomposer struct {
	maxSubtasks int
}

// NewDecomposer creates a new goal decomposer
func NewDecomposer(maxSubtasks int) *Decomposer {
	if maxSubtasks <= 0 {
		maxSubtasks = 10
	}
	return &Decomposer{
		maxSubtasks: maxSubtasks,
	}
}

// DecomposeGoal breaks down a goal into tasks
func (d *Decomposer) DecomposeGoal(ctx context.Context, goal Goal) ([]Task, error) {
	if goal.Description == "" {
		return nil, ErrInvalidGoal
	}

	var tasks []Task

	tasks = append(tasks, *d.createAnalysisTask(goal))
	tasks = append(tasks, *d.createPlanningTask(goal))
	executionTasks := d.createExecutionTasks(goal)
	for i := range executionTasks {
		tasks = append(tasks, *executionTasks[i])
	}
	tasks = append(tasks, *d.createValidationTask(goal))

	if goal.IsDeadlineMissed() {
		tasks = append(tasks, *d.createRecoveryTask(goal))
	}

	analyzer := NewTaskPrioritizer()
	tasks, _ = analyzer.PrioritizeTasks(ctx, tasks)

	return tasks, nil
}

func (d *Decomposer) createAnalysisTask(goal Goal) *Task {
	task := NewTask(goal.ID, fmt.Sprintf("Analyze goal: %s", goal.Description))
	task.WithPriority(100)
	task.AddMetadata("phase", "analysis")
	return task
}

func (d *Decomposer) createPlanningTask(goal Goal) *Task {
	task := NewTask(goal.ID, fmt.Sprintf("Create plan for: %s", goal.Description))
	task.WithPriority(90)
	task.WithDependencies(d.findTaskID(goal.ID, "analysis"))
	task.AddMetadata("phase", "planning")
	return task
}

func (d *Decomposer) createExecutionTasks(goal Goal) []*Task {
	var tasks []*Task

	description := strings.ToLower(goal.Description)

	if containsAny(description, []string{"search", "find", "retrieve", "fetch"}) {
		tasks = append(tasks, d.createRetrievalTask(goal))
	}

	if containsAny(description, []string{"create", "generate", "build", "make"}) {
		tasks = append(tasks, d.createGenerationTask(goal))
	}

	if containsAny(description, []string{"update", "modify", "change", "edit"}) {
		tasks = append(tasks, d.createModificationTask(goal))
	}

	if containsAny(description, []string{"delete", "remove", "clear"}) {
		tasks = append(tasks, d.createDeletionTask(goal))
	}

	if containsAny(description, []string{"analyze", "evaluate", "assess"}) {
		tasks = append(tasks, d.createAnalysisSubtask(goal))
	}

	if containsAny(description, []string{"compare", "contrast", "differentiate"}) {
		tasks = append(tasks, d.createComparisonTask(goal))
	}

	if containsAny(description, []string{"summarize", "summarise", "condense"}) {
		tasks = append(tasks, d.createSummaryTask(goal))
	}

	if containsAny(description, []string{"explain", "describe", "clarify"}) {
		tasks = append(tasks, d.createExplanationTask(goal))
	}

	if len(tasks) == 0 {
		tasks = append(tasks, d.createGenericExecutionTask(goal))
	}

	return tasks
}

func (d *Decomposer) createRetrievalTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Retrieve relevant information")
	task.WithPriority(80)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(2 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "retrieval")
	return task
}

func (d *Decomposer) createGenerationTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Generate output based on retrieved information")
	task.WithPriority(70)
	task.WithDependencies(d.findTaskID(goal.ID, "execution"))
	task.WithTimeout(3 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "generation")
	return task
}

func (d *Decomposer) createModificationTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Modify existing content")
	task.WithPriority(75)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(2 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "modification")
	return task
}

func (d *Decomposer) createDeletionTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Remove target content")
	task.WithPriority(60)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(1 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "deletion")
	return task
}

func (d *Decomposer) createAnalysisSubtask(goal Goal) *Task {
	task := NewTask(goal.ID, "Perform detailed analysis")
	task.WithPriority(70)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(3 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "analysis")
	return task
}

func (d *Decomposer) createComparisonTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Compare and contrast elements")
	task.WithPriority(70)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(3 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "comparison")
	return task
}

func (d *Decomposer) createSummaryTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Generate summary")
	task.WithPriority(65)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(2 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "summary")
	return task
}

func (d *Decomposer) createExplanationTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Provide explanation")
	task.WithPriority(65)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(2 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "explanation")
	return task
}

func (d *Decomposer) createGenericExecutionTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Execute main task")
	task.WithPriority(70)
	task.WithDependencies(d.findTaskID(goal.ID, "planning"))
	task.WithTimeout(5 * time.Minute)
	task.AddMetadata("phase", "execution")
	task.AddMetadata("type", "generic")
	return task
}

func (d *Decomposer) createValidationTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Validate results against constraints")
	task.WithPriority(50)
	task.WithTimeout(1 * time.Minute)
	task.AddMetadata("phase", "validation")
	return task
}

func (d *Decomposer) createRecoveryTask(goal Goal) *Task {
	task := NewTask(goal.ID, "Recovery from deadline miss")
	task.WithPriority(200)
	task.AddMetadata("phase", "recovery")
	task.AddMetadata("reason", "deadline_missed")
	return task
}

func (d *Decomposer) findTaskID(goalID, phase string) string {
	return fmt.Sprintf("%s-%s", goalID, phase)
}

func containsAny(s string, substrings []string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// TaskPrioritizer handles task prioritization
type TaskPrioritizer struct {
	mu sync.Mutex
}

// NewTaskPrioritizer creates a new task prioritizer
func NewTaskPrioritizer() *TaskPrioritizer {
	return &TaskPrioritizer{}
}

// PrioritizeTasks sorts tasks by priority
func (p *TaskPrioritizer) PrioritizeTasks(ctx context.Context, tasks []Task) ([]Task, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	sort.Slice(tasks, func(i, j int) bool {
		return p.compareTasks(&tasks[i], &tasks[j]) < 0
	})

	for i := range tasks {
		if tasks[i].CanExecute(p.getCompletedTasksMap(tasks[:i])) {
			tasks[i].Status = TaskStatusReady
		}
	}

	return tasks, nil
}

func (p *TaskPrioritizer) compareTasks(a, b *Task) int {
	if a.Priority != b.Priority {
		return b.Priority - a.Priority
	}

	if len(a.Dependencies) != len(b.Dependencies) {
		return len(a.Dependencies) - len(b.Dependencies)
	}

	return 0
}

func (p *TaskPrioritizer) getCompletedTasksMap(completedTasks []Task) map[string]bool {
	result := make(map[string]bool)
	for _, task := range completedTasks {
		if task.Status == TaskStatusCompleted {
			result[task.ID] = true
		}
	}
	return result
}

// ExecutionMonitor monitors task execution
type ExecutionMonitor struct {
	tasks   map[string]*Task
	mu      sync.RWMutex
	metrics *ExecutionMetrics
}

// ExecutionMetrics tracks execution metrics
type ExecutionMetrics struct {
	TotalTasks      int                `json:"total_tasks"`
	CompletedTasks  int                `json:"completed_tasks"`
	FailedTasks     int                `json:"failed_tasks"`
	AverageDuration time.Duration      `json:"average_duration"`
	TaskDurations   map[string]float64 `json:"task_durations"`
}

// NewExecutionMonitor creates a new execution monitor
func NewExecutionMonitor() *ExecutionMonitor {
	return &ExecutionMonitor{
		tasks: make(map[string]*Task),
		metrics: &ExecutionMetrics{
			TaskDurations: make(map[string]float64),
		},
	}
}

// RegisterTask registers a task with the monitor
func (m *ExecutionMonitor) RegisterTask(task *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
	m.metrics.TotalTasks++
}

// StartTask marks a task as started
func (m *ExecutionMonitor) StartTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now

	slog.Debug("Task started", slog.String("task_id", taskID))
	return nil
}

// CompleteTask marks a task as completed
func (m *ExecutionMonitor) CompleteTask(taskID string, result TaskResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	now := time.Now()
	task.Status = TaskStatusCompleted
	task.CompletedAt = &now
	task.Result = &result
	m.metrics.CompletedTasks++

	if task.StartedAt != nil {
		duration := now.Sub(*task.StartedAt)
		m.metrics.TaskDurations[taskID] = duration.Seconds()
		m.updateAverageDuration()
	}

	slog.Debug("Task completed",
		slog.String("task_id", taskID),
		slog.Bool("success", result.Success),
	)
	return nil
}

// FailTask marks a task as failed
func (m *ExecutionMonitor) FailTask(taskID string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return ErrTaskNotFound
	}

	now := time.Now()
	task.Status = TaskStatusFailed
	task.CompletedAt = &now
	task.Result = &TaskResult{
		Success: false,
		Error:   err.Error(),
	}
	m.metrics.FailedTasks++

	if task.StartedAt != nil {
		duration := now.Sub(*task.StartedAt)
		m.metrics.TaskDurations[taskID] = duration.Seconds()
		m.updateAverageDuration()
	}

	slog.Debug("Task failed",
		slog.String("task_id", taskID),
		slog.String("error", err.Error()),
	)
	return nil
}

// GetTaskStatus returns the current status of a task
func (m *ExecutionMonitor) GetTaskStatus(taskID string) (TaskStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return "", ErrTaskNotFound
	}

	return task.Status, nil
}

// GetMetrics returns execution metrics
func (m *ExecutionMonitor) GetMetrics() ExecutionMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ExecutionMetrics{
		TotalTasks:      m.metrics.TotalTasks,
		CompletedTasks:  m.metrics.CompletedTasks,
		FailedTasks:     m.metrics.FailedTasks,
		AverageDuration: m.metrics.AverageDuration,
		TaskDurations:   m.metrics.TaskDurations,
	}
}

func (m *ExecutionMonitor) updateAverageDuration() {
	if m.metrics.CompletedTasks == 0 {
		return
	}

	var totalDuration float64
	for _, duration := range m.metrics.TaskDurations {
		totalDuration += duration
	}

	completedCount := float64(m.metrics.CompletedTasks)
	m.metrics.AverageDuration = time.Duration(totalDuration/completedCount) * time.Second
}
