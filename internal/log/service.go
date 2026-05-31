package log

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Service provides request log business logic
type Service struct {
	repo *Repository
}

// NewService creates a new request log service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// LogRequest records a new request log entry
func (s *Service) LogRequest(ctx context.Context, log *RequestLog) error {
	if err := log.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.repo.Create(ctx, log)
}

// LogRequestFromContext creates a log entry from request context
func (s *Service) LogRequestFromContext(ctx context.Context, req RequestContext) error {
	log := &RequestLog{
		ChannelID:    req.ChannelID,
		Model:        req.Model,
		TokenGroupID: req.TokenGroupID,
		UserID:       req.UserID,
		APIKeyID:     req.APIKeyID,
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		LatencyMS:    req.LatencyMS,
		Status:       req.Status,
		ErrorMessage: req.ErrorMessage,
		Provider:     req.Provider,
		RequestID:    req.RequestID,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ModelRaw:     req.ModelRaw,
	}

	return s.LogRequest(ctx, log)
}

// GetByID retrieves a request log by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*RequestLog, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves request logs with filtering
func (s *Service) List(ctx context.Context, filter *RequestLogFilter) ([]*RequestLog, error) {
	if filter == nil {
		filter = &RequestLogFilter{}
	}

	// Default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	return s.repo.List(ctx, filter)
}

// GetStats retrieves aggregated statistics
func (s *Service) GetStats(ctx context.Context, filter *RequestLogFilter) (*RequestLogStats, error) {
	return s.repo.GetStats(ctx, filter)
}

// GetModelUsage retrieves model usage statistics
func (s *Service) GetModelUsage(ctx context.Context, days int) ([]ModelUsageStats, error) {
	if days <= 0 {
		days = 30
	}

	startTime := time.Now().AddDate(0, 0, -days)
	return s.repo.GetModelUsage(ctx, &startTime, nil, 20)
}

// GetChannelUsage retrieves channel usage statistics
func (s *Service) GetChannelUsage(ctx context.Context, days int) ([]ChannelUsageStats, error) {
	if days <= 0 {
		days = 30
	}

	startTime := time.Now().AddDate(0, 0, -days)
	return s.repo.GetChannelUsage(ctx, &startTime, nil)
}

// GetDailyUsage retrieves daily usage trend
func (s *Service) GetDailyUsage(ctx context.Context, days int) ([]DailyUsageStats, error) {
	if days <= 0 {
		days = 30
	}

	return s.repo.GetDailyUsage(ctx, days)
}

// GetOverview returns dashboard overview data
func (s *Service) GetOverview(ctx context.Context) (*AnalyticsOverview, error) {
	overview := &AnalyticsOverview{}

	// Total stats
	stats, err := s.repo.GetStats(ctx, nil)
	if err != nil {
		slog.Warn("Failed to get total stats", slog.Any("error", err))
	} else {
		overview.TotalRequests = stats.TotalRequests
		overview.TotalTokens = stats.TotalTokens
		overview.AvgLatencyMS = stats.AvgLatencyMS
		overview.SuccessRate = stats.SuccessRate
	}

	// Today's stats
	today := time.Now().Truncate(24 * time.Hour)
	todayStats, err := s.repo.GetStats(ctx, &RequestLogFilter{StartTime: &today})
	if err != nil {
		slog.Warn("Failed to get today's stats", slog.Any("error", err))
	} else {
		overview.TodayRequests = todayStats.TotalRequests
		overview.TodayTokens = todayStats.TotalTokens
	}

	// Top models
	models, err := s.GetModelUsage(ctx, 7)
	if err != nil {
		slog.Warn("Failed to get top models", slog.Any("error", err))
	} else {
		overview.TopModels = models
		overview.ActiveModels = len(models)
	}

	// Top channels
	channels, err := s.GetChannelUsage(ctx, 7)
	if err != nil {
		slog.Warn("Failed to get top channels", slog.Any("error", err))
	} else {
		overview.TopChannels = channels
		overview.ActiveChannels = len(channels)
	}

	// Daily trend
	daily, err := s.GetDailyUsage(ctx, 14)
	if err != nil {
		slog.Warn("Failed to get daily trend", slog.Any("error", err))
	} else {
		overview.DailyTrend = daily
	}

	return overview, nil
}

// RequestContext holds context data for logging
type RequestContext struct {
	ChannelID    *int64
	Model        string
	TokenGroupID *int64
	UserID       *int64
	APIKeyID     string
	InputTokens  int64
	OutputTokens int64
	LatencyMS    int64
	Status       string
	ErrorMessage string
	Provider     string
	RequestID    string
	IPAddress    string
	UserAgent    string
	ModelRaw     string
}
