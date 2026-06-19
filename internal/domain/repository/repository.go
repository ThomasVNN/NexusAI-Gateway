package repository

import (
	"context"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

// Authenticator defines the interface for authentication
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*UserIdentity, error)
}

// UserIdentity represents an authenticated user
type UserIdentity struct {
	ID       string
	TenantID string
	Roles    []string
}

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
	ListLogs(ctx context.Context) ([]*model.UsageRecord, error)
}

// ProviderRepository handles active AI provider credentials and configurations
type ProviderRepository interface {
	ListAll(ctx context.Context) ([]*model.ProviderConnection, error)
	Save(ctx context.Context, conn *model.ProviderConnection) error
	GetByID(ctx context.Context, id string) (*model.ProviderConnection, error)
}
