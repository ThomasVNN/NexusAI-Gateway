package provider

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Repository handles provider persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new provider repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new provider into the database
func (r *Repository) Create(ctx context.Context, p *Provider) error {
	query := `
		INSERT INTO providers (id, name, type, endpoint, credentials, status, priority, enabled, health_check_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRowContext(ctx, query,
		p.ID, p.Name, p.Type, p.Endpoint, p.Credentials,
		p.Status, p.Priority, p.Enabled, p.HealthCheckURL, now, now,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	slog.Debug("Created provider", slog.String("id", p.ID), slog.String("name", p.Name))
	return nil
}

// Update modifies an existing provider
func (r *Repository) Update(ctx context.Context, p *Provider) error {
	query := `
		UPDATE providers SET
			name = $1, type = $2, endpoint = $3, credentials = $4,
			status = $5, priority = $6, enabled = $7, health_check_url = $8, updated_at = $9
		WHERE id = $10`

	p.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		p.Name, p.Type, p.Endpoint, p.Credentials,
		p.Status, p.Priority, p.Enabled, p.HealthCheckURL, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrProviderNotFound
	}

	slog.Debug("Updated provider", slog.String("id", p.ID))
	return nil
}

// Delete removes a provider from the database
func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM providers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrProviderNotFound
	}

	slog.Debug("Deleted provider", slog.String("id", id))
	return nil
}

// GetByID retrieves a provider by its ID
func (r *Repository) GetByID(ctx context.Context, id string) (*Provider, error) {
	query := `
		SELECT id, name, type, endpoint, credentials, status, priority, enabled, health_check_url, created_at, updated_at
		FROM providers WHERE id = $1`

	p := &Provider{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Type, &p.Endpoint, &p.Credentials,
		&p.Status, &p.Priority, &p.Enabled, &p.HealthCheckURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return p, nil
}

// List retrieves all providers with optional filtering
func (r *Repository) List(ctx context.Context, filter *ProviderFilter) ([]*Provider, error) {
	query := `
		SELECT id, name, type, endpoint, credentials, status, priority, enabled, health_check_url, created_at, updated_at
		FROM providers WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.Type != "" {
			query += fmt.Sprintf(" AND type = $%d", argIdx)
			args = append(args, filter.Type)
			argIdx++
		}
		if filter.Enabled != nil {
			query += fmt.Sprintf(" AND enabled = $%d", argIdx)
			args = append(args, *filter.Enabled)
			argIdx++
		}
		if filter.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, filter.Status)
			argIdx++
		}
	}

	query += " ORDER BY priority ASC, name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Type, &p.Endpoint, &p.Credentials,
			&p.Status, &p.Priority, &p.Enabled, &p.HealthCheckURL, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// ListEnabled retrieves all enabled providers ordered by priority
func (r *Repository) ListEnabled(ctx context.Context) ([]*Provider, error) {
	query := `
		SELECT id, name, type, endpoint, credentials, status, priority, enabled, health_check_url, created_at, updated_at
		FROM providers
		WHERE enabled = true AND status = 'active'
		ORDER BY priority ASC, name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Type, &p.Endpoint, &p.Credentials,
			&p.Status, &p.Priority, &p.Enabled, &p.HealthCheckURL, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// GetHealth retrieves health metrics for a provider
func (r *Repository) GetHealth(ctx context.Context, providerID string) (*ProviderHealth, error) {
	query := `
		SELECT provider_id, last_health_check, error_count, latency_p99, success_rate
		FROM provider_health WHERE provider_id = $1`

	h := &ProviderHealth{}
	var lastCheck sql.NullTime
	var latencyP99, successRate sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, providerID).Scan(
		&h.ProviderID, &lastCheck, &h.ErrorCount, &latencyP99, &successRate,
	)
	if err == sql.ErrNoRows {
		// Return default health if not found
		return &ProviderHealth{
			ProviderID: providerID,
			IsHealthy:  true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider health: %w", err)
	}

	if lastCheck.Valid {
		h.LastHealthCheck = lastCheck.Time
	}
	if latencyP99.Valid {
		h.LatencyP99 = latencyP99.Float64
	}
	if successRate.Valid {
		h.SuccessRate = successRate.Float64
	}
	h.IsHealthy = h.ErrorCount < 5 && h.SuccessRate >= 80.0

	return h, nil
}

// UpdateHealth updates health metrics for a provider
func (r *Repository) UpdateHealth(ctx context.Context, h *ProviderHealth) error {
	query := `
		INSERT INTO provider_health (provider_id, last_health_check, error_count, latency_p99, success_rate)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider_id) DO UPDATE SET
			last_health_check = EXCLUDED.last_health_check,
			error_count = EXCLUDED.error_count,
			latency_p99 = EXCLUDED.latency_p99,
			success_rate = EXCLUDED.success_rate`

	_, err := r.db.ExecContext(ctx, query,
		h.ProviderID, h.LastHealthCheck, h.ErrorCount, h.LatencyP99, h.SuccessRate,
	)
	if err != nil {
		return fmt.Errorf("failed to update provider health: %w", err)
	}

	return nil
}

// IncrementErrorCount increments the error count for a provider
func (r *Repository) IncrementErrorCount(ctx context.Context, providerID string) error {
	query := `
		INSERT INTO provider_health (provider_id, error_count, last_health_check)
		VALUES ($1, 1, $2)
		ON CONFLICT (provider_id) DO UPDATE SET
			error_count = provider_health.error_count + 1,
			last_health_check = $2`

	_, err := r.db.ExecContext(ctx, query, providerID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment error count: %w", err)
	}

	return nil
}

// ResetErrorCount resets the error count for a provider (after successful request)
func (r *Repository) ResetErrorCount(ctx context.Context, providerID string) error {
	query := `
		UPDATE provider_health SET error_count = 0 WHERE provider_id = $1`

	_, err := r.db.ExecContext(ctx, query, providerID)
	if err != nil {
		return fmt.Errorf("failed to reset error count: %w", err)
	}

	return nil
}

// GetCredentials retrieves the credentials for a provider (for internal use)
func (r *Repository) GetCredentials(ctx context.Context, id string) (string, error) {
	query := `SELECT credentials FROM providers WHERE id = $1`
	var credentials string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&credentials)
	if err == sql.ErrNoRows {
		return "", ErrProviderNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get credentials: %w", err)
	}
	return credentials, nil
}
