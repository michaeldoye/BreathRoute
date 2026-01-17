-- Create users table for authentication
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(26) PRIMARY KEY,
    apple_sub VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255),
    locale VARCHAR(10) NOT NULL DEFAULT 'nl-NL',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for Apple subject lookup (used during authentication)
CREATE INDEX IF NOT EXISTS idx_users_apple_sub ON users(apple_sub);

-- Index for email lookup (optional, for future use)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
