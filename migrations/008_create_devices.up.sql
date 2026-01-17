-- Create devices table for push notification tokens
-- Story 2019: Implement device registration endpoint for push tokens

CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(10) NOT NULL,
    token TEXT NOT NULL,
    device_model VARCHAR(255),
    os_version VARCHAR(50),
    app_version VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure platform is valid
    CONSTRAINT chk_platform CHECK (platform IN ('FCM', 'APNS'))
);

-- Index for looking up devices by user
CREATE INDEX idx_devices_user_id ON devices(user_id);

-- Unique constraint on token to prevent duplicate registrations
-- If same token is registered again, we update instead of insert
CREATE UNIQUE INDEX idx_devices_token ON devices(token);

COMMENT ON TABLE devices IS 'Push notification device tokens for users';
COMMENT ON COLUMN devices.token IS 'Full push token (APNs device token or FCM registration token)';
COMMENT ON COLUMN devices.platform IS 'Push platform: FCM (Firebase Cloud Messaging) or APNS (Apple Push Notification Service)';
