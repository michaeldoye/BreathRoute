-- BreatheRoute Local Development Database Initialization
-- This script runs automatically when the PostgreSQL container starts for the first time

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- For text search

-- Verify PostGIS is working
SELECT PostGIS_Version();

-- Create application schema (optional, for organization)
-- CREATE SCHEMA IF NOT EXISTS app;

-- Grant permissions (the breatheroute user is created by POSTGRES_USER env var)
-- Additional grants if needed for specific schemas

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'BreatheRoute database initialized successfully';
    RAISE NOTICE 'PostGIS version: %', PostGIS_Version();
END $$;
