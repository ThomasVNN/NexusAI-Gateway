package planner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// TaskExecutor executes tasks with self-correction capabilities
type TaskExecutor struct {
	monitor          *ExecutionMonitor
	selfCorrector    *SelfCorrector
	maxRetries       int
	executionTimeout time.Duration
	mu               sync.RWMutex
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(maxRetries int, executionTimeout time.Duration) *TaskExecutor {
	return &TaskExecutor{
		monitor:          NewExecutionMonitor(),
		selfCorrector:    NewSelfCorrector(),
		maxRetries:       maxRetries,
		executionTimeout: executionTimeout,
	}
}

// ExecuteWithCorrection executes a task with automatic self-correction
func (e *TaskExecutor) ExecuteWithCorrection(ctx context.Context, task *Task) (TaskResult, error) {
	e.mu.Lock()
	e.monitor.RegisterTask(task)
	e.mu.Unlock()

	if err := e.monitor.StartTask(task.ID); err != nil {
		return TaskResult{}, err
	}

	var lastErr error
	for attempt := 0; attempt <= task.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return TaskResult{}, ctx.Err()
		default:
		}

		if attempt > 0 {
			slog.Info("Retrying task",
				slog.String("task_id", task.ID),
				slog.Int("attempt", attempt),
				slog.Int("max_retries", task.MaxRetries),
			)
			task.Status = TaskStatusRetrying
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		result := e.executeTask(ctx, task)

		if result.Success {
			if err := e.monitor.CompleteTask(task.ID, result); err != nil {
				return result, err
			}
			return result, nil
		}

		lastErr = fmt.Errorf("%s", result.Error)

		if !e.shouldRetry(task, result) {
			if err := e.monitor.FailTask(task.ID, lastErr); err != nil {
				return result, err
			}
			return result, lastErr
		}

		corrected := e.selfCorrector.Correct(task, result)
		if corrected != nil {
			*task = *corrected
			slog.Info("Task corrected based on feedback",
				slog.String("task_id", task.ID),
			)
		}
	}

	if err := e.monitor.FailTask(task.ID, lastErr); err != nil {
		return TaskResult{}, err
	}
	return TaskResult{Success: false, Error: lastErr.Error()}, ErrMaxRetriesExceeded
}

func (e *TaskExecutor) executeTask(ctx context.Context, task *Task) TaskResult {
	executionCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	done := make(chan TaskResult, 1)

	go func() {
		result := e.performTask(executionCtx, task)
		done <- result
	}()

	select {
	case <-executionCtx.Done():
		return TaskResult{
			Success: false,
			Error:   fmt.Sprintf("task timed out after %v", task.Timeout),
		}
	case result := <-done:
		return result
	}
}

func (e *TaskExecutor) performTask(ctx context.Context, task *Task) TaskResult {
	slog.Debug("Executing task",
		slog.String("task_id", task.ID),
		slog.String("description", task.Description),
		slog.String("type", fmt.Sprintf("%v", task.Metadata["type"])),
	)

	time.Sleep(100 * time.Millisecond)

	taskType, ok := task.Metadata["type"].(string)
	if !ok {
		taskType = "generic"
	}

	switch taskType {
	case "retrieval":
		return e.executeRetrievalTask(ctx, task)
	case "generation":
		return e.executeGenerationTask(ctx, task)
	case "modification":
		return e.executeModificationTask(ctx, task)
	case "deletion":
		return e.executeDeletionTask(ctx, task)
	case "analysis":
		return e.executeAnalysisTask(ctx, task)
	case "comparison":
		return e.executeComparisonTask(ctx, task)
	case "summary":
		return e.executeSummaryTask(ctx, task)
	case "explanation":
		return e.executeExplanationTask(ctx, task)
	default:
		return e.executeGenericTask(ctx, task)
	}
}

func (e *TaskExecutor) executeRetrievalTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"retrieved": true, "count": 10},
		Metrics: map[string]float64{"retrieval_time_ms": 150},
	}
}

func (e *TaskExecutor) executeGenerationTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"generated": true, "tokens": 500},
		Metrics: map[string]float64{"generation_time_ms": 2000},
	}
}

func (e *TaskExecutor) executeModificationTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"modified": true, "changes": 5},
		Metrics: map[string]float64{"modification_time_ms": 500},
	}
}

func (e *TaskExecutor) executeDeletionTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"deleted": true},
		Metrics: map[string]float64{"deletion_time_ms": 100},
	}
}

func (e *TaskExecutor) executeAnalysisTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"analyzed": true, "insights": 3},
		Metrics: map[string]float64{"analysis_time_ms": 1500},
	}
}

func (e *TaskExecutor) executeComparisonTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"compared": true, "differences": 7},
		Metrics: map[string]float64{"comparison_time_ms": 800},
	}
}

func (e *TaskExecutor) executeSummaryTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"summarized": true, "word_count": 150},
		Metrics: map[string]float64{"summary_time_ms": 600},
	}
}

