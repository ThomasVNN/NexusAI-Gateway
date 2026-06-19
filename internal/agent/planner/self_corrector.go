package planner

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

// SelfCorrector handles self-correction of tasks based on feedback
type SelfCorrector struct {
	mu         sync.RWMutex
	strategies map[string]CorrectionStrategy
	history    []CorrectionRecord
	maxHistory int
}

// CorrectionStrategy defines how to correct a task
type CorrectionStrategy struct {
	Name         string
	Apply        func(task *Task, feedback Feedback) *Task
	Priority     int
	ApplicableTo []FeedbackType
}

// CorrectionRecord tracks correction history
type CorrectionRecord struct {
	TaskID        string       `json:"task_id"`
	FeedbackType  FeedbackType `json:"feedback_type"`
	OriginalTask  Task         `json:"original_task"`
	CorrectedTask *Task        `json:"corrected_task"`
	Timestamp     time.Time    `json:"timestamp"`
}

// NewSelfCorrector creates a new self-corrector
func NewSelfCorrector() *SelfCorrector {
	sc := &SelfCorrector{
		strategies: make(map[string]CorrectionStrategy),
		history:    make([]CorrectionRecord, 0),
		maxHistory: 100,
	}

	sc.registerStrategies()
	return sc
}

func (s *SelfCorrector) registerStrategies() {
	s.strategies["retry_timeout"] = CorrectionStrategy{
		Name:         "retry_timeout",
		Apply:        s.applyTimeoutCorrection,
		Priority:     1,
		ApplicableTo: []FeedbackType{FeedbackTypeFailure, FeedbackTypeSlow},
	}

	s.strategies["resource_increase"] = CorrectionStrategy{
		Name:         "resource_increase",
		Apply:        s.applyResourceCorrection,
		Priority:     2,
		ApplicableTo: []FeedbackType{FeedbackTypeSlow},
	}

	s.strategies["quality_improve"] = CorrectionStrategy{
		Name:         "quality_improve",
		Apply:        s.applyQualityCorrection,
		Priority:     3,
		ApplicableTo: []FeedbackType{FeedbackTypeQualityIssue},
	}

	s.strategies["simplify"] = CorrectionStrategy{
		Name:         "simplify",
		Apply:        s.applySimplifyCorrection,
		Priority:     4,
		ApplicableTo: []FeedbackType{FeedbackTypeFailure},
	}

	s.strategies["decompose"] = CorrectionStrategy{
		Name:         "decompose",
		Apply:        s.applyDecomposeCorrection,
		Priority:     5,
		ApplicableTo: []FeedbackType{FeedbackTypeQualityIssue, FeedbackTypeSlow},
	}

	s.strategies["user_feedback"] = CorrectionStrategy{
		Name:         "user_feedback",
		Apply:        s.applyUserFeedbackCorrection,
		Priority:     1,
		ApplicableTo: []FeedbackType{FeedbackTypeUserInput},
	}
}

// Correct applies corrections to a task based on feedback
func (s *SelfCorrector) Correct(task *Task, result TaskResult) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	feedback := s.resultToFeedback(task, result)

	slog.Info("Applying corrections",
		slog.String("task_id", task.ID),
		slog.String("feedback_type", string(feedback.Type)),
	)

	var bestCorrection *Task

	for _, strategy := range s.strategies {
		if s.isApplicable(strategy, feedback.Type) {
			corrected := strategy.Apply(task, feedback)
			if corrected != nil {
				bestCorrection = corrected
				s.recordCorrection(task, feedback, corrected)
				break
			}
		}
	}

	return bestCorrection
}

func (s *SelfCorrector) isApplicable(strategy CorrectionStrategy, feedbackType FeedbackType) bool {
	for _, ft := range strategy.ApplicableTo {
		if ft == feedbackType {
			return true
		}
	}
	return false
}

