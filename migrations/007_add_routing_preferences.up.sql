-- Add routing preferences fields to user_profiles table
-- Story 2018: Store routing preferences (mode + avoidances + sensitivity)

-- Preferred transport mode (BIKE, WALK, TRANSIT)
ALTER TABLE user_profiles
ADD COLUMN preferred_mode VARCHAR(10) NOT NULL DEFAULT 'BIKE';

ALTER TABLE user_profiles
ADD CONSTRAINT chk_preferred_mode CHECK (preferred_mode IN ('BIKE', 'WALK', 'TRANSIT'));

COMMENT ON COLUMN user_profiles.preferred_mode IS 'User preferred transport mode for route planning';

-- Exposure sensitivity level (LOW, MEDIUM, HIGH)
ALTER TABLE user_profiles
ADD COLUMN exposure_sensitivity VARCHAR(10) NOT NULL DEFAULT 'MEDIUM';

ALTER TABLE user_profiles
ADD CONSTRAINT chk_exposure_sensitivity CHECK (exposure_sensitivity IN ('LOW', 'MEDIUM', 'HIGH'));

COMMENT ON COLUMN user_profiles.exposure_sensitivity IS 'User sensitivity to air quality exposure (affects scoring weight)';
