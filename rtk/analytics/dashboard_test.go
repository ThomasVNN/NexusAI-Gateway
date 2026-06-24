package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use a shared in-memory connection with a specific URI to ensure state is visible
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&mode=memory")
	require.NoError(t, err)

	// Verify the connection works
	err = db.Ping()
	require.NoError(t, err)

	// Create table
	schema := `
	CREATE TABLE IF NOT EXISTS command_records (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		workspace_id TEXT NOT NULL,
		command TEXT NOT NULL,
		command_type TEXT NOT NULL,
		original_tokens INTEGER NOT NULL,
		optimized_tokens INTEGER NOT NULL,
		savings INTEGER NOT NULL,
		execution_time_ms INTEGER NOT NULL,
		success INTEGER NOT NULL DEFAULT 1,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON command_records(timestamp);
	CREATE INDEX IF NOT EXISTS idx_user_id ON command_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_command_type ON command_records(command_type);`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	assert.NotNil(t, service)
	assert.Equal(t, 90, service.retentionDays)
}

func TestRecordCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	record := &CommandRecord{
		ID:              "test-id-1",
		UserID:          "user-1",
		WorkspaceID:     "workspace-1",
		Command:         "git status",
		CommandType:     "git",
		OriginalTokens:  1000,
		OptimizedTokens: 600,
		Savings:         400,
		ExecutionTime:   100 * time.Millisecond,
		Success:         true,
		Timestamp:       time.Now(),
	}

	err := service.RecordCommand(ctx, record)
	require.NoError(t, err)

	// Verify record was inserted
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM command_records WHERE id = ?", record.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRecordCommandValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	tests := []struct {
		name    string
		record  *CommandRecord
		wantErr bool
	}{
		{
			name: "missing ID",
			record: &CommandRecord{
				UserID:      "user-1",
				WorkspaceID: "workspace-1",
				Command:     "git status",
			},
			wantErr: true,
		},
		{
			name: "missing UserID",
			record: &CommandRecord{
				ID:          "test-id",
				WorkspaceID: "workspace-1",
				Command:     "git status",
			},
			wantErr: true,
		},
		{
			name: "missing WorkspaceID",
			record: &CommandRecord{
				ID:      "test-id",
				UserID:  "user-1",
				Command: "git status",
			},
			wantErr: true,
		},
		{
			name: "missing Command",
			record: &CommandRecord{
				ID:          "test-id",
				UserID:      "user-1",
				WorkspaceID: "workspace-1",
			},
			wantErr: true,
		},
		{
			name: "valid record with auto-timestamp",
			record: &CommandRecord{
				ID:          "test-id-2",
				UserID:      "user-1",
				WorkspaceID: "workspace-1",
				Command:     "cargo build",
				CommandType: "cargo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RecordCommand(ctx, tt.record)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDashboardStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data
	records := []*CommandRecord{
		{
			ID:              "rec-1",
			UserID:          "user-1",
			WorkspaceID:     "workspace-1",
			Command:         "git status",
			CommandType:     "git",
			OriginalTokens:  1000,
			OptimizedTokens: 600,
			Savings:         400,
			ExecutionTime:   100 * time.Millisecond,
			Success:         true,
			Timestamp:       time.Now(),
		},
		{
			ID:              "rec-2",
			UserID:          "user-1",
			WorkspaceID:     "workspace-1",
			Command:         "git diff",
			CommandType:     "git",
			OriginalTokens:  2000,
			OptimizedTokens: 1200,
			Savings:         800,
			ExecutionTime:   150 * time.Millisecond,
			Success:         true,
			Timestamp:       time.Now(),
		},
		{
			ID:              "rec-3",
			UserID:          "user-2",
			WorkspaceID:     "workspace-1",
			Command:         "npm install",
			CommandType:     "npm",
			OriginalTokens:  3000,
			OptimizedTokens: 2000,
			Savings:         1000,
			ExecutionTime:   500 * time.Millisecond,
			Success:         true,
			Timestamp:       time.Now(),
		},
	}

	for _, rec := range records {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	// Test GetDashboardStats
	stats, err := service.GetDashboardStats(ctx, "7d")
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, "7d", stats.Period)

	// Verify overall savings
	assert.Equal(t, 6000, stats.OverallSavings.TotalOriginal)
	assert.Equal(t, 3800, stats.OverallSavings.TotalOptimized)
	assert.Equal(t, 2200, stats.OverallSavings.TotalSavings)
	assert.InDelta(t, 36.67, stats.OverallSavings.SavingsPercent, 0.1)

	// Verify command breakdown
	assert.Len(t, stats.CommandBreakdown, 2)

	// Verify user breakdown
	assert.Len(t, stats.UserBreakdown, 2)
}

func TestGetSavingsByCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data
	testRecords := []*CommandRecord{
		{ID: "r1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "r2", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 200, OptimizedTokens: 100, Savings: 100, ExecutionTime: 20 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "r3", UserID: "u2", WorkspaceID: "w1", Command: "cargo build", CommandType: "cargo", OriginalTokens: 500, OptimizedTokens: 250, Savings: 250, ExecutionTime: 100 * time.Millisecond, Success: true, Timestamp: time.Now()},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	since := time.Now().Add(-24 * time.Hour)
	results, err := service.GetSavingsByCommand(ctx, since)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Check git has highest savings (140)
	foundGit := false
	for _, r := range results {
		if r.CommandType == "git" {
			foundGit = true
			assert.Equal(t, 2, r.Count)
			assert.Equal(t, 140, r.TotalSavings)
		}
	}
	assert.True(t, foundGit, "git command should be in results")
}

func TestGetSavingsByUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data
	testRecords := []*CommandRecord{
		{ID: "r1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "r2", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 200, OptimizedTokens: 100, Savings: 100, ExecutionTime: 20 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "r3", UserID: "u2", WorkspaceID: "w1", Command: "cargo build", CommandType: "cargo", OriginalTokens: 500, OptimizedTokens: 250, Savings: 250, ExecutionTime: 100 * time.Millisecond, Success: true, Timestamp: time.Now()},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	since := time.Now().Add(-24 * time.Hour)
	results, err := service.GetSavingsByUser(ctx, since)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Check u1 has highest savings (140)
	foundU1 := false
	for _, r := range results {
		if r.UserID == "u1" {
			foundU1 = true
			assert.Equal(t, 2, r.CommandCount)
			assert.Equal(t, 140, r.TotalSavings)
		}
	}
	assert.True(t, foundU1, "user u1 should be in results")
}

func TestGetTrends(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data with timestamps spanning multiple hours
	testRecords := []*CommandRecord{
		{ID: "t1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "t2", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 200, OptimizedTokens: 100, Savings: 100, ExecutionTime: 20 * time.Millisecond, Success: true, Timestamp: time.Now().Add(-1 * time.Hour)},
		{ID: "t3", UserID: "u2", WorkspaceID: "w1", Command: "cargo build", CommandType: "cargo", OriginalTokens: 500, OptimizedTokens: 250, Savings: 250, ExecutionTime: 100 * time.Millisecond, Success: true, Timestamp: time.Now().Add(-2 * time.Hour)},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	// Test trends for 24h period (hourly buckets)
	trends, err := service.GetTrends(ctx, "24h")
	require.NoError(t, err)
	assert.NotEmpty(t, trends)
}

func TestCleanupOldRecords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert old and new records
	oldTime := time.Now().Add(-100 * 24 * time.Hour) // 100 days ago
	newTime := time.Now()

	testRecords := []*CommandRecord{
		{ID: "old1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: oldTime},
		{ID: "new1", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 200, OptimizedTokens: 100, Savings: 100, ExecutionTime: 20 * time.Millisecond, Success: true, Timestamp: newTime},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	// Cleanup with 90-day retention
	deleted, err := service.CleanupOldRecords(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only new record remains
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM command_records").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTopCommands(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data with varying command counts
	testRecords := []*CommandRecord{
		{ID: "c1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "c2", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "c3", UserID: "u1", WorkspaceID: "w1", Command: "git log", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "c4", UserID: "u2", WorkspaceID: "w1", Command: "cargo build", CommandType: "cargo", OriginalTokens: 500, OptimizedTokens: 250, Savings: 250, ExecutionTime: 100 * time.Millisecond, Success: true, Timestamp: time.Now()},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	top, err := service.TopCommands(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, top, 2)

	// git should be first (3 occurrences)
	assert.Equal(t, "git", top[0].CommandType)
	assert.Equal(t, 3, top[0].Count)
}

func TestExportToCSV(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := New(db, 90)
	ctx := context.Background()

	// Insert test data
	testRecords := []*CommandRecord{
		{ID: "csv1", UserID: "u1", WorkspaceID: "w1", Command: "git status", CommandType: "git", OriginalTokens: 100, OptimizedTokens: 60, Savings: 40, ExecutionTime: 10 * time.Millisecond, Success: true, Timestamp: time.Now()},
		{ID: "csv2", UserID: "u1", WorkspaceID: "w1", Command: "git diff", CommandType: "git", OriginalTokens: 200, OptimizedTokens: 100, Savings: 100, ExecutionTime: 20 * time.Millisecond, Success: true, Timestamp: time.Now()},
	}

	for _, rec := range testRecords {
		err := service.RecordCommand(ctx, rec)
		require.NoError(t, err)
	}

	csvData, err := service.ExportToCSV(ctx, "7d")
	require.NoError(t, err)
	assert.NotEmpty(t, csvData)

	// Verify CSV contains expected content
	csvStr := string(csvData)
	assert.Contains(t, csvStr, "ID,UserID,WorkspaceID")
	assert.Contains(t, csvStr, "csv1")
	assert.Contains(t, csvStr, "csv2")
	assert.Contains(t, csvStr, "git status")
	assert.Contains(t, csvStr, "git diff")
	assert.Contains(t, csvStr, "true")
}

func TestCalculateSinceTime(t *testing.T) {
	tests := []struct {
		period string
	}{
		{"24h"},
		{"7d"},
		{"30d"},
		{"90d"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			since := calculateSinceTime(tt.period)
			assert.False(t, since.IsZero())
			assert.True(t, since.Before(time.Now()))
		})
	}
}
