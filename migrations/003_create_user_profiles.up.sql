-- Create user_profiles table for user settings and preferences
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id VARCHAR(26) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- General settings
    locale VARCHAR(10) NOT NULL DEFAULT 'nl-NL',
    units VARCHAR(10) NOT NULL DEFAULT 'METRIC',

    -- Exposure weights (0-1 range)
    weight_no2 DECIMAL(3,2) NOT NULL DEFAULT 0.40,
    weight_pm25 DECIMAL(3,2) NOT NULL DEFAULT 0.30,
    weight_o3 DECIMAL(3,2) NOT NULL DEFAULT 0.20,
    weight_pollen DECIMAL(3,2) NOT NULL DEFAULT 0.10,

    -- Route constraints
    avoid_major_roads BOOLEAN NOT NULL DEFAULT FALSE,
    prefer_parks BOOLEAN,
    max_extra_minutes_vs_fastest INTEGER,
    max_transfers INTEGER,

    -- Consents
    consent_analytics BOOLEAN NOT NULL DEFAULT FALSE,
    consent_marketing BOOLEAN NOT NULL DEFAULT FALSE,
    consent_push_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    consents_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
