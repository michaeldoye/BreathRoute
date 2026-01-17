-- Create commutes table for saved user commutes
CREATE TABLE IF NOT EXISTS commutes (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Commute details
    label VARCHAR(80) NOT NULL,

    -- Origin location (using PostGIS geography type for accurate distance calculations)
    origin_lat DOUBLE PRECISION NOT NULL,
    origin_lon DOUBLE PRECISION NOT NULL,
    origin_geohash VARCHAR(12),
    origin_point GEOGRAPHY(POINT, 4326) GENERATED ALWAYS AS (
        ST_SetSRID(ST_MakePoint(origin_lon, origin_lat), 4326)::geography
    ) STORED,

    -- Destination location
    destination_lat DOUBLE PRECISION NOT NULL,
    destination_lon DOUBLE PRECISION NOT NULL,
    destination_geohash VARCHAR(12),
    destination_point GEOGRAPHY(POINT, 4326) GENERATED ALWAYS AS (
        ST_SetSRID(ST_MakePoint(destination_lon, destination_lat), 4326)::geography
    ) STORED,

    -- Schedule
    days_of_week INTEGER[] NOT NULL DEFAULT '{}',
    preferred_arrival_time_local VARCHAR(5) NOT NULL,

    -- Optional notes
    notes TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_origin_lat CHECK (origin_lat >= -90 AND origin_lat <= 90),
    CONSTRAINT chk_origin_lon CHECK (origin_lon >= -180 AND origin_lon <= 180),
    CONSTRAINT chk_destination_lat CHECK (destination_lat >= -90 AND destination_lat <= 90),
    CONSTRAINT chk_destination_lon CHECK (destination_lon >= -180 AND destination_lon <= 180),
    CONSTRAINT chk_label_length CHECK (char_length(label) <= 80),
    CONSTRAINT chk_notes_length CHECK (notes IS NULL OR char_length(notes) <= 500),
    CONSTRAINT chk_days_of_week CHECK (days_of_week <@ ARRAY[1,2,3,4,5,6,7])
);

-- Index for user's commutes lookup
CREATE INDEX IF NOT EXISTS idx_commutes_user_id ON commutes(user_id);

-- Spatial index for origin points (for future proximity queries)
CREATE INDEX IF NOT EXISTS idx_commutes_origin_point ON commutes USING GIST(origin_point);

-- Spatial index for destination points
CREATE INDEX IF NOT EXISTS idx_commutes_destination_point ON commutes USING GIST(destination_point);
