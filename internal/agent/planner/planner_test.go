package planner

import (
	"context"
	"testing"
	"time"
)

func TestGoal_NewGoal(t *testing.T) {
	goal := NewGoal("Test goal")

	if goal.ID == "" {
		t.Error("Expected goal ID to be generated")
	}
	if goal.Description != "Test goal" {
		t.Errorf("Expected description 'Test goal', got '%s'", goal.Description)
	}
	if goal.Status != GoalStatusPending {
		t.Errorf("Expected status Pending, got %s", goal.Status)
	}
}

func TestGoal_WithConstraint(t *testing.T) {
	goal := NewGoal("Test goal")
	goal.WithConstraint(ConstraintTypeTime, 0.8)

	if len(goal.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(goal.Constraints))
	}
	if goal.Constraints[0].Type != ConstraintTypeTime {
		t.Errorf("Expected constraint type Time, got %s", goal.Constraints[0].Type)
	}
}

func TestGoal_WithObjective(t *testing.T) {
	goal := NewGoal("Test goal")
	goal.WithObjective("First objective", 1)
	goal.WithObjective("Second objective", 2)

	if len(goal.Objectives) != 2 {
		t.Errorf("Expected 2 objectives, got %d", len(goal.Objectives))
	}
}

func TestGoal_IsComplete(t *testing.T) {
	goal := NewGoal("Test goal")
	goal.WithObjective("Objective 1", 1)

	if goal.IsComplete() {
		t.Error("Goal should not be complete when objectives are not done")
	}

	goal.Objectives[0].Completed = true

	if !goal.IsComplete() {
		t.Error("Goal should be complete when all objectives are done")
	}
}

func TestGoal_IsDeadlineMissed(t *testing.T) {
	goal := NewGoal("Test goal")

	if goal.IsDeadlineMissed() {
		t.Error("Goal without deadline should not be missed")
	}

	futureDeadline := time.Now().Add(time.Hour)
	goal.WithDeadline(futureDeadline)

	if goal.IsDeadlineMissed() {
		t.Error("Goal with future deadline should not be missed")
	}

	pastDeadline := time.Now().Add(-time.Hour)
	goal.WithDeadline(pastDeadline)

	if !goal.IsDeadlineMissed() {
		t.Error("Goal with past deadline should be missed")
	}
}

func TestTask_NewTask(t *testing.T) {
	task := NewTask("goal-123", "Test task")

	if task.ID == "" {
		t.Error("Expected task ID to be generated")
	}
	if task.GoalID != "goal-123" {
		t.Errorf("Expected GoalID 'goal-123', got '%s'", task.GoalID)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Expected status Pending, got %s", task.Status)
	}
}

func TestTask_WithDependencies(t *testing.T) {
	task := NewTask("goal-123", "Test task")
	task.WithDependencies("task-1", "task-2")

	if len(task.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}
}

func TestTask_CanExecute(t *testing.T) {
	task := NewTask("goal-123", "Test task")
	task.WithDependencies("task-1", "task-2")

	if task.CanExecute(map[string]bool{}) {
		t.Error("Task should not execute without dependencies")
	}

	if task.CanExecute(map[string]bool{"task-1": true}) {
		t.Error("Task should not execute with only one dependency")
	}

	if !task.CanExecute(map[string]bool{"task-1": true, "task-2": true}) {
		t.Error("Task should execute when all dependencies are complete")
	}
}

func TestDecomposer_DecomposeGoal(t *testing.T) {
	decomposer := NewDecomposer(10)

	goal := NewGoal("Search for information about AI")
	decomposer.DecomposeGoal(context.Background(), *goal)

	goal2 := NewGoal("Create a summary of the findings")
	tasks, err := decomposer.DecomposeGoal(context.Background(), *goal2)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(tasks) == 0 {
		t.Error("Expected at least one task")
	}
}

func TestTaskPrioritizer_PrioritizeTasks(t *testing.T) {
	prioritizer := NewTaskPrioritizer()

	tasks := []Task{
		*NewTask("goal-1", "Low priority task"),
		*NewTask("goal-1", "High priority task"),
		*NewTask("goal-1", "Medium priority task"),
	}

	tasks[0].Priority = 10
	tasks[1].Priority = 100
	tasks[2].Priority = 50

	prioritized, err := prioritizer.PrioritizeTasks(context.Background(), tasks)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(prioritized) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(prioritized))
	}

	if prioritized[0].Priority != 100 {
		t.Errorf("Expected first task to have priority 100, got %d", prioritized[0].Priority)
	}
}

