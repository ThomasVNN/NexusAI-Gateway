package analytics

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"time"
)

// CommandRecord tracks a single command execution
type CommandRecord struct {
	ID              string
	UserID          string
	WorkspaceID     string
	Command         string
	CommandType     string // git, cargo, npm, docker, etc.
	OriginalTokens  int
	OptimizedTokens int
	Savings         int
	ExecutionTime   time.Duration
	Success         bool
	Timestamp       time.Time
}

// TokenSavings represents cumulative savings
type TokenSavings struct {
	TotalOriginal  int
	TotalOptimized int
	TotalSavings   int
	SavingsPercent float64
}

// SavingsByCommand shows savings broken down by command type
type SavingsByCommand struct {
	CommandType string
	Count       int
	TotalSavings int
	AvgSavings  float64
}

// SavingsByUser shows savings broken down by user
type SavingsByUser struct {
	UserID       string
	TotalSavings int
	CommandCount int
	TopCommands  []string
}

// TimeSeriesPoint represents a data point in time
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
}

// DashboardStats contains all dashboard statistics
type DashboardStats struct {
	OverallSavings   TokenSavings
	CommandBreakdown []SavingsByCommand
	UserBreakdown    []SavingsByUser
	Trends           []TimeSeriesPoint
	Period           string // "7d", "30d", "90d"
}

// AnalyticsService provides analytics operations
type AnalyticsService struct {
	db             *sql.DB
	retentionDays  int
}

// New creates a new AnalyticsService
func New(db *sql.DB, retentionDays int) *AnalyticsService {
	return &AnalyticsService{
		db:            db,
		retentionDays: retentionDays,
	}
}

