package service

import (
	"context"
	"errors"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
)

// GatewayService coordinates high-level business validation rules
type GatewayService struct {
	keyRepo   repository.KeyRepository
	usageRepo repository.UsageRepository
}

func NewGatewayService(kr repository.KeyRepository, ur repository.UsageRepository) *GatewayService {
	return &GatewayService{
		keyRepo:   kr,
		usageRepo: ur,
	}
}

// ValidateRequest checks if an API key is active and within active quota limits
func (s *GatewayService) ValidateRequest(ctx context.Context, apiKeyHash string) (*model.RegisteredKey, error) {
	key, err := s.keyRepo.GetByHash(ctx, apiKeyHash)
	if err != nil {
		return nil, err
	}
	if !key.Active {
		return nil, errors.New("API key is inactive")
	}

	// Quota compliance check
	dailyCount, err := s.usageRepo.GetDailyUsage(ctx, key.ID)
	if err == nil && dailyCount >= key.DailyQuota {
		return nil, errors.New("daily request limit exceeded")
	}

	return key, nil
}
