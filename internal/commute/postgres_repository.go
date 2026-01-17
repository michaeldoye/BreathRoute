package commute

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

// NewPostgresRepository creates a new PostgreSQL commute repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Get retrieves a commute by ID.
func (r *PostgresRepository) Get(ctx context.Context, id string) (*Commute, error) {
	query := `
		SELECT
			id, user_id, label,
			origin_lat, origin_lon, origin_geohash,
			destination_lat, destination_lon, destination_geohash,
			days_of_week, preferred_arrival_time_local, notes,
			created_at, updated_at
		FROM commutes
		WHERE id = $1
	`

	return r.scanCommute(ctx, query, id)
}

// GetByUserAndID retrieves a commute by user ID and commute ID.
func (r *PostgresRepository) GetByUserAndID(ctx context.Context, userID, commuteID string) (*Commute, error) {
	query := `
		SELECT
			id, user_id, label,
			origin_lat, origin_lon, origin_geohash,
			destination_lat, destination_lon, destination_geohash,
			days_of_week, preferred_arrival_time_local, notes,
			created_at, updated_at
		FROM commutes
		WHERE id = $1 AND user_id = $2
	`

	return r.scanCommute(ctx, query, commuteID, userID)
}

// scanCommute scans a commute from a query result.
func (r *PostgresRepository) scanCommute(ctx context.Context, query string, args ...interface{}) (*Commute, error) {
	var commute Commute

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&commute.ID,
		&commute.UserID,
		&commute.Label,
		&commute.Origin.Point.Lat,
		&commute.Origin.Point.Lon,
		&commute.Origin.Geohash,
		&commute.Destination.Point.Lat,
		&commute.Destination.Point.Lon,
		&commute.Destination.Geohash,
		&commute.DaysOfWeek,
		&commute.PreferredArrivalTimeLocal,
		&commute.Notes,
		&commute.CreatedAt,
		&commute.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommuteNotFound
		}
		return nil, err
	}

	return &commute, nil
}

// List retrieves all commutes for a user with pagination.
func (r *PostgresRepository) List(ctx context.Context, userID string, opts ListOptions) (*ListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	// Fetch one extra to determine if there are more results
	fetchLimit := limit + 1

	query := `
		SELECT
			id, user_id, label,
			origin_lat, origin_lon, origin_geohash,
			destination_lat, destination_lon, destination_geohash,
			days_of_week, preferred_arrival_time_local, notes,
			created_at, updated_at
		FROM commutes
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, userID, fetchLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commutes []*Commute
	for rows.Next() {
		var commute Commute
		err := rows.Scan(
			&commute.ID,
			&commute.UserID,
			&commute.Label,
			&commute.Origin.Point.Lat,
			&commute.Origin.Point.Lon,
			&commute.Origin.Geohash,
			&commute.Destination.Point.Lat,
			&commute.Destination.Point.Lon,
			&commute.Destination.Geohash,
			&commute.DaysOfWeek,
			&commute.PreferredArrivalTimeLocal,
			&commute.Notes,
			&commute.CreatedAt,
			&commute.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		commutes = append(commutes, &commute)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &ListResult{
		Items: commutes,
	}

	// If we got more results than the limit, there are more pages
	if len(commutes) > limit {
		result.Items = commutes[:limit]
		// Use the last item's ID as the cursor for the next page
		result.NextCursor = commutes[limit-1].ID
	}

	return result, nil
}

// Create creates a new commute.
func (r *PostgresRepository) Create(ctx context.Context, commute *Commute) error {
	query := `
		INSERT INTO commutes (
			id, user_id, label,
			origin_lat, origin_lon, origin_geohash,
			destination_lat, destination_lon, destination_geohash,
			days_of_week, preferred_arrival_time_local, notes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.pool.Exec(ctx, query,
		commute.ID,
		commute.UserID,
		commute.Label,
		commute.Origin.Point.Lat,
		commute.Origin.Point.Lon,
		commute.Origin.Geohash,
		commute.Destination.Point.Lat,
		commute.Destination.Point.Lon,
		commute.Destination.Geohash,
		commute.DaysOfWeek,
		commute.PreferredArrivalTimeLocal,
		commute.Notes,
		commute.CreatedAt,
		commute.UpdatedAt,
	)
	return err
}

// Update updates an existing commute.
func (r *PostgresRepository) Update(ctx context.Context, commute *Commute) error {
	query := `
		UPDATE commutes SET
			label = $2,
			origin_lat = $3,
			origin_lon = $4,
			origin_geohash = $5,
			destination_lat = $6,
			destination_lon = $7,
			destination_geohash = $8,
			days_of_week = $9,
			preferred_arrival_time_local = $10,
			notes = $11,
			updated_at = $12
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		commute.ID,
		commute.Label,
		commute.Origin.Point.Lat,
		commute.Origin.Point.Lon,
		commute.Origin.Geohash,
		commute.Destination.Point.Lat,
		commute.Destination.Point.Lon,
		commute.Destination.Geohash,
		commute.DaysOfWeek,
		commute.PreferredArrivalTimeLocal,
		commute.Notes,
		commute.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrCommuteNotFound
	}

	return nil
}

// Delete deletes a commute by ID.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM commutes WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// Ensure PostgresRepository implements Repository interface.
var _ Repository = (*PostgresRepository)(nil)
