package token

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository handles token group persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new token group repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new token group into the database
func (r *Repository) Create(ctx context.Context, tg *TokenGroup) error {
	modelsJSON, err := json.Marshal(tg.AllowedModels)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		INSERT INTO token_groups (name, allowed_models, daily_quota, hourly_quota, monthly_quota,
		                          used_today, used_this_hour, used_this_month, is_active, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err = r.db.QueryRowContext(ctx, query,
		tg.Name, modelsJSON, tg.DailyQuota, tg.HourlyQuota, tg.MonthlyQuota,
		tg.UsedToday, tg.UsedThisHour, tg.UsedThisMonth, tg.IsActive, tg.Priority,
		now, now,
	).Scan(&tg.ID, &tg.CreatedAt, &tg.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create token group: %w", err)
	}

	return nil
}

// Update modifies an existing token group
func (r *Repository) Update(ctx context.Context, tg *TokenGroup) error {
	modelsJSON, err := json.Marshal(tg.AllowedModels)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		UPDATE token_groups SET
			name = $1, allowed_models = $2, daily_quota = $3, hourly_quota = $4,
			monthly_quota = $5, used_today = $6, used_this_hour = $7, used_this_month = $8,
			is_active = $9, priority = $10, updated_at = $11
		WHERE id = $12`

	tg.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		tg.Name, modelsJSON, tg.DailyQuota, tg.HourlyQuota, tg.MonthlyQuota,
		tg.UsedToday, tg.UsedThisHour, tg.UsedThisMonth, tg.IsActive, tg.Priority,
		tg.UpdatedAt, tg.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update token group: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenGroupNotFound
	}

	return nil
}

// Delete removes a token group
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM token_groups WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete token group: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTokenGroupNotFound
	}

	return nil
}

// GetByID retrieves a token group by ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*TokenGroup, error) {
	query := `
		SELECT id, name, allowed_models, daily_quota, hourly_quota, monthly_quota,
		       used_today, used_this_hour, used_this_month, is_active, priority, created_at, updated_at
		FROM token_groups WHERE id = $1`

	tg := &TokenGroup{}
	var modelsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tg.ID, &tg.Name, &modelsJSON, &tg.DailyQuota, &tg.HourlyQuota, &tg.MonthlyQuota,
		&tg.UsedToday, &tg.UsedThisHour, &tg.UsedThisMonth, &tg.IsActive, &tg.Priority,
		&tg.CreatedAt, &tg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTokenGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token group: %w", err)
	}

	if err := json.Unmarshal(modelsJSON, &tg.AllowedModels); err != nil {
		tg.AllowedModels = []string{}
	}

	return tg, nil
}

// List retrieves all token groups
func (r *Repository) List(ctx context.Context) ([]*TokenGroup, error) {
	query := `
		SELECT id, name, allowed_models, daily_quota, hourly_quota, monthly_quota,
		       used_today, used_this_hour, used_this_month, is_active, priority, created_at, updated_at
		FROM token_groups ORDER BY priority DESC, name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list token groups: %w", err)
	}
	defer rows.Close()

	var groups []*TokenGroup
	for rows.Next() {
		tg := &TokenGroup{}
		var modelsJSON []byte
		if err := rows.Scan(
			&tg.ID, &tg.Name, &modelsJSON, &tg.DailyQuota, &tg.HourlyQuota, &tg.MonthlyQuota,
			&tg.UsedToday, &tg.UsedThisHour, &tg.UsedThisMonth, &tg.IsActive, &tg.Priority,
			&tg.CreatedAt, &tg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan token group: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &tg.AllowedModels); err != nil {
			tg.AllowedModels = []string{}
		}
		groups = append(groups, tg)
	}

	return groups, nil
}

// IncrementUsage increments usage counters for a token group
func (r *Repository) IncrementUsage(ctx context.Context, id int64, inputTokens, outputTokens int64) error {
	query := `
		UPDATE token_groups SET
			used_today = used_today + $1,
			used_this_hour = used_this_hour + $1,
			used_this_month = used_this_month + $1,
			updated_at = $2
		WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, inputTokens+outputTokens, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	return nil
}

// ResetHourlyUsage resets the hourly usage counter (called by scheduled job)
func (r *Repository) ResetHourlyUsage(ctx context.Context, id int64) error {
	query := `UPDATE token_groups SET used_this_hour = 0, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reset hourly usage: %w", err)
	}
	return nil
}

// ResetDailyUsage resets the daily usage counter (called by scheduled job)
func (r *Repository) ResetDailyUsage(ctx context.Context, id int64) error {
	query := `UPDATE token_groups SET used_today = 0, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reset daily usage: %w", err)
	}
	return nil
}

// ResetMonthlyUsage resets the monthly usage counter (called by scheduled job)
func (r *Repository) ResetMonthlyUsage(ctx context.Context, id int64) error {
	query := `UPDATE token_groups SET used_this_month = 0, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reset monthly usage: %w", err)
	}
	return nil
}

// GetStats retrieves aggregated statistics for a token group
func (r *Repository) GetStats(ctx context.Context, id int64) (*TokenGroupStats, error) {
	tg, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	stats := &TokenGroupStats{
		GroupID:          tg.ID,
		Name:             tg.Name,
		DailyQuota:       tg.DailyQuota,
		DailyUsed:        tg.UsedToday,
		DailyRemaining:   tg.UsedToday,
		HourlyQuota:      tg.HourlyQuota,
		HourlyUsed:       tg.UsedThisHour,
		HourlyRemaining:  tg.UsedThisHour,
		MonthlyQuota:     tg.MonthlyQuota,
		MonthlyUsed:      tg.UsedThisMonth,
		MonthlyRemaining: tg.UsedThisMonth,
	}

	if tg.DailyQuota > 0 {
		stats.DailyRemaining = tg.DailyQuota - tg.UsedToday
		stats.DailyUsagePercent = float64(tg.UsedToday) / float64(tg.DailyQuota) * 100
	}

	if tg.HourlyQuota > 0 {
		stats.HourlyRemaining = tg.HourlyQuota - tg.UsedThisHour
	}

	if tg.MonthlyQuota > 0 {
		stats.MonthlyRemaining = tg.MonthlyQuota - tg.UsedThisMonth
	}

	return stats, nil
}
