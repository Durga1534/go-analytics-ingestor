-- Create analytics table for event persistence
CREATE TABLE IF NOT EXISTS analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_timestamp CHECK (timestamp <= CURRENT_TIMESTAMP)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_analytics_type ON analytics(type);
CREATE INDEX IF NOT EXISTS idx_analytics_timestamp ON analytics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_created_at ON analytics(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_event_id ON analytics(event_id);

-- Create table for monitoring metrics
CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    events_processed BIGINT NOT NULL,
    stream_pending BIGINT NOT NULL,
    memory_usage_mb BIGINT NOT NULL,
    uptime_seconds BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for metrics queries
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON analytics TO analytics;
GRANT SELECT, INSERT ON metrics TO analytics;
