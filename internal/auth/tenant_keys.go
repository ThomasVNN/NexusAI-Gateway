package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

// TenantKeyRepository provides tenant-scoped API key operations.
type TenantKeyRepository struct {
	db *postgres.DB
}

// NewTenantKeyRepository creates a new TenantKeyRepository.
func NewTenantKeyRepository(db *postgres.DB) *TenantKeyRepository {
	return &TenantKeyRepository{db: db}
}

// GetByHash retrieves an API key by its hash, scoped to a tenant if provided.
func (r *TenantKeyRepository) GetByHash(ctx context.Context, keyHash string, tenantID string) (*model.RegisteredKey, error) {
	if r.db == nil {
		return nil, errors.New("database unavailable")
	}

	var query string
	var args []interface{}

	if tenantID != "" {
		// Tenant-scoped query
		query = `
			SELECT ak.id, ak.key_hash, ak.name, ak.tenant_id, ak.scopes, ak.expires_at, ak.revoked_at, ak.created_at, ak.last_used_at,
			       t.status as tenant_status
			FROM api_keys ak
			JOIN tenants t ON ak.tenant_id = t.id
			WHERE ak.key_hash = $1 AND ak.tenant_id = $2`
		args = []interface{}{keyHash, tenantID}
	} else {
		// Cross-tenant query (admin only)
		query = `
			SELECT ak.id, ak.key_hash, ak.name, ak.tenant_id, ak.scopes, ak.expires_at, ak.revoked_at, ak.created_at, ak.last_used_at,
			       t.status as tenant_status
			FROM api_keys ak
			JOIN tenants t ON ak.tenant_id = t.id
			WHERE ak.key_hash = $1`
		args = []interface{}{keyHash}
	}

	var key model.RegisteredKey
	var tenantStatus string
	var expiresAt, revokedAt, lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.SourceApp,
		&key.Scopes, &expiresAt, &revokedAt, &key.CreatedAt, &lastUsedAt,
		&tenantStatus,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("key not found by hash")
		}
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	// Check tenant status
	if tenantStatus != "active" {
		return nil, errors.New("tenant is not active")
	}

	// Check if revoked
	if revokedAt.Valid {
		return nil, errors.New("API key is revoked")
	}

	// Check expiration
	if expiresAt.Valid && !expiresAt.Time.IsZero() && expiresAt.Time.Before(time.Now()) {
		return nil, errors.New("API key has expired")
	}

	key.Active = !revokedAt.Valid
	key.ExpiresAt = expiresAt
	key.RevokedAt = revokedAt
	key.LastUsedAt = lastUsedAt

	return &key, nil
}

// GetByTenant retrieves all API keys for a tenant.
func (r *TenantKeyRepository) GetByTenant(ctx context.Context, tenantID string) ([]*model.RegisteredKey, error) {
	if r.db == nil {
		return nil, errors.New("database unavailable")
	}

	query := `
		SELECT id, key_hash, name, tenant_id, scopes, expires_at, revoked_at, created_at, last_used_at
		FROM api_keys
		WHERE tenant_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	var keys []*model.RegisteredKey
	for rows.Next() {
		var key model.RegisteredKey
		var expiresAt, revokedAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID, &key.KeyHash, &key.Name, &key.SourceApp,
			&key.Scopes, &expiresAt, &revokedAt, &key.CreatedAt, &lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		key.Active = !revokedAt.Valid
		keys = append(keys, &key)
	}

	return keys, nil
}

// Create creates a new API key for a tenant.
func (r *TenantKeyRepository) Create(ctx context.Context, key *model.RegisteredKey) error {
	if r.db == nil {
		return errors.New("database unavailable")
	}

	query := `
		INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, scopes, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query,
		key.ID, key.SourceApp, key.Name, key.KeyHash, key.KeyPrefix,
		key.Scopes, key.ExpiresAt, key.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	return nil
}

// Revoke marks an API key as revoked.
func (r *TenantKeyRepository) Revoke(ctx context.Context, keyID string) error {
	if r.db == nil {
		return errors.New("database unavailable")
	}

	query := `UPDATE api_keys SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for an API key.
func (r *TenantKeyRepository) UpdateLastUsed(ctx context.Context, keyID string) error {
	if r.db == nil {
		return nil // Skip in sandbox mode
	}

	query := `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, keyID)
	return err
}

// HasScope checks if the API key has a specific scope.
func HasScope(key *model.RegisteredKey, scope string) bool {
	for _, s := range key.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// RequireScope creates an error if the key doesn't have the required scope.
func RequireScope(key *model.RegisteredKey, scope string) error {
	if !HasScope(key, scope) {
		return fmt.Errorf("missing required scope: %s", scope)
	}
	return nil
}
