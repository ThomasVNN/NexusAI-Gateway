package routing

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

// ModelConfigRepository manages model configurations in the database
type ModelConfigRepository struct {
	db *postgres.DB
}

// NewModelConfigRepository creates a new ModelConfigRepository
func NewModelConfigRepository(db *postgres.DB) *ModelConfigRepository {
	return &ModelConfigRepository{db: db}
}

// ModelConfig represents a model configuration stored in the database
type ModelConfig struct {
	ID              string
	Name            string
	ProviderID      string
	Endpoint        string
	APIKeyRef       string
	CostPer1KInput  float64
	CostPer1KOutput float64
	MaxTokens       int
	Capabilities    []string
	Priority        int
	IsActive        bool
	TenantID        string
	CreatedAt       string
	UpdatedAt       string
}

// GetModelsForTenant retrieves all models available for a tenant
func (r *ModelConfigRepository) GetModelsForTenant(ctx context.Context, tenantID string) ([]*ModelConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `
		SELECT id, name, provider_id, endpoint, cost_per_1k_input, cost_per_1k_output, 
		       max_tokens, capabilities, priority, is_active, tenant_id, created_at, updated_at
		FROM model_configs
		WHERE (tenant_id = $1 OR tenant_id IS NULL) AND is_active = true
		ORDER BY priority ASC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	var configs []*ModelConfig
	for rows.Next() {
		cfg := &ModelConfig{}
		var caps sql.NullString
		var endpoint sql.NullString
		var tenantIDNull sql.NullString

		err := rows.Scan(
			&cfg.ID, &cfg.Name, &cfg.ProviderID, &endpoint,
			&cfg.CostPer1KInput, &cfg.CostPer1KOutput,
			&cfg.MaxTokens, &caps, &cfg.Priority, &cfg.IsActive,
			&tenantIDNull, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model config: %w", err)
		}

		if caps.Valid {
			cfg.Capabilities = parseCapabilities(caps.String)
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// GetModelByName retrieves a specific model configuration
func (r *ModelConfigRepository) GetModelByName(ctx context.Context, tenantID, modelName string) (*ModelConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `
		SELECT id, name, provider_id, endpoint, cost_per_1k_input, cost_per_1k_output, 
		       max_tokens, capabilities, priority, is_active, tenant_id, created_at, updated_at
		FROM model_configs
		WHERE name = $1 AND (tenant_id = $2 OR tenant_id IS NULL) AND is_active = true
		ORDER BY CASE WHEN tenant_id = $2 THEN 0 ELSE 1 END
		LIMIT 1`

	cfg := &ModelConfig{}
	var caps sql.NullString
	var endpoint sql.NullString
	var tenantIDNull sql.NullString

	err := r.db.QueryRowContext(ctx, query, modelName, tenantID).Scan(
		&cfg.ID, &cfg.Name, &cfg.ProviderID, &endpoint,
		&cfg.CostPer1KInput, &cfg.CostPer1KOutput,
		&cfg.MaxTokens, &caps, &cfg.Priority, &cfg.IsActive,
		&tenantIDNull, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found: %s", modelName)
		}
		return nil, fmt.Errorf("failed to query model: %w", err)
	}

	if caps.Valid {
		cfg.Capabilities = parseCapabilities(caps.String)
	}

	return cfg, nil
}

// parseCapabilities parses comma-separated capabilities string
func parseCapabilities(caps string) []string {
	var result []string
	for _, c := range splitAndTrim(caps, ",") {
		result = append(result, c)
	}
	return result
}

// splitAndTrim splits a string and trims whitespace
func splitAndTrim(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// TenantRoutingProfile stores routing preferences for a tenant
type TenantRoutingProfile struct {
	TenantID        string
	DefaultStrategy string
	PreferredModels []string
	BlockedModels   []string
	MaxLatencyMs    int
	MaxCostPerDay   float64
}

// GetTenantProfile retrieves routing preferences for a tenant
func (r *ModelConfigRepository) GetTenantProfile(ctx context.Context, tenantID string) (*TenantRoutingProfile, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `
		SELECT tenant_id, default_strategy, preferred_models, blocked_models, max_latency_ms, max_cost_per_day
		FROM tenant_routing_profiles
		WHERE tenant_id = $1`

	profile := &TenantRoutingProfile{}
	var preferredModels, blockedModels sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&profile.TenantID, &profile.DefaultStrategy,
		&preferredModels, &blockedModels,
		&profile.MaxLatencyMs, &profile.MaxCostPerDay,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default profile
			return &TenantRoutingProfile{
				TenantID:        tenantID,
				DefaultStrategy: "cost_optimized",
				PreferredModels: []string{},
				BlockedModels:   []string{},
				MaxLatencyMs:    5000,
				MaxCostPerDay:   100.0,
			}, nil
		}
		return nil, fmt.Errorf("failed to query profile: %w", err)
	}

	if preferredModels.Valid {
		profile.PreferredModels = splitAndTrim(preferredModels.String, ",")
	}
	if blockedModels.Valid {
		profile.BlockedModels = splitAndTrim(blockedModels.String, ",")
	}

	return profile, nil
}

// SaveTenantProfile saves routing preferences for a tenant
func (r *ModelConfigRepository) SaveTenantProfile(ctx context.Context, profile *TenantRoutingProfile) error {
	if r.db == nil {
		return fmt.Errorf("database unavailable")
	}

	query := `
		INSERT INTO tenant_routing_profiles (tenant_id, default_strategy, preferred_models, blocked_models, max_latency_ms, max_cost_per_day)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id) DO UPDATE SET
			default_strategy = EXCLUDED.default_strategy,
			preferred_models = EXCLUDED.preferred_models,
			blocked_models = EXCLUDED.blocked_models,
			max_latency_ms = EXCLUDED.max_latency_ms,
			max_cost_per_day = EXCLUDED.max_cost_per_day`

	_, err := r.db.ExecContext(ctx, query,
		profile.TenantID, profile.DefaultStrategy,
		joinStrings(profile.PreferredModels, ","),
		joinStrings(profile.BlockedModels, ","),
		profile.MaxLatencyMs, profile.MaxCostPerDay,
	)
	return err
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
