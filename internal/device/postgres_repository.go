package device

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is a PostgreSQL implementation of Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL device repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Get retrieves a device by user ID and device ID.
func (r *PostgresRepository) Get(ctx context.Context, userID, deviceID string) (*Device, error) {
	query := `
		SELECT id, user_id, platform, token, device_model, os_version, app_version, created_at, updated_at
		FROM devices
		WHERE id = $1 AND user_id = $2
	`

	return r.scanDevice(ctx, query, deviceID, userID)
}

// GetByToken retrieves a device by token.
func (r *PostgresRepository) GetByToken(ctx context.Context, token string) (*Device, error) {
	query := `
		SELECT id, user_id, platform, token, device_model, os_version, app_version, created_at, updated_at
		FROM devices
		WHERE token = $1
	`

	return r.scanDevice(ctx, query, token)
}

// scanDevice scans a single device from a query.
func (r *PostgresRepository) scanDevice(ctx context.Context, query string, args ...interface{}) (*Device, error) {
	var device Device

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&device.ID,
		&device.UserID,
		&device.Platform,
		&device.Token,
		&device.DeviceModel,
		&device.OSVersion,
		&device.AppVersion,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	return &device, nil
}

// ListByUser retrieves all devices for a user.
func (r *PostgresRepository) ListByUser(ctx context.Context, userID string, opts ListOptions) (*ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	fetchLimit := limit + 1

	query := `
		SELECT id, user_id, platform, token, device_model, os_version, app_version, created_at, updated_at
		FROM devices
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, fetchLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		var device Device
		err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.Platform,
			&device.Token,
			&device.DeviceModel,
			&device.OSVersion,
			&device.AppVersion,
			&device.CreatedAt,
			&device.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &ListResult{
		Items: devices,
	}

	if len(devices) > limit {
		result.Items = devices[:limit]
		result.NextCursor = devices[limit-1].ID
	}

	return result, nil
}

// Create creates a new device.
func (r *PostgresRepository) Create(ctx context.Context, device *Device) error {
	query := `
		INSERT INTO devices (id, user_id, platform, token, device_model, os_version, app_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		device.ID,
		device.UserID,
		device.Platform,
		device.Token,
		device.DeviceModel,
		device.OSVersion,
		device.AppVersion,
		device.CreatedAt,
		device.UpdatedAt,
	)
	return err
}

// Update updates an existing device.
func (r *PostgresRepository) Update(ctx context.Context, device *Device) error {
	query := `
		UPDATE devices SET
			platform = $2,
			token = $3,
			device_model = $4,
			os_version = $5,
			app_version = $6,
			updated_at = $7
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		device.ID,
		device.Platform,
		device.Token,
		device.DeviceModel,
		device.OSVersion,
		device.AppVersion,
		device.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// Upsert creates or updates a device based on the token.
// Returns true if a new device was created, false if updated.
func (r *PostgresRepository) Upsert(ctx context.Context, device *Device) (bool, error) {
	// Use INSERT ... ON CONFLICT to handle upsert
	// We use the token as the conflict target since tokens should be unique
	query := `
		INSERT INTO devices (id, user_id, platform, token, device_model, os_version, app_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (token) DO UPDATE SET
			id = EXCLUDED.id,
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			device_model = EXCLUDED.device_model,
			os_version = EXCLUDED.os_version,
			app_version = EXCLUDED.app_version,
			updated_at = EXCLUDED.updated_at
		RETURNING (xmax = 0) AS inserted
	`

	var inserted bool
	err := r.pool.QueryRow(ctx, query,
		device.ID,
		device.UserID,
		device.Platform,
		device.Token,
		device.DeviceModel,
		device.OSVersion,
		device.AppVersion,
		device.CreatedAt,
		device.UpdatedAt,
	).Scan(&inserted)

	if err != nil {
		return false, err
	}

	return inserted, nil
}

// Delete deletes a device.
func (r *PostgresRepository) Delete(ctx context.Context, userID, deviceID string) error {
	query := `DELETE FROM devices WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, deviceID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}

	return nil
}

// DeleteByUser deletes all devices for a user.
func (r *PostgresRepository) DeleteByUser(ctx context.Context, userID string) error {
	query := `DELETE FROM devices WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// Ensure PostgresRepository implements Repository interface.
var _ Repository = (*PostgresRepository)(nil)
