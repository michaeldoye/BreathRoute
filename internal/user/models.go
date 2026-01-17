// Package user provides user profile and settings management.
//
// # PII Considerations
//
// This package handles user profile data with minimal PII collection:
//
// Data Stored:
//   - UserID: Internal identifier (not PII, randomly generated)
//   - Locale: Language/region preference (e.g., "nl-NL") - minimal PII risk
//   - Units: Display unit preference (METRIC/IMPERIAL) - not PII
//   - ExposureWeights: Sensitivity preferences (0-1 values) - not PII
//   - RouteConstraints: Routing preferences - not PII
//
// Data NOT Stored:
//   - Name, email, phone (handled separately in auth with Apple's privacy relay)
//   - Location history (routes are computed on-demand, not stored)
//   - Health data (sensitivity weights are preferences, not medical data)
//
// GDPR Compliance:
//   - All user data can be exported via /v1/gdpr/export-requests
//   - All user data can be deleted via /v1/gdpr/deletion-requests
//   - Data minimization: only essential preferences are stored
package user

import (
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// User represents a user's complete profile and settings.
type User struct {
	// ID is the unique user identifier (format: usr_XXXX).
	ID string

	// Locale is the user's preferred language/region (BCP 47 format, e.g., "nl-NL").
	Locale string

	// Units is the user's preferred unit system for distances.
	Units models.Units

	// Profile contains the user's sensitivity and routing preferences.
	Profile *Profile

	// Consents contains the user's privacy consent states.
	Consents *Consents

	// CreatedAt is when the user was created.
	CreatedAt time.Time

	// UpdatedAt is when the user was last updated.
	UpdatedAt time.Time
}

// Profile represents the user's sensitivity and routing preferences.
type Profile struct {
	// Weights define the relative importance of different exposure factors.
	Weights ExposureWeights

	// Constraints define routing preferences.
	Constraints RouteConstraints

	// CreatedAt is when the profile was created.
	CreatedAt time.Time

	// UpdatedAt is when the profile was last updated.
	UpdatedAt time.Time
}

// ExposureWeights represents the relative importance of pollutant factors.
// All values should be in the range [0, 1].
type ExposureWeights struct {
	NO2    float64
	PM25   float64
	O3     float64
	Pollen float64
}

// RouteConstraints represents route generation preferences.
type RouteConstraints struct {
	AvoidMajorRoads          bool
	PreferParks              *bool
	MaxExtraMinutesVsFastest *int
	MaxTransfers             *int
}

// Consents represents the user's privacy consent states.
type Consents struct {
	Analytics         bool
	Marketing         bool
	PushNotifications bool
	UpdatedAt         time.Time
}

// DefaultUser returns a new user with default settings.
func DefaultUser(id string) *User {
	now := time.Now()
	return &User{
		ID:        id,
		Locale:    "nl-NL",
		Units:     models.UnitsMetric,
		Profile:   DefaultProfile(),
		Consents:  DefaultConsents(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DefaultProfile returns a profile with default settings.
func DefaultProfile() *Profile {
	now := time.Now()
	return &Profile{
		Weights: ExposureWeights{
			NO2:    0.4,
			PM25:   0.3,
			O3:     0.2,
			Pollen: 0.1,
		},
		Constraints: RouteConstraints{
			AvoidMajorRoads: false,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DefaultConsents returns consents with all options disabled by default.
func DefaultConsents() *Consents {
	return &Consents{
		Analytics:         false,
		Marketing:         false,
		PushNotifications: false,
		UpdatedAt:         time.Now(),
	}
}
