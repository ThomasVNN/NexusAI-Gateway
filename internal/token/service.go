package token

import (
	"context"
	"fmt"
	"log/slog"
)

// Service provides token group business logic
type Service struct {
	repo *Repository
}

// NewService creates a new token group service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new token group
func (s *Service) Create(ctx context.Context, tg *TokenGroup) error {
	if err := tg.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if tg.AllowedModels == nil {
		tg.AllowedModels = []string{}
	}
	if !tg.IsActive {
		tg.IsActive = true
	}
	if tg.Priority == 0 {
		tg.Priority = 1
	}

	slog.Info("Creating token group",
		slog.String("name", tg.Name),
		slog.Int64("daily_quota", tg.DailyQuota),
	)

	return s.repo.Create(ctx, tg)
}

// Update updates an existing token group
func (s *Service) Update(ctx context.Context, tg *TokenGroup) error {
	if err := tg.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	slog.Info("Updating token group",
		slog.Int64("id", tg.ID),
		slog.String("name", tg.Name),
	)

	return s.repo.Update(ctx, tg)
}

// Delete removes a token group
func (s *Service) Delete(ctx context.Context, id int64) error {
	slog.Info("Deleting token group", slog.Int64("id", id))
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a token group by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*TokenGroup, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all token groups
func (s *Service) List(ctx context.Context) ([]*TokenGroup, error) {
	return s.repo.List(ctx)
}

// GetStats retrieves statistics for a token group
func (s *Service) GetStats(ctx context.Context, id int64) (*TokenGroupStats, error) {
	return s.repo.GetStats(ctx, id)
}

// CheckQuota checks if a token group can consume tokens
func (s *Service) CheckQuota(ctx context.Context, id int64, tokens int64) (bool, error) {
	tg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	if !tg.CanConsume(tokens) {
		slog.Warn("Quota exceeded",
			slog.Int64("group_id", id),
			slog.Int64("requested_tokens", tokens),
			slog.Int64("used_today", tg.UsedToday),
			slog.Int64("daily_quota", tg.DailyQuota),
		)
		return false, ErrQuotaExceeded
	}

	return true, nil
}

// ConsumeTokens increments usage for a token group
func (s *Service) ConsumeTokens(ctx context.Context, id int64, inputTokens, outputTokens int64) error {
	tg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	totalTokens := inputTokens + outputTokens

	if !tg.CanConsume(totalTokens) {
		return ErrQuotaExceeded
	}

	slog.Debug("Consuming tokens",
		slog.Int64("group_id", id),
		slog.Int64("input_tokens", inputTokens),
		slog.Int64("output_tokens", outputTokens),
	)

	return s.repo.IncrementUsage(ctx, id, inputTokens, outputTokens)
}

// ResetUsage resets usage counters for a token group
func (s *Service) ResetUsage(ctx context.Context, id int64, period string) error {
	switch period {
	case "hourly":
		return s.repo.ResetHourlyUsage(ctx, id)
	case "daily":
		return s.repo.ResetDailyUsage(ctx, id)
	case "monthly":
		return s.repo.ResetMonthlyUsage(ctx, id)
	default:
		return fmt.Errorf("invalid period: %s", period)
	}
}

// Activate enables a token group
func (s *Service) Activate(ctx context.Context, id int64) error {
	tg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	tg.IsActive = true
	return s.repo.Update(ctx, tg)
}

// Deactivate disables a token group
func (s *Service) Deactivate(ctx context.Context, id int64) error {
	tg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	tg.IsActive = false
	return s.repo.Update(ctx, tg)
}

// AssignModels sets the allowed models for a token group
func (s *Service) AssignModels(ctx context.Context, id int64, models []string) error {
	tg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	tg.AllowedModels = models
	return s.repo.Update(ctx, tg)
}
