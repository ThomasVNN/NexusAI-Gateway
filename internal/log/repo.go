package log

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles request log persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new request log repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new request log entry
func (r *Repository) Create(ctx context.Context, log *RequestLog) error {
	query := `
		INSERT INTO request_logs (channel_id, model, token_group_id, user_id, api_key_id,
		                         input_tokens, output_tokens, total_tokens, latency_ms,
		                         status, error_message, provider, request_id, ip_address,
		                         user_agent, model_raw, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`

	log.TotalTokens = log.InputTokens + log.OutputTokens
	log.CreatedAt = time.Now()

	err := r.db.QueryRowContext(ctx, query,
		log.ChannelID, log.Model, log.TokenGroupID, log.UserID, log.APIKeyID,
		log.InputTokens, log.OutputTokens, log.TotalTokens, log.LatencyMS,
		log.Status, log.ErrorMessage, log.Provider, log.RequestID, log.IPAddress,
		log.UserAgent, log.ModelRaw, log.CreatedAt,
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to create request log: %w", err)
	}

	return nil
}

// GetByID retrieves a request log by ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*RequestLog, error) {
	query := `
		SELECT id, channel_id, model, token_group_id, user_id, api_key_id,
		       input_tokens, output_tokens, total_tokens, latency_ms, status,
		       error_message, provider, request_id, ip_address, user_agent,
		       model_raw, created_at
		FROM request_logs WHERE id = $1`

	log := &RequestLog{}
	var channelID, tokenGroupID, userID sql.NullInt64
	var apiKeyID, errorMsg, provider, requestID, ipAddress, userAgent, modelRaw sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID, &channelID, &log.Model, &tokenGroupID, &userID, &apiKeyID,
		&log.InputTokens, &log.OutputTokens, &log.TotalTokens, &log.LatencyMS,
		&log.Status, &errorMsg, &provider, &requestID, &ipAddress,
		&userAgent, &modelRaw, &log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrModelRequired
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get request log: %w", err)
	}

	if channelID.Valid {
		log.ChannelID = &channelID.Int64
	}
	if tokenGroupID.Valid {
		log.TokenGroupID = &tokenGroupID.Int64
	}
	if userID.Valid {
		log.UserID = &userID.Int64
	}
	if apiKeyID.Valid {
		log.APIKeyID = apiKeyID.String
	}
	if errorMsg.Valid {
		log.ErrorMessage = errorMsg.String
	}
	if provider.Valid {
		log.Provider = provider.String
	}
	if requestID.Valid {
		log.RequestID = requestID.String
	}
	if ipAddress.Valid {
		log.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		log.UserAgent = userAgent.String
	}
	if modelRaw.Valid {
		log.ModelRaw = modelRaw.String
	}

	return log, nil
}

// List retrieves request logs with filtering and pagination
func (r *Repository) List(ctx context.Context, filter *RequestLogFilter) ([]*RequestLog, error) {
	query := `
		SELECT id, channel_id, model, token_group_id, user_id, api_key_id,
		       input_tokens, output_tokens, total_tokens, latency_ms, status,
		       error_message, provider, request_id, ip_address, user_agent,
		       model_raw, created_at
		FROM request_logs WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.ChannelID != nil {
			query += fmt.Sprintf(" AND channel_id = $%d", argIdx)
			args = append(args, *filter.ChannelID)
			argIdx++
		}
		if filter.TokenGroupID != nil {
			query += fmt.Sprintf(" AND token_group_id = $%d", argIdx)
			args = append(args, *filter.TokenGroupID)
			argIdx++
		}
		if filter.UserID != nil {
			query += fmt.Sprintf(" AND user_id = $%d", argIdx)
			args = append(args, *filter.UserID)
			argIdx++
		}
		if filter.Model != "" {
			query += fmt.Sprintf(" AND model = $%d", argIdx)
			args = append(args, filter.Model)
			argIdx++
		}
		if filter.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, filter.Status)
			argIdx++
		}
		if filter.StartTime != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, *filter.StartTime)
			argIdx++
		}
		if filter.EndTime != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, *filter.EndTime)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIdx)
			args = append(args, filter.Limit)
			argIdx++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIdx)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list request logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetStats retrieves aggregated statistics
func (r *Repository) GetStats(ctx context.Context, filter *RequestLogFilter) (*RequestLogStats, error) {
	query := `
		SELECT COUNT(*),
		       SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status != 'success' THEN 1 ELSE 0 END),
		       COALESCE(SUM(input_tokens), 0),
		       COALESCE(SUM(output_tokens), 0),
		       COALESCE(AVG(latency_ms), 0)
		FROM request_logs WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.ChannelID != nil {
			query += fmt.Sprintf(" AND channel_id = $%d", argIdx)
			args = append(args, *filter.ChannelID)
			argIdx++
		}
		if filter.TokenGroupID != nil {
			query += fmt.Sprintf(" AND token_group_id = $%d", argIdx)
			args = append(args, *filter.TokenGroupID)
			argIdx++
		}
		if filter.UserID != nil {
			query += fmt.Sprintf(" AND user_id = $%d", argIdx)
			args = append(args, *filter.UserID)
			argIdx++
		}
		if filter.Model != "" {
			query += fmt.Sprintf(" AND model = $%d", argIdx)
			args = append(args, filter.Model)
			argIdx++
		}
		if filter.StartTime != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, *filter.StartTime)
			argIdx++
		}
		if filter.EndTime != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, *filter.EndTime)
		}
	}

	stats := &RequestLogStats{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalRequests, &stats.SuccessfulRequests, &stats.FailedRequests,
		&stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.AvgLatencyMS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
	}

	return stats, nil
}

