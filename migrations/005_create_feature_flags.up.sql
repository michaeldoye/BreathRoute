-- Create feature_flags table for runtime configuration
CREATE TABLE IF NOT EXISTS feature_flags (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL DEFAULT 'false'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default feature flags
INSERT INTO feature_flags (key, value, updated_at) VALUES
    ('disable_train_mode', 'false'::jsonb, NOW()),
    ('cached_only_air_quality', 'false'::jsonb, NOW()),
    ('disable_alerts_sending', 'false'::jsonb, NOW()),
    ('disable_pollen_factor', 'false'::jsonb, NOW()),
    ('routing_bike_only', 'false'::jsonb, NOW()),
    ('enable_time_shift', 'true'::jsonb, NOW())
ON CONFLICT (key) DO NOTHING;