// RecordCommand records a command execution
func (s *AnalyticsService) RecordCommand(ctx context.Context, record *CommandRecord) error {
	if record.ID == "" {
		return fmt.Errorf("record ID is required")
	}
	if record.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if record.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if record.Command == "" {
		return fmt.Errorf("command is required")
	}

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	query := `
		INSERT INTO command_records (id, user_id, workspace_id, command, command_type,
		                             original_tokens, optimized_tokens, savings, execution_time_ms, success, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := s.db.ExecContext(ctx, query,
		record.ID,
		record.UserID,
		record.WorkspaceID,
		record.Command,
		record.CommandType,
		record.OriginalTokens,
		record.OptimizedTokens,
		record.Savings,
		int(record.ExecutionTime.Milliseconds()),
		record.Success,
		record.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to record command: %w", err)
	}

	return nil
}

// GetDashboardStats returns dashboard statistics
func (s *AnalyticsService) GetDashboardStats(ctx context.Context, period string) (*DashboardStats, error) {
	since := calculateSinceTime(period)

	overall, err := s.getOverallSavings(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall savings: %w", err)
	}

	commandBreakdown, err := s.GetSavingsByCommand(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get command breakdown: %w", err)
	}

	userBreakdown, err := s.GetSavingsByUser(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get user breakdown: %w", err)
	}

	trends, err := s.GetTrends(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}

	return &DashboardStats{
		OverallSavings:   overall,
		CommandBreakdown: commandBreakdown,
		UserBreakdown:    userBreakdown,
		Trends:           trends,
		Period:           period,
	}, nil
}

// GetSavingsByCommand returns savings grouped by command type
func (s *AnalyticsService) GetSavingsByCommand(ctx context.Context, since time.Time) ([]SavingsByCommand, error) {
	query := `
		SELECT command_type, COUNT(*), SUM(savings), COALESCE(AVG(savings), 0)
		FROM command_records
		WHERE timestamp >= $1
		GROUP BY command_type
		ORDER BY SUM(savings) DESC`

	rows, err := s.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query savings by command: %w", err)
	}
	defer rows.Close()

	var results []SavingsByCommand
	for rows.Next() {
		var sbc SavingsByCommand
		if err := rows.Scan(&sbc.CommandType, &sbc.Count, &sbc.TotalSavings, &sbc.AvgSavings); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, sbc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

// GetSavingsByUser returns savings grouped by user
func (s *AnalyticsService) GetSavingsByUser(ctx context.Context, since time.Time) ([]SavingsByUser, error) {
	query := `
		SELECT user_id, SUM(savings), COUNT(*), ARRAY_AGG(DISTINCT command_type)
		FROM command_records
		WHERE timestamp >= $1
		GROUP BY user_id
		ORDER BY SUM(savings) DESC`

	rows, err := s.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query savings by user: %w", err)
	}
	defer rows.Close()

	var results []SavingsByUser
	for rows.Next() {
		var sbu SavingsByUser
		var topCmds []string
		if err := rows.Scan(&sbu.UserID, &sbu.TotalSavings, &sbu.CommandCount, (*[]string)(&topCmds)); err != nil {
			// Handle case where ARRAY_AGG returns NULL
			if err2 := rows.Scan(&sbu.UserID, &sbu.TotalSavings, &sbu.CommandCount, nil); err2 != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}
			sbu.TopCommands = []string{}
		} else {
			sbu.TopCommands = topCmds
		}
		results = append(results, sbu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

// GetTrends returns savings trends over time
func (s *AnalyticsService) GetTrends(ctx context.Context, period string) ([]TimeSeriesPoint, error) {
	since := calculateSinceTime(period)

	// Determine bucket interval based on period
	bucketInterval := "day"
	switch period {
	case "24h":
		bucketInterval = "hour"
	case "7d":
		bucketInterval = "day"
	case "30d":
		bucketInterval = "day"
	case "90d":
		bucketInterval = "week"
	}

	query := fmt.Sprintf(`
		SELECT date_trunc('%s', timestamp) as bucket, COALESCE(SUM(savings), 0)
		FROM command_records
		WHERE timestamp >= $1
		GROUP BY bucket
		ORDER BY bucket`, bucketInterval)

	rows, err := s.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query trends: %w", err)
	}
	defer rows.Close()

	var trends []TimeSeriesPoint
	for rows.Next() {
		var tsp TimeSeriesPoint
		if err := rows.Scan(&tsp.Timestamp, &tsp.Value); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		trends = append(trends, tsp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return trends, nil
}

// CleanupOldRecords removes records older than retention period
func (s *AnalyticsService) CleanupOldRecords(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM command_records
		WHERE timestamp < now() - ($1 || ' days')::INTERVAL`

	result, err := s.db.ExecContext(ctx, query, s.retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old records: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return deleted, nil
}

// TopCommands returns most frequently optimized commands
func (s *AnalyticsService) TopCommands(ctx context.Context, limit int) ([]SavingsByCommand, error) {
	query := `
		SELECT command_type, COUNT(*), SUM(savings), COALESCE(AVG(savings), 0)
		FROM command_records
		WHERE savings > 0
		GROUP BY command_type
		ORDER BY COUNT(*) DESC
		LIMIT $1`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top commands: %w", err)
	}
	defer rows.Close()

	var results []SavingsByCommand
	for rows.Next() {
		var sbc SavingsByCommand
		if err := rows.Scan(&sbc.CommandType, &sbc.Count, &sbc.TotalSavings, &sbc.AvgSavings); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, sbc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

// ExportToCSV exports analytics data as CSV
func (s *AnalyticsService) ExportToCSV(ctx context.Context, period string) ([]byte, error) {
	since := calculateSinceTime(period)

	query := `
		SELECT id, user_id, workspace_id, command, command_type,
		       original_tokens, optimized_tokens, savings, execution_time_ms, success, timestamp
		FROM command_records
		WHERE timestamp >= $1
		ORDER BY timestamp DESC`

	rows, err := s.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var buf []byte
	writer := csv.NewWriter(&writerWrapper{buf: &buf})

	// Write header
	header := []string{"ID", "UserID", "WorkspaceID", "Command", "CommandType",
		"OriginalTokens", "OptimizedTokens", "Savings", "ExecutionTimeMs", "Success", "Timestamp"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for rows.Next() {
		var record CommandRecord
		var successStr string
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.WorkspaceID,
			&record.Command, &record.CommandType,
			&record.OriginalTokens, &record.OptimizedTokens,
			&record.Savings, &record.ExecutionTime, &successStr, &record.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.Success = successStr == "true"

		row := []string{
			record.ID,
			record.UserID,
			record.WorkspaceID,
			record.Command,
			record.CommandType,
			fmt.Sprintf("%d", record.OriginalTokens),
			fmt.Sprintf("%d", record.OptimizedTokens),
			fmt.Sprintf("%d", record.Savings),
			fmt.Sprintf("%d", record.ExecutionTime),
			successStr,
			record.Timestamp.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("csv writer error: %w", err)
	}

	return buf, nil
}

// getOverallSavings calculates overall savings for a time period
func (s *AnalyticsService) getOverallSavings(ctx context.Context, since time.Time) (TokenSavings, error) {
	query := `
		SELECT
			COALESCE(SUM(original_tokens), 0),
			COALESCE(SUM(optimized_tokens), 0),
			COALESCE(SUM(savings), 0),
			CASE WHEN SUM(original_tokens) > 0
			     THEN ROUND((SUM(savings)::NUMERIC / SUM(original_tokens)) * 100, 2)
			     ELSE 0
			END
		FROM command_records
		WHERE timestamp >= $1`

	var ts TokenSavings
	err := s.db.QueryRowContext(ctx, query, since).Scan(
		&ts.TotalOriginal, &ts.TotalOptimized, &ts.TotalSavings, &ts.SavingsPercent,
	)
	if err != nil {
		return ts, fmt.Errorf("failed to get overall savings: %w", err)
	}

	return ts, nil
}

// calculateSinceTime converts period string to time.Duration
func calculateSinceTime(period string) time.Time {
	var since time.Duration
	switch period {
	case "24h":
		since = 24 * time.Hour
	case "7d":
		since = 7 * 24 * time.Hour
	case "30d":
		since = 30 * 24 * time.Hour
	case "90d":
		since = 90 * 24 * time.Hour
	default:
		since = 7 * 24 * time.Hour // Default to 7 days
	}
	return time.Now().Add(-since)
}

// writerWrapper wraps a byte slice to implement io.Writer for csv.NewWriter
type writerWrapper struct {
	buf *[]byte
}

func (w *writerWrapper) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
