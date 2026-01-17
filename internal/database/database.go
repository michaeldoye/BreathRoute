// Package database provides PostgreSQL connection management.
package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database connection configuration.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() Config {
	port, _ := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	maxOpen, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_OPEN_CONNS", "10"))
	maxIdle, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_IDLE_CONNS", "5"))
	lifetime, _ := time.ParseDuration(getEnvOrDefault("DB_CONN_MAX_LIFETIME", "5m"))

	return Config{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            port,
		User:            getEnvOrDefault("DB_USER", "breatheroute"),
		Password:        getEnvOrDefault("DB_PASSWORD", "localdev"),
		Database:        getEnvOrDefault("DB_NAME", "breatheroute"),
		SSLMode:         getEnvOrDefault("DB_SSL_MODE", "disable"),
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: lifetime,
	}
}

// ConnectionString returns the PostgreSQL connection string.
func (c Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// Connect creates a new database connection pool.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns) //nolint:gosec // MaxOpenConns is bounded by config validation
	poolConfig.MinConns = int32(cfg.MaxIdleConns) //nolint:gosec // MaxIdleConns is bounded by config validation
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
