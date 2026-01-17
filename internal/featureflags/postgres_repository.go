package featureflags

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is a PostgreSQL implementation of Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL feature flags repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetFlag retrieves a single feature flag by key.
func (r *PostgresRepository) GetFlag(ctx context.Context, key string) (*Flag, error) {
	query := `
		SELECT key, value, updated_at
		FROM feature_flags
		WHERE key = $1
	`

	var (
		flag      Flag
		valueJSON []byte
	)

	err := r.pool.QueryRow(ctx, query, key).Scan(
		&flag.Key,
		&valueJSON,
		&flag.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFlagNotFound
		}
		return nil, err
	}

	// Unmarshal JSON value
	if err := json.Unmarshal(valueJSON, &flag.Value); err != nil {
		return nil, err
	}

	return &flag, nil
}

// GetAllFlags retrieves all feature flags.
func (r *PostgresRepository) GetAllFlags(ctx context.Context) (map[string]*Flag, error) {
	query := `
		SELECT key, value, updated_at
		FROM feature_flags
		ORDER BY key
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make(map[string]*Flag)
	for rows.Next() {
		var (
			flag      Flag
			valueJSON []byte
		)

		err := rows.Scan(
			&flag.Key,
			&valueJSON,
			&flag.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON value
		if err := json.Unmarshal(valueJSON, &flag.Value); err != nil {
			return nil, err
		}

		flags[flag.Key] = &flag
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return flags, nil
}

// SetFlag creates or updates a feature flag.
func (r *PostgresRepository) SetFlag(ctx context.Context, flag *Flag) error {
	query := `
		INSERT INTO feature_flags (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	valueJSON, err := json.Marshal(flag.Value)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, flag.Key, valueJSON, time.Now())
	return err
}

// SetFlags creates or updates multiple feature flags atomically.
func (r *PostgresRepository) SetFlags(ctx context.Context, flags []*Flag) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback error is not critical

	query := `
		INSERT INTO feature_flags (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	for _, flag := range flags {
		valueJSON, err := json.Marshal(flag.Value)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, query, flag.Key, valueJSON, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// DeleteFlag removes a feature flag by key.
func (r *PostgresRepository) DeleteFlag(ctx context.Context, key string) error {
	query := `DELETE FROM feature_flags WHERE key = $1`
	_, err := r.pool.Exec(ctx, query, key)
	return err
}

// Ensure PostgresRepository implements Repository interface.
var _ Repository = (*PostgresRepository)(nil)
