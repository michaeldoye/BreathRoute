package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// PostgresRepository is a PostgreSQL implementation of Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL user repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Get retrieves a user by ID.
func (r *PostgresRepository) Get(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT
			user_id, locale, units,
			weight_no2, weight_pm25, weight_o3, weight_pollen,
			avoid_major_roads, prefer_parks, max_extra_minutes_vs_fastest, max_transfers,
			consent_analytics, consent_marketing, consent_push_notifications, consents_updated_at,
			created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1
	`

	var (
		userID                     string
		locale                     string
		units                      models.Units
		weightNO2                  float64
		weightPM25                 float64
		weightO3                   float64
		weightPollen               float64
		avoidMajorRoads            bool
		preferParks                *bool
		maxExtraMinutesVsFastest   *int
		maxTransfers               *int
		consentAnalytics           bool
		consentMarketing           bool
		consentPushNotifications   bool
		consentsUpdatedAt          time.Time
		createdAt                  time.Time
		updatedAt                  time.Time
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&userID,
		&locale,
		&units,
		&weightNO2,
		&weightPM25,
		&weightO3,
		&weightPollen,
		&avoidMajorRoads,
		&preferParks,
		&maxExtraMinutesVsFastest,
		&maxTransfers,
		&consentAnalytics,
		&consentMarketing,
		&consentPushNotifications,
		&consentsUpdatedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user := &User{
		ID:        userID,
		Locale:    locale,
		Units:     units,
		Profile: &Profile{
			Weights: ExposureWeights{
				NO2:    weightNO2,
				PM25:   weightPM25,
				O3:     weightO3,
				Pollen: weightPollen,
			},
			Constraints: RouteConstraints{
				AvoidMajorRoads:          avoidMajorRoads,
				PreferParks:              preferParks,
				MaxExtraMinutesVsFastest: maxExtraMinutesVsFastest,
				MaxTransfers:             maxTransfers,
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Consents: &Consents{
			Analytics:         consentAnalytics,
			Marketing:         consentMarketing,
			PushNotifications: consentPushNotifications,
			UpdatedAt:         consentsUpdatedAt,
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return user, nil
}

// Create creates a new user profile.
func (r *PostgresRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO user_profiles (
			user_id, locale, units,
			weight_no2, weight_pm25, weight_o3, weight_pollen,
			avoid_major_roads, prefer_parks, max_extra_minutes_vs_fastest, max_transfers,
			consent_analytics, consent_marketing, consent_push_notifications, consents_updated_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	profile := user.Profile
	if profile == nil {
		profile = DefaultProfile()
	}
	consents := user.Consents
	if consents == nil {
		consents = DefaultConsents()
	}

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Locale,
		user.Units,
		profile.Weights.NO2,
		profile.Weights.PM25,
		profile.Weights.O3,
		profile.Weights.Pollen,
		profile.Constraints.AvoidMajorRoads,
		profile.Constraints.PreferParks,
		profile.Constraints.MaxExtraMinutesVsFastest,
		profile.Constraints.MaxTransfers,
		consents.Analytics,
		consents.Marketing,
		consents.PushNotifications,
		consents.UpdatedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// Update updates an existing user profile.
func (r *PostgresRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE user_profiles SET
			locale = $2,
			units = $3,
			weight_no2 = $4,
			weight_pm25 = $5,
			weight_o3 = $6,
			weight_pollen = $7,
			avoid_major_roads = $8,
			prefer_parks = $9,
			max_extra_minutes_vs_fastest = $10,
			max_transfers = $11,
			consent_analytics = $12,
			consent_marketing = $13,
			consent_push_notifications = $14,
			consents_updated_at = $15,
			updated_at = $16
		WHERE user_id = $1
	`

	profile := user.Profile
	if profile == nil {
		profile = DefaultProfile()
	}
	consents := user.Consents
	if consents == nil {
		consents = DefaultConsents()
	}

	result, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Locale,
		user.Units,
		profile.Weights.NO2,
		profile.Weights.PM25,
		profile.Weights.O3,
		profile.Weights.Pollen,
		profile.Constraints.AvoidMajorRoads,
		profile.Constraints.PreferParks,
		profile.Constraints.MaxExtraMinutesVsFastest,
		profile.Constraints.MaxTransfers,
		consents.Analytics,
		consents.Marketing,
		consents.PushNotifications,
		consents.UpdatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete deletes a user profile.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM user_profiles WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateOrUpdate creates a user profile if it doesn't exist, or updates it if it does.
// This is useful when a user is created in auth but needs a profile.
func (r *PostgresRepository) CreateOrUpdate(ctx context.Context, user *User) error {
	query := `
		INSERT INTO user_profiles (
			user_id, locale, units,
			weight_no2, weight_pm25, weight_o3, weight_pollen,
			avoid_major_roads, prefer_parks, max_extra_minutes_vs_fastest, max_transfers,
			consent_analytics, consent_marketing, consent_push_notifications, consents_updated_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (user_id) DO UPDATE SET
			locale = EXCLUDED.locale,
			units = EXCLUDED.units,
			weight_no2 = EXCLUDED.weight_no2,
			weight_pm25 = EXCLUDED.weight_pm25,
			weight_o3 = EXCLUDED.weight_o3,
			weight_pollen = EXCLUDED.weight_pollen,
			avoid_major_roads = EXCLUDED.avoid_major_roads,
			prefer_parks = EXCLUDED.prefer_parks,
			max_extra_minutes_vs_fastest = EXCLUDED.max_extra_minutes_vs_fastest,
			max_transfers = EXCLUDED.max_transfers,
			consent_analytics = EXCLUDED.consent_analytics,
			consent_marketing = EXCLUDED.consent_marketing,
			consent_push_notifications = EXCLUDED.consent_push_notifications,
			consents_updated_at = EXCLUDED.consents_updated_at,
			updated_at = EXCLUDED.updated_at
	`

	profile := user.Profile
	if profile == nil {
		profile = DefaultProfile()
	}
	consents := user.Consents
	if consents == nil {
		consents = DefaultConsents()
	}

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Locale,
		user.Units,
		profile.Weights.NO2,
		profile.Weights.PM25,
		profile.Weights.O3,
		profile.Weights.Pollen,
		profile.Constraints.AvoidMajorRoads,
		profile.Constraints.PreferParks,
		profile.Constraints.MaxExtraMinutesVsFastest,
		profile.Constraints.MaxTransfers,
		consents.Analytics,
		consents.Marketing,
		consents.PushNotifications,
		consents.UpdatedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// Ensure PostgresRepository implements Repository interface.
var _ Repository = (*PostgresRepository)(nil)
