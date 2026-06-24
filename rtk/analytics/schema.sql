-- RTK Analytics Dashboard Schema
-- Token savings tracking with 90-day retention

CREATE TABLE IF NOT EXISTS command_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    command TEXT NOT NULL,
    command_type TEXT NOT NULL, -- git, cargo, npm, docker, etc.
    original_tokens INT NOT NULL,
    optimized_tokens INT NOT NULL,
    savings INT NOT NULL,
    execution_time_ms INT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT true,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_command_records_timestamp ON command_records(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_command_records_user_id ON command_records(user_id);
CREATE INDEX IF NOT EXISTS idx_command_records_workspace_id ON command_records(workspace_id);
CREATE INDEX IF NOT EXISTS idx_command_records_command_type ON command_records(command_type);

-- Composite index for period-based queries
CREATE INDEX IF NOT EXISTS idx_command_records_period ON command_records(timestamp DESC, command_type);

-- Index for cleanup job (old records)
CREATE INDEX IF NOT EXISTS idx_command_records_cleanup ON command_records(timestamp);

-- View for overall savings statistics
CREATE OR REPLACE VIEW savings_overview AS
SELECT
    COUNT(*) as total_commands,
    SUM(original_tokens) as total_original_tokens,
    SUM(optimized_tokens) as total_optimized_tokens,
    SUM(savings) as total_savings,
    CASE WHEN SUM(original_tokens) > 0
         THEN ROUND((SUM(savings)::NUMERIC / SUM(original_tokens)) * 100, 2)
         ELSE 0
    END as savings_percent
FROM command_records;

-- View for savings by command type
CREATE OR REPLACE VIEW savings_by_command_type AS
SELECT
    command_type,
    COUNT(*) as command_count,
    SUM(savings) as total_savings,
    ROUND(AVG(savings)::NUMERIC, 2) as avg_savings,
    SUM(original_tokens) as total_original,
    SUM(optimized_tokens) as total_optimized
FROM command_records
GROUP BY command_type
ORDER BY total_savings DESC;

-- View for savings by user
CREATE OR REPLACE VIEW savings_by_user AS
SELECT
    user_id,
    COUNT(*) as command_count,
    SUM(savings) as total_savings,
    ROUND(AVG(savings)::NUMERIC, 2) as avg_savings,
    ARRAY_AGG(DISTINCT command_type) as command_types
FROM command_records
GROUP BY user_id
ORDER BY total_savings DESC;

-- Function to clean up old records (called by scheduled job)
CREATE OR REPLACE FUNCTION cleanup_old_command_records(retention_days INT DEFAULT 90)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM command_records
    WHERE timestamp < now() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get time series data for trends
CREATE OR REPLACE FUNCTION get_savings_trends(
    period_days INT DEFAULT 7,
    bucket_interval TEXT DEFAULT 'day'
)
RETURNS TABLE (
    bucket TIMESTAMPTZ,
    total_savings BIGINT,
    command_count BIGINT,
    avg_savings NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        date_trunc(bucket_interval, timestamp) as bucket,
        SUM(savings)::BIGINT as total_savings,
        COUNT(*)::BIGINT as command_count,
        ROUND(AVG(savings)::NUMERIC, 2) as avg_savings
    FROM command_records
    WHERE timestamp >= now() - (period_days || ' days')::INTERVAL
    GROUP BY date_trunc(bucket_interval, timestamp)
    ORDER BY bucket;
END;
$$ LANGUAGE plpgsql;
