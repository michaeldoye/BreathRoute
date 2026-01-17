package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository is a PostgreSQL implementation of UserRepository.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgreSQL user repository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// FindByAppleSub finds a user by their Apple subject identifier.
func (r *PostgresUserRepository) FindByAppleSub(ctx context.Context, appleSub string) (*User, error) {
	query := `
		SELECT id, apple_sub, email, locale, created_at, updated_at
		FROM users
		WHERE apple_sub = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, appleSub).Scan(
		&user.ID,
		&user.AppleSub,
		&user.Email,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// Create creates a new user.
func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, apple_sub, email, locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.AppleSub,
		user.Email,
		user.Locale,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// FindByID finds a user by their internal ID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, apple_sub, email, locale, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.AppleSub,
		&user.Email,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// PostgresRefreshTokenRepository is a PostgreSQL implementation of RefreshTokenRepository.
type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository creates a new PostgreSQL refresh token repository.
func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

// Create stores a new refresh token.
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, token, user_id, expires_at, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.Token,
		token.UserID,
		token.ExpiresAt,
		token.CreatedAt,
		token.RevokedAt,
	)
	return err
}

// FindByToken finds a refresh token by its value.
func (r *PostgresRefreshTokenRepository) FindByToken(ctx context.Context, tokenValue string) (*RefreshToken, error) {
	query := `
		SELECT id, token, user_id, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token = $1
	`

	var token RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenValue).Scan(
		&token.ID,
		&token.Token,
		&token.UserID,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	return &token, nil
}

// Revoke marks a refresh token as revoked.
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, tokenValue string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE token = $2 AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), tokenValue)
	return err
}

// RevokeAllForUser revokes all refresh tokens for a user.
func (r *PostgresRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), userID)
	return err
}