func TestExecutionMonitor_RegisterTask(t *testing.T) {
	monitor := NewExecutionMonitor()
	task := NewTask("goal-123", "Test task")

	monitor.RegisterTask(task)

	metrics := monitor.GetMetrics()
	if metrics.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", metrics.TotalTasks)
	}
}

func TestExecutionMonitor_StartTask(t *testing.T) {
	monitor := NewExecutionMonitor()
	task := NewTask("goal-123", "Test task")
	monitor.RegisterTask(task)

	err := monitor.StartTask(task.ID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	status, _ := monitor.GetTaskStatus(task.ID)
	if status != TaskStatusRunning {
		t.Errorf("Expected status Running, got %s", status)
	}
}

func TestExecutionMonitor_CompleteTask(t *testing.T) {
	monitor := NewExecutionMonitor()
	task := NewTask("goal-123", "Test task")
	monitor.RegisterTask(task)
	monitor.StartTask(task.ID)

	result := TaskResult{Success: true, Output: "test output"}
	err := monitor.CompleteTask(task.ID, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	metrics := monitor.GetMetrics()
	if metrics.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", metrics.CompletedTasks)
	}
}

func TestExecutionMonitor_FailTask(t *testing.T) {
	monitor := NewExecutionMonitor()
	task := NewTask("goal-123", "Test task")
	monitor.RegisterTask(task)
	monitor.StartTask(task.ID)

	err := monitor.FailTask(task.ID, ErrInvalidTask)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	metrics := monitor.GetMetrics()
	if metrics.FailedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", metrics.FailedTasks)
	}
}

func TestSelfCorrector_Correct(t *testing.T) {
	corrector := NewSelfCorrector()
	task := NewTask("goal-123", "Test task")
	task.Timeout = time.Second

	feedback := Feedback{
		Type:    FeedbackTypeFailure,
		Message: "Task timed out",
		Metrics: map[string]float64{"execution_time_ms": 5000},
	}

	corrected := corrector.Correct(task, TaskResult{
		Success: false,
		Error:   "timeout",
		Metrics: feedback.Metrics,
	})

	if corrected == nil {
		t.Error("Expected corrected task, got nil")
	}
}

func TestSelfCorrector_GetStatistics(t *testing.T) {
	corrector := NewSelfCorrector()

	stats := corrector.GetStatistics()
	if stats["total_corrections"].(int) != 0 {
		t.Error("Expected 0 corrections initially")
	}
}

func TestTaskExecutor_ExecuteWithCorrection(t *testing.T) {
	executor := NewTaskExecutor(3, 5*time.Second)
	task := NewTask("goal-123", "Test execution task")
	task.WithTimeout(2 * time.Second)

	result, err := executor.ExecuteWithCorrection(context.Background(), task)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !result.Success {
		t.Error("Expected successful execution")
	}
}

func TestPlanAdaptor_AdaptPlan(t *testing.T) {
	adaptor := NewPlanAdaptor()

	goal := NewGoal("Test goal adaptation")
	feedback := Feedback{
		Type:    FeedbackTypeFailure,
		Message: "Previous attempt failed",
		Metrics: map[string]float64{"error_rate": 0.5},
	}

	adaptedTasks, err := adaptor.AdaptPlan(context.Background(), *goal, feedback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(adaptedTasks) == 0 {
		t.Error("Expected at least one adapted task")
	}
}

func TestPlanAdaptor_AdaptPlanSlow(t *testing.T) {
	adaptor := NewPlanAdaptor()

	goal := NewGoal("Test slow goal")
	feedback := Feedback{
		Type:    FeedbackTypeSlow,
		Message: "Task is too slow",
		Metrics: map[string]float64{"time_taken": 10000},
	}

	adaptedTasks, err := adaptor.AdaptPlan(context.Background(), *goal, feedback)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(adaptedTasks) == 0 {
		t.Error("Expected at least one adapted task for slow feedback")
	}
}
