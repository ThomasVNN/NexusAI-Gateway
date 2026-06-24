package tracking

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

// CommandEntry represents a tracked command
type CommandEntry struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspace_id"`
	UserID           string    `json:"user_id"`
	Command          string    `json:"command"`
	CommandType      string    `json:"command_type"`
	OriginalSize     int       `json:"original_size"`
	OptimizedSize    int       `json:"optimized_size"`
	Savings          int       `json:"savings"`
	ExecutionTimeMs  int64     `json:"execution_time_ms"`
	Success          bool      `json:"success"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
	SyncedToPostgres bool      `json:"synced_to_postgres"`
	PostgresID       string    `json:"postgres_id,omitempty"`
}

// TrackingStore provides command tracking operations
type TrackingStore struct {
	db            *sql.DB
	retentionDays int
	syncEnabled   bool
	pgDB          *postgres.DB
}

// NewSQLiteStore creates a new SQLite tracking store
func NewSQLiteStore(dbPath string, retentionDays int) (*TrackingStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_sync=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &TrackingStore{
		db:            db,
		retentionDays: retentionDays,
		syncEnabled:   false,
		pgDB:          nil,
	}, nil
}

// NewSQLiteStoreWithPostgres creates a new SQLite tracking store with PostgreSQL sync
func NewSQLiteStoreWithPostgres(dbPath string, retentionDays int, pgDB *postgres.DB) (*TrackingStore, error) {
	store, err := NewSQLiteStore(dbPath, retentionDays)
	if err != nil {
		return nil, err
	}
	store.syncEnabled = pgDB != nil
	store.pgDB = pgDB
	return store, nil
}

// initSchema creates the necessary tables
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS command_tracking (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		command TEXT NOT NULL,
		command_type TEXT NOT NULL,
		original_size INTEGER NOT NULL DEFAULT 0,
		optimized_size INTEGER NOT NULL DEFAULT 0,
		savings INTEGER NOT NULL DEFAULT 0,
		execution_time_ms INTEGER NOT NULL DEFAULT 0,
		success INTEGER NOT NULL DEFAULT 1,
		error_message TEXT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		synced_to_postgres INTEGER NOT NULL DEFAULT 0,
		postgres_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_command_tracking_workspace ON command_tracking(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_user ON command_tracking(user_id);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_type ON command_tracking(command_type);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_timestamp ON command_tracking(timestamp);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_synced ON command_tracking(synced_to_postgres);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_workspace_time ON command_tracking(workspace_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_command_tracking_user_time ON command_tracking(user_id, timestamp);
	`

	_, err := db.Exec(schema)
	return err
}

// TrackCommand records a command execution
func (s *TrackingStore) TrackCommand(ctx context.Context, entry *CommandEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	query := `
	INSERT INTO command_tracking (
		id, workspace_id, user_id, command, command_type,
		original_size, optimized_size, savings, execution_time_ms,
		success, error_message, timestamp, synced_to_postgres
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`

	success := 0
	if entry.Success {
		success = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		entry.ID, entry.WorkspaceID, entry.UserID, entry.Command, entry.CommandType,
		entry.OriginalSize, entry.OptimizedSize, entry.Savings, entry.ExecutionTimeMs,
		success, entry.ErrorMessage, entry.Timestamp,
	)
	return err
}

// CommandFilter contains filtering options
type CommandFilter struct {
	WorkspaceID string
	UserID     string
	CommandType string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
	Offset     int
}

