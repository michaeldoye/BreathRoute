package user

import (
	"context"
	"errors"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// Service errors.
var (
	ErrUserExists = errors.New("user already exists")
)

// Service provides user profile operations.
type Service struct {
	repo Repository
}

// NewService creates a new user service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetMe retrieves the user's account summary.
func (s *Service) GetMe(ctx context.Context, userID string) (*models.Me, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.Me{
		UserID:    user.ID,
		Locale:    user.Locale,
		Units:     user.Units,
		CreatedAt: models.Timestamp(user.CreatedAt),
	}, nil
}

// UpdateMe updates the user's account settings.
func (s *Service) UpdateMe(ctx context.Context, userID string, input *models.MeInput) (*models.Me, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if input.Locale != nil {
		user.Locale = *input.Locale
	}
	if input.Units != nil {
		user.Units = *input.Units
	}
	user.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &models.Me{
		UserID:    user.ID,
		Locale:    user.Locale,
		Units:     user.Units,
		CreatedAt: models.Timestamp(user.CreatedAt),
	}, nil
}

// GetProfile retrieves the user's sensitivity profile.
func (s *Service) GetProfile(ctx context.Context, userID string) (*models.Profile, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Profile == nil {
		// Return default profile if none exists
		user.Profile = DefaultProfile()
	}

	return s.toAPIProfile(user.Profile), nil
}

// UpsertProfile creates or updates the user's sensitivity profile.
func (s *Service) UpsertProfile(ctx context.Context, userID string, input *models.ProfileInput) (*models.Profile, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if user.Profile == nil {
		user.Profile = DefaultProfile()
		user.Profile.CreatedAt = now
	}

	// Update profile
	user.Profile.Weights = ExposureWeights{
		NO2:    input.Weights.NO2,
		PM25:   input.Weights.PM25,
		O3:     input.Weights.O3,
		Pollen: input.Weights.Pollen,
	}
	user.Profile.Constraints = RouteConstraints{
		AvoidMajorRoads:          input.Constraints.AvoidMajorRoads,
		PreferParks:              input.Constraints.PreferParks,
		MaxExtraMinutesVsFastest: input.Constraints.MaxExtraMinutesVsFastest,
		MaxTransfers:             input.Constraints.MaxTransfers,
	}

	// Update routing preferences if provided
	if input.PreferredMode != nil {
		user.Profile.PreferredMode = TransportMode(*input.PreferredMode)
	}
	if input.ExposureSensitivity != nil {
		user.Profile.ExposureSensitivity = ExposureSensitivity(*input.ExposureSensitivity)
	}

	user.Profile.UpdatedAt = now
	user.UpdatedAt = now

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.toAPIProfile(user.Profile), nil
}

// GetConsents retrieves the user's consent states.
func (s *Service) GetConsents(ctx context.Context, userID string) (*models.Consents, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Consents == nil {
		user.Consents = DefaultConsents()
	}

	return &models.Consents{
		Analytics:         user.Consents.Analytics,
		Marketing:         user.Consents.Marketing,
		PushNotifications: user.Consents.PushNotifications,
		UpdatedAt:         models.Timestamp(user.Consents.UpdatedAt),
	}, nil
}

// UpdateConsents updates the user's consent states.
func (s *Service) UpdateConsents(ctx context.Context, userID string, input *models.ConsentsInput) (*models.Consents, error) {
	user, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if user.Consents == nil {
		user.Consents = DefaultConsents()
	}

	// Apply updates
	if input.Analytics != nil {
		user.Consents.Analytics = *input.Analytics
	}
	if input.Marketing != nil {
		user.Consents.Marketing = *input.Marketing
	}
	if input.PushNotifications != nil {
		user.Consents.PushNotifications = *input.PushNotifications
	}
	user.Consents.UpdatedAt = now
	user.UpdatedAt = now

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &models.Consents{
		Analytics:         user.Consents.Analytics,
		Marketing:         user.Consents.Marketing,
		PushNotifications: user.Consents.PushNotifications,
		UpdatedAt:         models.Timestamp(user.Consents.UpdatedAt),
	}, nil
}

// CreateUser creates a new user with default settings.
// This is typically called after authentication to ensure the user exists.
func (s *Service) CreateUser(ctx context.Context, userID, locale string) (*User, error) {
	// Check if user already exists
	existing, err := s.repo.Get(ctx, userID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Create new user with defaults
	user := DefaultUser(userID)
	if locale != "" {
		user.Locale = locale
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser deletes a user and all associated data.
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}

// toAPIProfile converts a domain Profile to an API Profile.
func (s *Service) toAPIProfile(p *Profile) *models.Profile {
	return &models.Profile{
		Weights: models.ExposureWeights{
			NO2:    p.Weights.NO2,
			PM25:   p.Weights.PM25,
			O3:     p.Weights.O3,
			Pollen: p.Weights.Pollen,
		},
		Constraints: models.RouteConstraints{
			AvoidMajorRoads:          p.Constraints.AvoidMajorRoads,
			PreferParks:              p.Constraints.PreferParks,
			MaxExtraMinutesVsFastest: p.Constraints.MaxExtraMinutesVsFastest,
			MaxTransfers:             p.Constraints.MaxTransfers,
		},
		PreferredMode:       models.TransportMode(p.PreferredMode),
		ExposureSensitivity: models.ExposureSensitivity(p.ExposureSensitivity),
		CreatedAt:           models.Timestamp(p.CreatedAt),
		UpdatedAt:           models.Timestamp(p.UpdatedAt),
	}
}
