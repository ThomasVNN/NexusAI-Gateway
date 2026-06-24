package tracking

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*TrackingStore, func()) {
	tmpFile, err := os.CreateTemp("", "tracking-test-*.db")
	require.NoError(t, err)
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name(), 90)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func TestNewSQLiteStore(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	assert.NotNil(t, store)
	assert.NoError(t, store.HealthCheck(context.Background()))
}

func TestTrackCommand(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entry := &CommandEntry{
		WorkspaceID:     "workspace-1",
		UserID:          "user-1",
		Command:         "ls -la /tmp",
		CommandType:     "shell",
		OriginalSize:    1000,
		OptimizedSize:   800,
		Savings:         200,
		ExecutionTimeMs: 150,
		Success:         true,
		Timestamp:       time.Now(),
	}

	err := store.TrackCommand(context.Background(), entry)
	assert.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
}

func TestGetCommands(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	for i := 0; i < 5; i++ {
		entry := &CommandEntry{
			WorkspaceID:     "workspace-1",
			UserID:          "user-1",
			Command:         "test command",
			CommandType:     "shell",
			OriginalSize:    1000 + i*100,
			OptimizedSize:   800 + i*100,
			Savings:         200,
			ExecutionTimeMs: 100,
			Success:         true,
			Timestamp:       time.Now(),
		}
		err := store.TrackCommand(context.Background(), entry)
		require.NoError(t, err)
	}

	// Test retrieval with filter
	filter := CommandFilter{
		WorkspaceID: "workspace-1",
		Limit:       10,
	}

	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 5)
}

func TestGetCommandsByUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert data for different users
	users := []string{"user-1", "user-2"}
	for _, user := range users {
		for i := 0; i < 3; i++ {
			entry := &CommandEntry{
				WorkspaceID:     "workspace-1",
				UserID:          user,
				Command:         "test command",
				CommandType:     "shell",
				OriginalSize:    1000,
				OptimizedSize:   800,
				Savings:         200,
				ExecutionTimeMs: 100,
				Success:         true,
				Timestamp:       time.Now(),
			}
			err := store.TrackCommand(context.Background(), entry)
			require.NoError(t, err)
		}
	}

	// Filter by user
	filter := CommandFilter{
		UserID: "user-1",
		Limit:  10,
	}

	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)

	for _, entry := range entries {
		assert.Equal(t, "user-1", entry.UserID)
	}
}

func TestGetCommandsByTimeRange(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)

	// Insert old entry
	oldEntry := &CommandEntry{
		WorkspaceID:     "workspace-1",
		UserID:          "user-1",
		Command:         "old command",
		CommandType:     "shell",
		OriginalSize:    1000,
		OptimizedSize:   800,
		Savings:         200,
		ExecutionTimeMs: 100,
		Success:         true,
		Timestamp:       pastTime,
	}
	err := store.TrackCommand(context.Background(), oldEntry)
	require.NoError(t, err)

	// Insert recent entry
	recentEntry := &CommandEntry{
		WorkspaceID:     "workspace-1",
		UserID:          "user-1",
		Command:         "recent command",
		CommandType:     "shell",
		OriginalSize:    1000,
		OptimizedSize:   800,
		Savings:         200,
		ExecutionTimeMs: 100,
		Success:         true,
		Timestamp:       now,
	}
	err = store.TrackCommand(context.Background(), recentEntry)
	require.NoError(t, err)

	// Filter by time range (last 12 hours)
	filter := CommandFilter{
		StartTime: now.Add(-12 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
		Limit:     10,
	}

	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "recent command", entries[0].Command)
}

