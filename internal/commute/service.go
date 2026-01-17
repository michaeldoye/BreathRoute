package commute

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// Service errors.
var (
	ErrNotAuthorized = errors.New("not authorized to access this commute")
)

// Validation constants.
const (
	MaxLabelLength = 80
	MaxNotesLength = 500
)

// timeHHMMRegex validates HH:mm format.
var timeHHMMRegex = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d$`)

// Service provides commute operations.
type Service struct {
	repo Repository
}

// NewService creates a new commute service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List retrieves all commutes for a user.
func (s *Service) List(ctx context.Context, userID string, limit int) (*models.PagedCommutes, error) {
	result, err := s.repo.List(ctx, userID, ListOptions{Limit: limit})
	if err != nil {
		return nil, err
	}

	items := make([]models.Commute, 0, len(result.Items))
	for _, c := range result.Items {
		items = append(items, s.toAPICommute(c))
	}

	var nextCursor *string
	if result.NextCursor != "" {
		nextCursor = &result.NextCursor
	}

	return &models.PagedCommutes{
		Items: items,
		Meta: models.PagedResponseMeta{
			Limit:      limit,
			NextCursor: nextCursor,
		},
	}, nil
}

// Get retrieves a commute by ID for a user.
func (s *Service) Get(ctx context.Context, userID, commuteID string) (*models.Commute, error) {
	commute, err := s.repo.GetByUserAndID(ctx, userID, commuteID)
	if err != nil {
		if errors.Is(err, ErrCommuteNotFound) {
			return nil, ErrCommuteNotFound
		}
		return nil, err
	}

	result := s.toAPICommute(commute)
	return &result, nil
}

// Create creates a new commute for a user.
func (s *Service) Create(ctx context.Context, userID string, input *models.CommuteCreateRequest) (*models.Commute, error) {
	// Validate input
	if fieldErrors := s.validateCreateInput(input); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	now := time.Now()
	commuteID := "cmt_" + uuid.New().String()[:22]

	commute := &Commute{
		ID:     commuteID,
		UserID: userID,
		Label:  input.Label,
		Origin: Location{
			Point:   Point{Lat: input.Origin.Point.Lat, Lon: input.Origin.Point.Lon},
			Geohash: input.Origin.Geohash,
		},
		Destination: Location{
			Point:   Point{Lat: input.Destination.Point.Lat, Lon: input.Destination.Point.Lon},
			Geohash: input.Destination.Geohash,
		},
		DaysOfWeek:                input.DaysOfWeek,
		PreferredArrivalTimeLocal: input.PreferredArrivalTimeLocal,
		Notes:                     input.Notes,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	if err := s.repo.Create(ctx, commute); err != nil {
		return nil, err
	}

	result := s.toAPICommute(commute)
	return &result, nil
}

// Update updates an existing commute for a user.
func (s *Service) Update(ctx context.Context, userID, commuteID string, input *models.CommuteUpdateRequest) (*models.Commute, error) {
	// Get existing commute
	commute, err := s.repo.GetByUserAndID(ctx, userID, commuteID)
	if err != nil {
		if errors.Is(err, ErrCommuteNotFound) {
			return nil, ErrCommuteNotFound
		}
		return nil, err
	}

	// Validate input
	if fieldErrors := s.validateUpdateInput(input); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Apply updates
	if input.Label != nil {
		commute.Label = *input.Label
	}
	if input.Origin != nil {
		commute.Origin = Location{
			Point:   Point{Lat: input.Origin.Point.Lat, Lon: input.Origin.Point.Lon},
			Geohash: input.Origin.Geohash,
		}
	}
	if input.Destination != nil {
		commute.Destination = Location{
			Point:   Point{Lat: input.Destination.Point.Lat, Lon: input.Destination.Point.Lon},
			Geohash: input.Destination.Geohash,
		}
	}
	if input.DaysOfWeek != nil {
		commute.DaysOfWeek = input.DaysOfWeek
	}
	if input.PreferredArrivalTimeLocal != nil {
		commute.PreferredArrivalTimeLocal = *input.PreferredArrivalTimeLocal
	}
	if input.Notes != nil {
		commute.Notes = input.Notes
	}
	commute.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, commute); err != nil {
		return nil, err
	}

	result := s.toAPICommute(commute)
	return &result, nil
}

// Delete deletes a commute for a user.
func (s *Service) Delete(ctx context.Context, userID, commuteID string) error {
	// Verify ownership
	_, err := s.repo.GetByUserAndID(ctx, userID, commuteID)
	if err != nil {
		if errors.Is(err, ErrCommuteNotFound) {
			return ErrCommuteNotFound
		}
		return err
	}

	return s.repo.Delete(ctx, commuteID)
}

// validateCreateInput validates the create commute input.
func (s *Service) validateCreateInput(input *models.CommuteCreateRequest) []models.FieldError {
	var errs []models.FieldError

	// Validate label
	if input.Label == "" {
		errs = append(errs, models.FieldError{Field: "label", Message: "is required"})
	} else if len(input.Label) > MaxLabelLength {
		errs = append(errs, models.FieldError{Field: "label", Message: "must be at most 80 characters"})
	}

	// Validate origin coordinates
	errs = append(errs, s.validateLocation(&input.Origin, "origin")...)

	// Validate destination coordinates
	errs = append(errs, s.validateLocation(&input.Destination, "destination")...)

	// Validate days of week
	if len(input.DaysOfWeek) == 0 {
		errs = append(errs, models.FieldError{Field: "daysOfWeek", Message: "is required"})
	} else {
		for _, day := range input.DaysOfWeek {
			if day < 1 || day > 7 {
				errs = append(errs, models.FieldError{Field: "daysOfWeek", Message: "must contain values between 1 and 7"})
				break
			}
		}
	}

	// Validate preferred arrival time
	if input.PreferredArrivalTimeLocal == "" {
		errs = append(errs, models.FieldError{Field: "preferredArrivalTimeLocal", Message: "is required"})
	} else if !timeHHMMRegex.MatchString(input.PreferredArrivalTimeLocal) {
		errs = append(errs, models.FieldError{Field: "preferredArrivalTimeLocal", Message: "must be in HH:mm format"})
	}

	// Validate notes (optional)
	if input.Notes != nil && len(*input.Notes) > MaxNotesLength {
		errs = append(errs, models.FieldError{Field: "notes", Message: "must be at most 500 characters"})
	}

	return errs
}

// validateUpdateInput validates the update commute input.
func (s *Service) validateUpdateInput(input *models.CommuteUpdateRequest) []models.FieldError {
	var errs []models.FieldError

	// Validate label (optional)
	if input.Label != nil {
		errs = append(errs, s.validateOptionalLabel(*input.Label)...)
	}

	// Validate origin coordinates (optional)
	if input.Origin != nil {
		errs = append(errs, s.validateLocation(input.Origin, "origin")...)
	}

	// Validate destination coordinates (optional)
	if input.Destination != nil {
		errs = append(errs, s.validateLocation(input.Destination, "destination")...)
	}

	// Validate days of week (optional)
	if input.DaysOfWeek != nil {
		errs = append(errs, s.validateDaysOfWeek(input.DaysOfWeek, false)...)
	}

	// Validate preferred arrival time (optional)
	if input.PreferredArrivalTimeLocal != nil {
		errs = append(errs, s.validateOptionalArrivalTime(*input.PreferredArrivalTimeLocal)...)
	}

	// Validate notes (optional)
	if input.Notes != nil && len(*input.Notes) > MaxNotesLength {
		errs = append(errs, models.FieldError{Field: "notes", Message: "must be at most 500 characters"})
	}

	return errs
}

// validateOptionalLabel validates an optional label field (for updates).
func (s *Service) validateOptionalLabel(label string) []models.FieldError {
	if label == "" {
		return []models.FieldError{{Field: "label", Message: "cannot be empty"}}
	}
	if len(label) > MaxLabelLength {
		return []models.FieldError{{Field: "label", Message: "must be at most 80 characters"}}
	}
	return nil
}

// validateDaysOfWeek validates days of week array.
func (s *Service) validateDaysOfWeek(days []int, required bool) []models.FieldError {
	if len(days) == 0 {
		if required {
			return []models.FieldError{{Field: "daysOfWeek", Message: "is required"}}
		}
		return []models.FieldError{{Field: "daysOfWeek", Message: "cannot be empty"}}
	}
	for _, day := range days {
		if day < 1 || day > 7 {
			return []models.FieldError{{Field: "daysOfWeek", Message: "must contain values between 1 and 7"}}
		}
	}
	return nil
}

// validateOptionalArrivalTime validates an optional arrival time (for updates).
func (s *Service) validateOptionalArrivalTime(time string) []models.FieldError {
	if time == "" {
		return []models.FieldError{{Field: "preferredArrivalTimeLocal", Message: "cannot be empty"}}
	}
	if !timeHHMMRegex.MatchString(time) {
		return []models.FieldError{{Field: "preferredArrivalTimeLocal", Message: "must be in HH:mm format"}}
	}
	return nil
}

// validateLocation validates a commute location.
func (s *Service) validateLocation(loc *models.CommuteLocation, prefix string) []models.FieldError {
	var errs []models.FieldError

	if loc.Point.Lat < -90 || loc.Point.Lat > 90 {
		errs = append(errs, models.FieldError{
			Field:   prefix + ".point.lat",
			Message: "must be between -90 and 90",
		})
	}

	if loc.Point.Lon < -180 || loc.Point.Lon > 180 {
		errs = append(errs, models.FieldError{
			Field:   prefix + ".point.lon",
			Message: "must be between -180 and 180",
		})
	}

	return errs
}

// toAPICommute converts a domain Commute to an API Commute.
func (s *Service) toAPICommute(c *Commute) models.Commute {
	return models.Commute{
		ID:    c.ID,
		Label: c.Label,
		Origin: models.CommuteLocation{
			Point:   models.Point{Lat: c.Origin.Point.Lat, Lon: c.Origin.Point.Lon},
			Geohash: c.Origin.Geohash,
		},
		Destination: models.CommuteLocation{
			Point:   models.Point{Lat: c.Destination.Point.Lat, Lon: c.Destination.Point.Lon},
			Geohash: c.Destination.Geohash,
		},
		DaysOfWeek:                c.DaysOfWeek,
		PreferredArrivalTimeLocal: c.PreferredArrivalTimeLocal,
		Notes:                     c.Notes,
		CreatedAt:                 models.Timestamp(c.CreatedAt),
		UpdatedAt:                 models.Timestamp(c.UpdatedAt),
	}
}

// ValidationError represents validation errors.
type ValidationError struct {
	Errors []models.FieldError
}

func (e *ValidationError) Error() string {
	return "validation failed"
}
