-- Add timezone field to commutes table for explicit timezone handling
-- Uses IANA timezone identifiers (e.g., "Europe/Amsterdam", "America/New_York")

ALTER TABLE commutes
ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'Europe/Amsterdam';

-- Add constraint to ensure valid format (basic check for slash-separated format)
-- Full validation is done at application level using Go's time.LoadLocation
ALTER TABLE commutes
ADD CONSTRAINT chk_timezone_format CHECK (timezone ~ '^[A-Za-z_]+/[A-Za-z_]+$' OR timezone = 'UTC');

COMMENT ON COLUMN commutes.timezone IS 'IANA timezone identifier for interpreting preferredArrivalTimeLocal';