func (e *TaskExecutor) executeExplanationTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"explained": true, "clarity_score": 0.9},
		Metrics: map[string]float64{"explanation_time_ms": 400},
	}
}

func (e *TaskExecutor) executeGenericTask(ctx context.Context, task *Task) TaskResult {
	return TaskResult{
		Success: true,
		Output:  map[string]interface{}{"executed": true},
		Metrics: map[string]float64{"execution_time_ms": 1000},
	}
}

func (e *TaskExecutor) shouldRetry(task *Task, result TaskResult) bool {
	if result.Success {
		return false
	}

	if task.RetryCount >= task.MaxRetries {
		return false
	}

	task.RetryCount++
	return true
}

// MonitorExecution monitors the execution of a task
func (e *TaskExecutor) MonitorExecution(ctx context.Context, taskID string) (TaskStatus, error) {
	return e.monitor.GetTaskStatus(taskID)
}

// GetMetrics returns execution metrics
func (e *TaskExecutor) GetMetrics() ExecutionMetrics {
	return e.monitor.GetMetrics()
}

// PlanAdaptor handles plan adaptation based on feedback
type PlanAdaptor struct {
	decomposer *Decomposer
	mu         sync.RWMutex
}

// NewPlanAdaptor creates a new plan adaptor
func NewPlanAdaptor() *PlanAdaptor {
	return &PlanAdaptor{
		decomposer: NewDecomposer(10),
	}
}

// AdaptPlan adapts a plan based on feedback
func (a *PlanAdaptor) AdaptPlan(ctx context.Context, goal Goal, feedback Feedback) ([]Task, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	slog.Info("Adapting plan",
		slog.String("goal_id", goal.ID),
		slog.String("feedback_type", string(feedback.Type)),
		slog.String("message", feedback.Message),
	)

	var result []Task

	switch feedback.Type {
	case FeedbackTypeFailure:
		result = a.handleFailureAdaptation(goal, feedback)
	case FeedbackTypeSlow:
		result = a.handleSlowAdaptation(goal, feedback)
	case FeedbackTypeQualityIssue:
		result = a.handleQualityAdaptation(goal, feedback)
	case FeedbackTypeUserInput:
		result = a.handleUserInputAdaptation(goal, feedback)
	default:
		return nil, fmt.Errorf("unknown feedback type: %s", feedback.Type)
	}

	return result, nil
}

func (a *PlanAdaptor) handleFailureAdaptation(goal Goal, feedback Feedback) []Task {
	slog.Info("Handling failure adaptation", slog.String("goal_id", goal.ID))

	recoveryTask := NewTask(goal.ID, "Recovery from failure: "+feedback.Message)
	recoveryTask.WithPriority(150)
	recoveryTask.AddMetadata("phase", "recovery")
	recoveryTask.AddMetadata("reason", "failure")
	recoveryTask.AddMetadata("metrics", feedback.Metrics)

	analysisTask := NewTask(goal.ID, "Analyze failure root cause")
	analysisTask.WithPriority(140)
	analysisTask.WithDependencies(recoveryTask.ID)
	analysisTask.AddMetadata("phase", "analysis")

	return []Task{*recoveryTask, *analysisTask}
}

func (a *PlanAdaptor) handleSlowAdaptation(goal Goal, feedback Feedback) []Task {
	slog.Info("Handling slow adaptation", slog.String("goal_id", goal.ID))

	if metrics, ok := feedback.Metrics["time_taken"]; ok {
		slog.Info("Task took too long", slog.Float64("time_taken", metrics))
	}

	optimizationTask := NewTask(goal.ID, "Optimize slow task")
	optimizationTask.WithPriority(120)
	optimizationTask.AddMetadata("phase", "optimization")
	optimizationTask.AddMetadata("reason", "performance")

	return []Task{*optimizationTask}
}

func (a *PlanAdaptor) handleQualityAdaptation(goal Goal, feedback Feedback) []Task {
	slog.Info("Handling quality adaptation", slog.String("goal_id", goal.ID))

	revisionTask := NewTask(goal.ID, "Revise output for quality: "+feedback.Message)
	revisionTask.WithPriority(130)
	revisionTask.AddMetadata("phase", "revision")
	revisionTask.AddMetadata("reason", "quality")

	validationTask := NewTask(goal.ID, "Enhanced validation")
	validationTask.WithPriority(125)
	validationTask.AddMetadata("phase", "validation")
	validationTask.AddMetadata("reason", "quality")

	return []Task{*revisionTask, *validationTask}
}

func (a *PlanAdaptor) handleUserInputAdaptation(goal Goal, feedback Feedback) []Task {
	slog.Info("Handling user input adaptation", slog.String("goal_id", goal.ID))

	incorporateTask := NewTask(goal.ID, "Incorporate user feedback: "+feedback.Message)
	incorporateTask.WithPriority(160)
	incorporateTask.AddMetadata("phase", "incorporation")
	incorporateTask.AddMetadata("reason", "user_input")

	return []Task{*incorporateTask}
}
