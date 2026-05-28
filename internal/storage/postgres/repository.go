package postgres

import (
	"context"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

type KeyRepository struct {
	db *postgres.DB
}

func NewKeyRepository(db *postgres.DB) *KeyRepository {
	return &KeyRepository{db: db}
}

func (r *KeyRepository) GetByID(ctx context.Context, id string) (*model.RegisteredKey, error) {
	query := `SELECT id, key_hash, name, source_app, daily_quota, hourly_quota, active, created_at, updated_at 
	          FROM registered_keys WHERE id = $1`
	var key model.RegisteredKey
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.SourceApp,
		&key.DailyQuota, &key.HourlyQuota, &key.Active,
		&key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *KeyRepository) GetByHash(ctx context.Context, hash string) (*model.RegisteredKey, error) {
	query := `SELECT id, key_hash, name, source_app, daily_quota, hourly_quota, active, created_at, updated_at 
	          FROM registered_keys WHERE key_hash = $1`
	var key model.RegisteredKey
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.SourceApp,
		&key.DailyQuota, &key.HourlyQuota, &key.Active,
		&key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *KeyRepository) Save(ctx context.Context, key *model.RegisteredKey) error {
	query := `INSERT INTO registered_keys (id, key_hash, name, source_app, daily_quota, hourly_quota, active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	          ON CONFLICT (id) DO UPDATE 
	          SET key_hash = EXCLUDED.key_hash, name = EXCLUDED.name, source_app = EXCLUDED.source_app, 
	              daily_quota = EXCLUDED.daily_quota, hourly_quota = EXCLUDED.hourly_quota, 
	              active = EXCLUDED.active, updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query,
		key.ID, key.KeyHash, key.Name, key.SourceApp,
		key.DailyQuota, key.HourlyQuota, key.Active,
		key.CreatedAt, key.UpdatedAt,
	)
	return err
}

func (r *KeyRepository) ListAll(ctx context.Context) ([]*model.RegisteredKey, error) {
	query := `SELECT id, key_hash, name, source_app, daily_quota, hourly_quota, active, created_at, updated_at 
	          FROM registered_keys ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.RegisteredKey
	for rows.Next() {
		var key model.RegisteredKey
		err := rows.Scan(
			&key.ID, &key.KeyHash, &key.Name, &key.SourceApp,
			&key.DailyQuota, &key.HourlyQuota, &key.Active,
			&key.CreatedAt, &key.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	return keys, nil
}

type UsageRepository struct {
	db *postgres.DB
}

func NewUsageRepository(db *postgres.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) LogUsage(ctx context.Context, record *model.UsageRecord) error {
	query := `INSERT INTO usage_records (key_id, model_id, prompt_tokens, completion_tokens, latency_ms, source_app, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)`
	_, err := r.db.ExecContext(ctx, query,
		record.KeyID, record.ModelID, record.PromptTokens,
		record.CompletionTokens, record.LatencyMS, record.SourceApp,
	)
	return err
}

func (r *UsageRepository) GetHourlyUsage(ctx context.Context, keyID string) (int, error) {
	query := `SELECT COUNT(*) FROM usage_records 
	          WHERE key_id = $1 AND created_at >= $2`
	var count int
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	err := r.db.QueryRowContext(ctx, query, keyID, oneHourAgo).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageRepository) GetDailyUsage(ctx context.Context, keyID string) (int, error) {
	query := `SELECT COUNT(*) FROM usage_records 
	          WHERE key_id = $1 AND created_at >= $2`
	var count int
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := r.db.QueryRowContext(ctx, query, keyID, oneDayAgo).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageRepository) GetAggregateUsage(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT COUNT(*), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(AVG(latency_ms), 0)
	          FROM usage_records`
	var totalCalls int
	var totalPromptTokens int
	var totalCompletionTokens int
	var avgLatency float64

	err := r.db.QueryRowContext(ctx, query).Scan(&totalCalls, &totalPromptTokens, &totalCompletionTokens, &avgLatency)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_calls":             totalCalls,
		"total_prompt_tokens":     totalPromptTokens,
		"total_completion_tokens": totalCompletionTokens,
		"average_latency_ms":      avgLatency,
	}, nil
}

func (r *UsageRepository) ListLogs(ctx context.Context) ([]*model.UsageRecord, error) {
	query := `SELECT id, key_id, model_id, prompt_tokens, completion_tokens, latency_ms, source_app, created_at 
	          FROM usage_records ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.UsageRecord
	for rows.Next() {
		var record model.UsageRecord
		err := rows.Scan(
			&record.ID, &record.KeyID, &record.ModelID, &record.PromptTokens,
			&record.CompletionTokens, &record.LatencyMS, &record.SourceApp, &record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &record)
	}
	return logs, nil
}

type ProviderRepository struct {
	db *postgres.DB
}

func NewProviderRepository(db *postgres.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) ListAll(ctx context.Context) ([]*model.ProviderConnection, error) {
	query := `SELECT id, provider, name, api_key, endpoint, is_active, priority, created_at, updated_at 
	          FROM provider_connections ORDER BY priority ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.ProviderConnection
	for rows.Next() {
		var p model.ProviderConnection
		err := rows.Scan(&p.ID, &p.Provider, &p.Name, &p.APIKey, &p.Endpoint, &p.IsActive, &p.Priority, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

func (r *ProviderRepository) Save(ctx context.Context, conn *model.ProviderConnection) error {
	query := `INSERT INTO provider_connections (id, provider, name, api_key, endpoint, is_active, priority, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	          ON CONFLICT (id) DO UPDATE 
	          SET provider = EXCLUDED.provider, name = EXCLUDED.name, api_key = EXCLUDED.api_key, 
	              endpoint = EXCLUDED.endpoint, is_active = EXCLUDED.is_active, priority = EXCLUDED.priority, 
	              updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, conn.ID, conn.Provider, conn.Name, conn.APIKey, conn.Endpoint, conn.IsActive, conn.Priority)
	return err
}

func (r *ProviderRepository) GetByID(ctx context.Context, id string) (*model.ProviderConnection, error) {
	query := `SELECT id, provider, name, api_key, endpoint, is_active, priority, created_at, updated_at 
	          FROM provider_connections WHERE id = $1`
	var p model.ProviderConnection
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Provider, &p.Name, &p.APIKey, &p.Endpoint, &p.IsActive, &p.Priority, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
