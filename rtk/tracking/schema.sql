-- SQLite schema for RTK command tracking
-- Retention: 90 days by default

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

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_command_tracking_workspace ON command_tracking(workspace_id);
CREATE INDEX IF NOT EXISTS idx_command_tracking_user ON command_tracking(user_id);
CREATE INDEX IF NOT EXISTS idx_command_tracking_type ON command_tracking(command_type);
CREATE INDEX IF NOT EXISTS idx_command_tracking_timestamp ON command_tracking(timestamp);
CREATE INDEX IF NOT EXISTS idx_command_tracking_synced ON command_tracking(synced_to_postgres);

-- Composite indexes for filtered queries
CREATE INDEX IF NOT EXISTS idx_command_tracking_workspace_time ON command_tracking(workspace_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_command_tracking_user_time ON command_tracking(user_id, timestamp);

-- Cleanup: older than 90 days
-- Can be executed with: DELETE FROM command_tracking WHERE timestamp < datetime('now', '-90 days');