func (s *SelfCorrector) resultToFeedback(task *Task, result TaskResult) Feedback {
	var feedbackType FeedbackType
	var message string

	if !result.Success {
		if strings.Contains(result.Error, "timeout") {
			feedbackType = FeedbackTypeSlow
			message = "Task timed out"
		} else {
			feedbackType = FeedbackTypeFailure
			message = result.Error
		}
	} else if metrics := result.Metrics; metrics != nil {
		if timeMs, ok := metrics["execution_time_ms"]; ok && timeMs > 5000 {
			feedbackType = FeedbackTypeSlow
			message = "Task execution was slow"
		} else if quality, ok := metrics["quality_score"]; ok && quality < 0.7 {
			feedbackType = FeedbackTypeQualityIssue
			message = "Task quality below threshold"
		} else {
			feedbackType = FeedbackTypeSuccess
			message = "Task completed successfully"
		}
	} else {
		feedbackType = FeedbackTypeSuccess
		message = "Task completed successfully"
	}

	return Feedback{
		Type:    feedbackType,
		Message: message,
		Metrics: result.Metrics,
	}
}

func (s *SelfCorrector) recordCorrection(task *Task, feedback Feedback, corrected *Task) {
	record := CorrectionRecord{
		TaskID:        task.ID,
		FeedbackType:  feedback.Type,
		OriginalTask:  *task,
		CorrectedTask: corrected,
		Timestamp:     time.Now(),
	}

	s.history = append(s.history, record)

	if len(s.history) > s.maxHistory {
		s.history = s.history[len(s.history)-s.maxHistory:]
	}

	slog.Debug("Recorded correction",
		slog.String("task_id", task.ID),
		slog.String("strategy", corrected.Description),
	)
}

func (s *SelfCorrector) applyTimeoutCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "Retry: " + task.Description
	newTask.Timeout = task.Timeout * 2
	newTask.MaxRetries = task.MaxRetries + 1
	newTask.AddMetadata("correction", "timeout")
	newTask.AddMetadata("original_timeout", task.Timeout.String())
	return &newTask
}

func (s *SelfCorrector) applyResourceCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "Optimized: " + task.Description
	newTask.AddMetadata("correction", "resource_optimization")

	if task.Metadata == nil {
		newTask.Metadata = make(map[string]interface{})
	}
	newTask.Metadata["resource_level"] = "high"
	return &newTask
}

func (s *SelfCorrector) applyQualityCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "Refined: " + task.Description
	newTask.AddMetadata("correction", "quality_improvement")

	if task.Metadata == nil {
		newTask.Metadata = make(map[string]interface{})
	}
	newTask.Metadata["quality_mode"] = "enhanced"
	newTask.Metadata["validation_level"] = "strict"
	return &newTask
}

func (s *SelfCorrector) applySimplifyCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "Simplified: " + task.Description
	newTask.AddMetadata("correction", "simplification")

	if task.Metadata == nil {
		newTask.Metadata = make(map[string]interface{})
	}
	newTask.Metadata["complexity"] = "reduced"
	return &newTask
}

func (s *SelfCorrector) applyDecomposeCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "Split: " + task.Description
	newTask.AddMetadata("correction", "decomposition")

	if task.Metadata == nil {
		newTask.Metadata = make(map[string]interface{})
	}
	newTask.Metadata["subtasks"] = true
	newTask.Metadata["batch_size"] = 5
	return &newTask
}

func (s *SelfCorrector) applyUserFeedbackCorrection(task *Task, feedback Feedback) *Task {
	newTask := *task
	newTask.Description = "User revised: " + task.Description
	newTask.AddMetadata("correction", "user_feedback")
	newTask.AddMetadata("user_message", feedback.Message)
	return &newTask
}

// GetHistory returns the correction history
func (s *SelfCorrector) GetHistory() []CorrectionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CorrectionRecord, len(s.history))
	copy(result, s.history)
	return result
}

// GetStatistics returns correction statistics
func (s *SelfCorrector) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_corrections": len(s.history),
		"by_type":           map[string]int{},
	}

	typeCounts := make(map[string]int)
	for _, record := range s.history {
		typeCounts[string(record.FeedbackType)]++
	}
	stats["by_type"] = typeCounts

	return stats
}
