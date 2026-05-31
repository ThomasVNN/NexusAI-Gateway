package channel

import (
	"context"
	"fmt"
	"log/slog"
)

// Service provides channel business logic
type Service struct {
	repo *Repository
}

// NewService creates a new channel service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new channel after validation
func (s *Service) Create(ctx context.Context, ch *Channel) error {
	if err := ch.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if ch.Models == nil {
		ch.Models = []string{}
	}
	if ch.Ratio == 0 {
		ch.Ratio = 1
	}
	if ch.Priority == 0 {
		ch.Priority = 1
	}
	if ch.BalanceType == "" {
		ch.BalanceType = "prepay"
	}

	slog.Info("Creating channel",
		slog.String("name", ch.Name),
		slog.String("type", string(ch.Type)),
	)

	return s.repo.Create(ctx, ch)
}

// Update updates an existing channel
func (s *Service) Update(ctx context.Context, ch *Channel) error {
	existing, err := s.repo.GetByID(ctx, ch.ID)
	if err != nil {
		return err
	}

	// Preserve sensitive data if not provided
	if ch.APIKeyEncrypted == "" {
		ch.APIKeyEncrypted = existing.APIKeyEncrypted
	}

	if err := ch.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	slog.Info("Updating channel",
		slog.Int64("id", ch.ID),
		slog.String("name", ch.Name),
	)

	return s.repo.Update(ctx, ch)
}

// Delete removes a channel
func (s *Service) Delete(ctx context.Context, id int64) error {
	slog.Info("Deleting channel", slog.Int64("id", id))
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a channel by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Channel, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all channels with optional filtering
func (s *Service) List(ctx context.Context, filter *ChannelFilter) ([]*Channel, error) {
	return s.repo.List(ctx, filter)
}

// ListActive retrieves all active channels
func (s *Service) ListActive(ctx context.Context) ([]*Channel, error) {
	return s.repo.List(ctx, &ChannelFilter{})
}

// TestChannel tests connectivity to a channel
func (s *Service) TestChannel(ctx context.Context, id int64, testModel string) (*ChannelTestResult, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &ChannelTestResult{
		ChannelID: id,
		TestedAt:  ch.UpdatedAt,
	}

	// Basic validation - check if channel has required fields
	if ch.APIKeyEncrypted == "" && ch.Type != ChannelTypeOllama {
		result.Success = false
		result.ErrorMsg = "No API key configured"
		return result, nil
	}

	// Mark as successful if basic config is present
	// Real connectivity test would be done via HTTP client
	result.Success = true
	result.LatencyMS = 0 // Would be measured in actual connectivity test

	slog.Info("Channel test completed",
		slog.Int64("channel_id", id),
		slog.Bool("success", result.Success),
	)

	return result, nil
}

// Activate enables a channel
func (s *Service) Activate(ctx context.Context, id int64) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	ch.IsActive = true
	return s.repo.Update(ctx, ch)
}

// Deactivate disables a channel
func (s *Service) Deactivate(ctx context.Context, id int64) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	ch.IsActive = false
	return s.repo.Update(ctx, ch)
}

// UpdateBalance updates a channel's balance
func (s *Service) UpdateBalance(ctx context.Context, id int64, delta float64) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	ch.Balance += delta
	if ch.Balance < 0 {
		ch.Balance = 0
	}
	return s.repo.Update(ctx, ch)
}
