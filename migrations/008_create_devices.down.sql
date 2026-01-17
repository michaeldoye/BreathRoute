-- Drop devices table

DROP INDEX IF EXISTS idx_devices_token;
DROP INDEX IF EXISTS idx_devices_user_id;
DROP TABLE IF EXISTS devices;