// GetCommands retrieves commands with filtering
func (s *TrackingStore) GetCommands(ctx context.Context, filter CommandFilter) ([]CommandEntry, error) {
	query := `SELECT id, workspace_id, user_id, command, command_type,
		original_size, optimized_size, savings, execution_time_ms,
		success, error_message, timestamp, synced_to_postgres, postgres_id
		FROM command_tracking WHERE 1=1`
	args := []interface{}{}

	if filter.WorkspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, filter.WorkspaceID)
	}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.CommandType != "" {
		query += " AND command_type = ?"
		args = append(args, filter.CommandType)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 100" // Default limit
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommandEntry
	for rows.Next() {
		var entry CommandEntry
		var success int
		var errorMsg sql.NullString
		var postgresID sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.WorkspaceID, &entry.UserID, &entry.Command, &entry.CommandType,
			&entry.OriginalSize, &entry.OptimizedSize, &entry.Savings, &entry.ExecutionTimeMs,
			&success, &errorMsg, &entry.Timestamp, &entry.SyncedToPostgres, &postgresID,
		)
		if err != nil {
			return nil, err
		}

		entry.Success = success == 1
		if errorMsg.Valid {
			entry.ErrorMessage = errorMsg.String
		}
		entry.SyncedToPostgres = entry.SyncedToPostgres
		if postgresID.Valid {
			entry.PostgresID = postgresID.String
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Stats contains tracking statistics
type Stats struct {
	TotalCommands   int               `json:"total_commands"`
	TotalSavings    int               `json:"total_savings"`
	AvgSavings      float64           `json:"avg_savings"`
	SuccessRate     float64           `json:"success_rate"`
	TopCommandTypes []CommandTypeStats `json:"top_command_types"`
}

// CommandTypeStats contains stats per command type
type CommandTypeStats struct {
	Type         string `json:"type"`
	Count        int    `json:"count"`
	TotalSavings int    `json:"total_savings"`
}

// GetStats returns statistics for a workspace
func (s *TrackingStore) GetStats(ctx context.Context, workspaceID string, since time.Time) (*Stats, error) {
	stats := &Stats{
		TopCommandTypes: []CommandTypeStats{},
	}

	// Get total counts
	totalQuery := `
	SELECT 
		COUNT(*) as total,
		COALESCE(SUM(savings), 0) as total_savings,
		COALESCE(AVG(savings), 0) as avg_savings,
		COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as successes,
		COUNT(*) as total_count
	FROM command_tracking
	WHERE workspace_id = ? AND timestamp >= ?
	`
	var avgSavings sql.NullFloat64
	err := s.db.QueryRowContext(ctx, totalQuery, workspaceID, since).Scan(
		&stats.TotalCommands, &stats.TotalSavings, &avgSavings, &stats.SuccessRate, &stats.TotalCommands,
	)
	if err != nil {
		return nil, err
	}
	if avgSavings.Valid {
		stats.AvgSavings = avgSavings.Float64
	}
	if stats.TotalCommands > 0 {
		stats.SuccessRate = stats.SuccessRate / float64(stats.TotalCommands) * 100
	}

	// Get top command types
	typesQuery := `
	SELECT command_type, COUNT(*) as count, COALESCE(SUM(savings), 0) as total_savings
	FROM command_tracking
	WHERE workspace_id = ? AND timestamp >= ?
	GROUP BY command_type
	ORDER BY count DESC
	LIMIT 5
	`
	rows, err := s.db.QueryContext(ctx, typesQuery, workspaceID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var typeStats CommandTypeStats
		if err := rows.Scan(&typeStats.Type, &typeStats.Count, &typeStats.TotalSavings); err != nil {
			return nil, err
		}
		stats.TopCommandTypes = append(stats.TopCommandTypes, typeStats)
	}

	return stats, rows.Err()
}

// Cleanup removes old records
func (s *TrackingStore) Cleanup(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -s.retentionDays)
	query := `DELETE FROM command_tracking WHERE timestamp < ?`
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// SyncToPostgres syncs data to PostgreSQL
func (s *TrackingStore) SyncToPostgres(ctx context.Context) error {
	if !s.syncEnabled || s.pgDB == nil {
		return nil
	}

	// Get unsynced entries
	query := `
	SELECT id, workspace_id, user_id, command, command_type,
		original_size, optimized_size, savings, execution_time_ms,
		success, error_message, timestamp
	FROM command_tracking
	WHERE synced_to_postgres = 0
	ORDER BY timestamp ASC
	LIMIT 1000
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query unsynced entries: %w", err)
	}
	defer rows.Close()

	// Ensure the PostgreSQL table exists
	if err := s.ensurePostgresTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure postgres table: %w", err)
	}

	syncedCount := 0
	for rows.Next() {
		var entry CommandEntry
		var success int
		var errorMsg sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.WorkspaceID, &entry.UserID, &entry.Command, &entry.CommandType,
			&entry.OriginalSize, &entry.OptimizedSize, &entry.Savings, &entry.ExecutionTimeMs,
			&success, &errorMsg, &entry.Timestamp,
		)
		if err != nil {
			continue
		}

		// Insert into PostgreSQL
		pgID := uuid.New().String()
		pgQuery := `
		INSERT INTO rtk_command_tracking (
			id, workspace_id, user_id, command, command_type,
			original_size, optimized_size, savings, execution_time_ms,
			success, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
		`

		var pgErrorMsg *string
		if errorMsg.Valid {
			pgErrorMsg = &errorMsg.String
		}

		_, err = s.pgDB.ExecContext(ctx, pgQuery,
			pgID, entry.WorkspaceID, entry.UserID, entry.Command, entry.CommandType,
			entry.OriginalSize, entry.OptimizedSize, entry.Savings, entry.ExecutionTimeMs,
			success == 1, pgErrorMsg, entry.Timestamp,
		)
		if err != nil {
			continue
		}

		// Mark as synced in SQLite
		updateQuery := `UPDATE command_tracking SET synced_to_postgres = 1, postgres_id = ? WHERE id = ?`
		_, _ = s.db.ExecContext(ctx, updateQuery, pgID, entry.ID)
		syncedCount++
	}

	return rows.Err()
}

// ensurePostgresTable creates the PostgreSQL tracking table if it doesn't exist
func (s *TrackingStore) ensurePostgresTable(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS rtk_command_tracking (
		id VARCHAR(255) PRIMARY KEY,
		workspace_id VARCHAR(255) NOT NULL,
		user_id VARCHAR(255) NOT NULL,
		command TEXT NOT NULL,
		command_type VARCHAR(100) NOT NULL,
		original_size INTEGER NOT NULL DEFAULT 0,
		optimized_size INTEGER NOT NULL DEFAULT 0,
		savings INTEGER NOT NULL DEFAULT 0,
		execution_time_ms BIGINT NOT NULL DEFAULT 0,
		success BOOLEAN NOT NULL DEFAULT true,
		error_message TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_rtk_tracking_workspace ON rtk_command_tracking(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_rtk_tracking_user ON rtk_command_tracking(user_id);
	CREATE INDEX IF NOT EXISTS idx_rtk_tracking_timestamp ON rtk_command_tracking(created_at);
	`

	_, err := s.pgDB.ExecContext(ctx, schema)
	return err
}

// Export exports tracking data as JSON
func (s *TrackingStore) Export(ctx context.Context, filter CommandFilter) ([]byte, error) {
	entries, err := s.GetCommands(ctx, filter)
	if err != nil {
		return nil, err
	}

	export := struct {
		ExportedAt time.Time      `json:"exported_at"`
		Count      int            `json:"count"`
		Filter     CommandFilter  `json:"filter"`
		Entries    []CommandEntry `json:"entries"`
	}{
		ExportedAt: time.Now(),
		Count:      len(entries),
		Filter:     filter,
		Entries:    entries,
	}

	return json.MarshalIndent(export, "", "  ")
}

// Close closes the database connection
func (s *TrackingStore) Close() error {
	return s.db.Close()
}

// HealthCheck returns database health status
func (s *TrackingStore) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// SetRetentionDays updates the retention period
func (s *TrackingStore) SetRetentionDays(days int) {
	s.retentionDays = days
}

// EnablePostgresSync enables PostgreSQL synchronization
func (s *TrackingStore) EnablePostgresSync(pgDB *postgres.DB) {
	s.pgDB = pgDB
	s.syncEnabled = pgDB != nil
}