func TestGetStats(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	for i := 0; i < 5; i++ {
		success := i < 4 // 4 successful, 1 failed
		errorMsg := ""
		if !success {
			errorMsg = "test error"
		}
		entry := &CommandEntry{
			WorkspaceID:     "workspace-1",
			UserID:          "user-1",
			Command:         "test command",
			CommandType:     "shell",
			OriginalSize:    1000,
			OptimizedSize:   800,
			Savings:         200,
			ExecutionTimeMs: 100,
			Success:         success,
			ErrorMessage:    errorMsg,
			Timestamp:       time.Now(),
		}
		err := store.TrackCommand(context.Background(), entry)
		require.NoError(t, err)
	}

	stats, err := store.GetStats(context.Background(), "workspace-1", time.Now().Add(-24*time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, 5, stats.TotalCommands)
	assert.Equal(t, 1000, stats.TotalSavings) // 5 * 200
	assert.InDelta(t, 80.0, stats.SuccessRate, 1.0) // 4/5 = 80%
	assert.Len(t, stats.TopCommandTypes, 1)
	assert.Equal(t, "shell", stats.TopCommandTypes[0].Type)
	assert.Equal(t, 5, stats.TopCommandTypes[0].Count)
}

func TestCleanup(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert old entry
	oldEntry := &CommandEntry{
		WorkspaceID:     "workspace-1",
		UserID:          "user-1",
		Command:         "old command",
		CommandType:     "shell",
		OriginalSize:    1000,
		OptimizedSize:   800,
		Savings:         200,
		ExecutionTimeMs: 100,
		Success:         true,
		Timestamp:       time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
	}
	err := store.TrackCommand(context.Background(), oldEntry)
	require.NoError(t, err)

	// Insert recent entry
	recentEntry := &CommandEntry{
		WorkspaceID:     "workspace-1",
		UserID:          "user-1",
		Command:         "recent command",
		CommandType:     "shell",
		OriginalSize:    1000,
		OptimizedSize:   800,
		Savings:         200,
		ExecutionTimeMs: 100,
		Success:         true,
		Timestamp:       time.Now(),
	}
	err = store.TrackCommand(context.Background(), recentEntry)
	require.NoError(t, err)

	// Set retention to 90 days
	store.SetRetentionDays(90)

	// Cleanup should remove entries older than 90 days
	deleted, err := store.Cleanup(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only recent entry remains
	filter := CommandFilter{Limit: 10}
	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "recent command", entries[0].Command)
}

func TestExport(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	for i := 0; i < 3; i++ {
		entry := &CommandEntry{
			WorkspaceID:     "workspace-1",
			UserID:          "user-1",
			Command:         "test command",
			CommandType:     "shell",
			OriginalSize:    1000,
			OptimizedSize:   800,
			Savings:         200,
			ExecutionTimeMs: 100,
			Success:         true,
			Timestamp:       time.Now(),
		}
		err := store.TrackCommand(context.Background(), entry)
		require.NoError(t, err)
	}

	filter := CommandFilter{
		WorkspaceID: "workspace-1",
		Limit:       10,
	}

	data, err := store.Export(context.Background(), filter)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), `"count": 3`)
	assert.Contains(t, string(data), `"entries"`)
}

func TestClose(t *testing.T) {
	store, cleanup := setupTestDB(t)

	// Close should work
	err := store.Close()
	assert.NoError(t, err)

	// Run cleanup after close
	cleanup()
}

func TestCommandFilterByCommandType(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert data with different command types
	types := []string{"shell", "api", "script"}
	for _, cmdType := range types {
		for i := 0; i < 2; i++ {
			entry := &CommandEntry{
				WorkspaceID:     "workspace-1",
				UserID:          "user-1",
				Command:         "test command",
				CommandType:     cmdType,
				OriginalSize:    1000,
				OptimizedSize:   800,
				Savings:         200,
				ExecutionTimeMs: 100,
				Success:         true,
				Timestamp:       time.Now(),
			}
			err := store.TrackCommand(context.Background(), entry)
			require.NoError(t, err)
		}
	}

	// Filter by command type
	filter := CommandFilter{
		CommandType: "api",
		Limit:       10,
	}

	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	for _, entry := range entries {
		assert.Equal(t, "api", entry.CommandType)
	}
}

func TestPagination(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert 10 entries
	for i := 0; i < 10; i++ {
		entry := &CommandEntry{
			WorkspaceID:     "workspace-1",
			UserID:          "user-1",
			Command:         "test command",
			CommandType:     "shell",
			OriginalSize:    1000,
			OptimizedSize:   800,
			Savings:         200,
			ExecutionTimeMs: 100,
			Success:         true,
			Timestamp:       time.Now(),
		}
		err := store.TrackCommand(context.Background(), entry)
		require.NoError(t, err)
	}

	// Test first page
	filter := CommandFilter{
		Limit:  5,
		Offset: 0,
	}
	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 5)

	// Test second page
	filter.Offset = 5
	entries, err = store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 5)
}

func TestDefaultLimit(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert 150 entries (more than default limit of 100)
	for i := 0; i < 150; i++ {
		entry := &CommandEntry{
			WorkspaceID:     "workspace-1",
			UserID:          "user-1",
			Command:         "test command",
			CommandType:     "shell",
			OriginalSize:    1000,
			OptimizedSize:   800,
			Savings:         200,
			ExecutionTimeMs: 100,
			Success:         true,
			Timestamp:       time.Now(),
		}
		err := store.TrackCommand(context.Background(), entry)
		require.NoError(t, err)
	}

	// Get without explicit limit (should use default 100)
	filter := CommandFilter{}
	entries, err := store.GetCommands(context.Background(), filter)
	assert.NoError(t, err)
	assert.Len(t, entries, 100) // Default limit
}