// GetModelUsage retrieves usage statistics per model
func (r *Repository) GetModelUsage(ctx context.Context, startTime, endTime *time.Time, limit int) ([]ModelUsageStats, error) {
	query := `
		SELECT model,
		       COUNT(*) as request_count,
		       COALESCE(SUM(input_tokens), 0) as input_tokens,
		       COALESCE(SUM(output_tokens), 0) as output_tokens,
		       COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_logs
		WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if startTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startTime)
		argIdx++
	}
	if endTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endTime)
		argIdx++
	}

	query += fmt.Sprintf(" GROUP BY model ORDER BY request_count DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	defer rows.Close()

	var stats []ModelUsageStats
	for rows.Next() {
		var s ModelUsageStats
		if err := rows.Scan(&s.Model, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.AvgLatencyMS); err != nil {
			return nil, fmt.Errorf("failed to scan model stats: %w", err)
		}
		s.TotalTokens = s.InputTokens + s.OutputTokens
		stats = append(stats, s)
	}

	return stats, nil
}

// GetChannelUsage retrieves usage statistics per channel
func (r *Repository) GetChannelUsage(ctx context.Context, startTime, endTime *time.Time) ([]ChannelUsageStats, error) {
	query := `
		SELECT channel_id,
		       COUNT(*) as request_count,
		       COALESCE(SUM(input_tokens), 0) as input_tokens,
		       COALESCE(SUM(output_tokens), 0) as output_tokens,
		       COALESCE(AVG(latency_ms), 0) as avg_latency,
		       CASE WHEN COUNT(*) > 0 
		            THEN (SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) * 100.0 / COUNT(*))
		            ELSE 0 
		       END as success_rate
		FROM request_logs
		WHERE channel_id IS NOT NULL`

	args := []interface{}{}
	argIdx := 1

	if startTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startTime)
		argIdx++
	}
	if endTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endTime)
	}

	query += " GROUP BY channel_id ORDER BY request_count DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel usage: %w", err)
	}
	defer rows.Close()

	var stats []ChannelUsageStats
	for rows.Next() {
		var s ChannelUsageStats
		if err := rows.Scan(&s.ChannelID, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.AvgLatencyMS, &s.SuccessRate); err != nil {
			return nil, fmt.Errorf("failed to scan channel stats: %w", err)
		}
		s.TotalTokens = s.InputTokens + s.OutputTokens
		stats = append(stats, s)
	}

	return stats, nil
}

// GetDailyUsage retrieves daily usage aggregation
func (r *Repository) GetDailyUsage(ctx context.Context, days int) ([]DailyUsageStats, error) {
	query := `
		SELECT DATE(created_at) as date,
		       COUNT(*) as request_count,
		       COALESCE(SUM(input_tokens), 0) as input_tokens,
		       COALESCE(SUM(output_tokens), 0) as output_tokens,
		       COUNT(DISTINCT user_id) as unique_users,
		       COUNT(DISTINCT model) as unique_models
		FROM request_logs
		WHERE created_at >= CURRENT_DATE - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC`

	query = fmt.Sprintf(query, days)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}
	defer rows.Close()

	var stats []DailyUsageStats
	for rows.Next() {
		var s DailyUsageStats
		var date time.Time
		if err := rows.Scan(&date, &s.RequestCount, &s.InputTokens, &s.OutputTokens, &s.UniqueUsers, &s.UniqueModels); err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		s.Date = date.Format("2006-01-02")
		s.TotalTokens = s.InputTokens + s.OutputTokens
		stats = append(stats, s)
	}

	return stats, nil
}

// scanLogs scans rows into RequestLog slice
func (r *Repository) scanLogs(rows *sql.Rows) ([]*RequestLog, error) {
	var logs []*RequestLog

	for rows.Next() {
		log := &RequestLog{}
		var channelID, tokenGroupID, userID sql.NullInt64
		var apiKeyID, errorMsg, provider, requestID, ipAddress, userAgent, modelRaw sql.NullString

		if err := rows.Scan(
			&log.ID, &channelID, &log.Model, &tokenGroupID, &userID, &apiKeyID,
			&log.InputTokens, &log.OutputTokens, &log.TotalTokens, &log.LatencyMS,
			&log.Status, &errorMsg, &provider, &requestID, &ipAddress,
			&userAgent, &modelRaw, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if channelID.Valid {
			log.ChannelID = &channelID.Int64
		}
		if tokenGroupID.Valid {
			log.TokenGroupID = &tokenGroupID.Int64
		}
		if userID.Valid {
			log.UserID = &userID.Int64
		}
		if apiKeyID.Valid {
			log.APIKeyID = apiKeyID.String
		}
		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}
		if provider.Valid {
			log.Provider = provider.String
		}
		if requestID.Valid {
			log.RequestID = requestID.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = userAgent.String
		}
		if modelRaw.Valid {
			log.ModelRaw = modelRaw.String
		}

		logs = append(logs, log)
	}

	return logs, nil
}
