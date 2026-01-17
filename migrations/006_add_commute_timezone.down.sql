-- Remove timezone field from commutes table

ALTER TABLE commutes
DROP CONSTRAINT IF EXISTS chk_timezone_format;

ALTER TABLE commutes
DROP COLUMN IF EXISTS timezone;
