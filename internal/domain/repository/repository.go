package repository

import (
	"context"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

// KeyRepository handles storage interactions for API Keys
type KeyRepository interface {
	GetByID(ctx context.Context, id string) (*model.RegisteredKey, error)
	GetByHash(ctx context.Context, hash string) (*model.RegisteredKey, error)
	Save(ctx context.Context, key *model.RegisteredKey) error
	ListAll(ctx context.Context) ([]*model.RegisteredKey, error)
}

// UsageRepository handles audit logs and usage accounting
type UsageRepository interface {
	LogUsage(ctx context.Context, record *model.UsageRecord) error
	GetHourlyUsage(ctx context.Context, keyID string) (int, error)
	GetDailyUsage(ctx context.Context, keyID string) (int, error)
	GetAggregateUsage(ctx context.Context) (map[string]interface{}, error)
}
