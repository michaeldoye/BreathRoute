-- Remove routing preferences fields from user_profiles table

ALTER TABLE user_profiles
DROP CONSTRAINT IF EXISTS chk_exposure_sensitivity;

ALTER TABLE user_profiles
DROP COLUMN IF EXISTS exposure_sensitivity;

ALTER TABLE user_profiles
DROP CONSTRAINT IF EXISTS chk_preferred_mode;

ALTER TABLE user_profiles
DROP COLUMN IF EXISTS preferred_mode;
