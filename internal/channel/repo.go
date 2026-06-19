package channel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Repository handles channel persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new channel repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new channel into the database
func (r *Repository) Create(ctx context.Context, ch *Channel) error {
	modelsJSON, err := json.Marshal(ch.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		INSERT INTO channels (name, type, base_url, api_key_encrypted, models, priority, ratio, is_active, balance, balance_type, group_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err = r.db.QueryRowContext(ctx, query,
		ch.Name, ch.Type, ch.BaseURL, ch.APIKeyEncrypted, modelsJSON,
		ch.Priority, ch.Ratio, ch.IsActive, ch.Balance, ch.BalanceType,
		ch.GroupName, now, now,
	).Scan(&ch.ID, &ch.CreatedAt, &ch.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	slog.Debug("Created channel", slog.Int64("id", ch.ID), slog.String("name", ch.Name))
	return nil
}

// Update modifies an existing channel
func (r *Repository) Update(ctx context.Context, ch *Channel) error {
	modelsJSON, err := json.Marshal(ch.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		UPDATE channels SET
			name = $1, type = $2, base_url = $3, api_key_encrypted = $4,
			models = $5, priority = $6, ratio = $7, is_active = $8,
			balance = $9, balance_type = $10, group_name = $11, updated_at = $12
		WHERE id = $13`

	ch.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		ch.Name, ch.Type, ch.BaseURL, ch.APIKeyEncrypted, modelsJSON,
		ch.Priority, ch.Ratio, ch.IsActive, ch.Balance, ch.BalanceType,
		ch.GroupName, ch.UpdatedAt, ch.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrChannelNotFound
	}

	slog.Debug("Updated channel", slog.Int64("id", ch.ID))
	return nil
}

// Delete removes a channel from the database
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM channels WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrChannelNotFound
	}

	slog.Debug("Deleted channel", slog.Int64("id", id))
	return nil
}

// GetByID retrieves a channel by its ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*Channel, error) {
	query := `
		SELECT id, name, type, base_url, api_key_encrypted, models, priority, ratio,
		       is_active, balance, balance_type, group_name, created_at, updated_at
		FROM channels WHERE id = $1`

	ch := &Channel{}
	var modelsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ch.ID, &ch.Name, &ch.Type, &ch.BaseURL, &ch.APIKeyEncrypted,
		&modelsJSON, &ch.Priority, &ch.Ratio, &ch.IsActive, &ch.Balance,
		&ch.BalanceType, &ch.GroupName, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrChannelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	if err := json.Unmarshal(modelsJSON, &ch.Models); err != nil {
		ch.Models = []string{}
	}

	return ch, nil
}

// List retrieves all channels with optional filtering
func (r *Repository) List(ctx context.Context, filter *ChannelFilter) ([]*Channel, error) {
	query := `
		SELECT id, name, type, base_url, api_key_encrypted, models, priority, ratio,
		       is_active, balance, balance_type, group_name, created_at, updated_at
		FROM channels WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.Type != "" {
			query += fmt.Sprintf(" AND type = $%d", argIdx)
			args = append(args, filter.Type)
			argIdx++
		}
		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIdx)
			args = append(args, *filter.IsActive)
			argIdx++
		}
		if filter.GroupName != "" {
			query += fmt.Sprintf(" AND group_name = $%d", argIdx)
			args = append(args, filter.GroupName)
			argIdx++
		}
	}

	query += " ORDER BY priority ASC, id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		ch := &Channel{}
		var modelsJSON []byte
		if err := rows.Scan(
			&ch.ID, &ch.Name, &ch.Type, &ch.BaseURL, &ch.APIKeyEncrypted,
			&modelsJSON, &ch.Priority, &ch.Ratio, &ch.IsActive, &ch.Balance,
			&ch.BalanceType, &ch.GroupName, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &ch.Models); err != nil {
			ch.Models = []string{}
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

// ListActive retrieves all active channels that support a given model
func (r *Repository) ListActive(ctx context.Context, model string) ([]*Channel, error) {
	query := `
		SELECT id, name, type, base_url, api_key_encrypted, models, priority, ratio,
		       is_active, balance, balance_type, group_name, created_at, updated_at
		FROM channels
		WHERE is_active = true
		AND (models = '[]' OR models::jsonb ? $1 OR models::jsonb ? '*')
		ORDER BY priority ASC, ratio DESC`

	rows, err := r.db.QueryContext(ctx, query, model)
	if err != nil {
		return nil, fmt.Errorf("failed to list active channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		ch := &Channel{}
		var modelsJSON []byte
		if err := rows.Scan(
			&ch.ID, &ch.Name, &ch.Type, &ch.BaseURL, &ch.APIKeyEncrypted,
			&modelsJSON, &ch.Priority, &ch.Ratio, &ch.IsActive, &ch.Balance,
			&ch.BalanceType, &ch.GroupName, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &ch.Models); err != nil {
			ch.Models = []string{}
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

// GetAPIKey retrieves the decrypted API key for a channel
func (r *Repository) GetAPIKey(ctx context.Context, id int64) (string, error) {
	query := `SELECT api_key_encrypted FROM channels WHERE id = $1`
	var apiKey string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&apiKey)
	if err == sql.ErrNoRows {
		return "", ErrChannelNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}
	return apiKey, nil
}

// ChannelFilter provides filtering options for channel queries
type ChannelFilter struct {
	Type      ChannelType
	IsActive  *bool
	GroupName string
}
