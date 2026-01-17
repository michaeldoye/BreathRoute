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
	MaxLabelLength  = 80
	MaxNotesLength  = 500
	DefaultTimezone = "Europe/Amsterdam"
)

// dayNames maps ISO weekday numbers (1=Monday, 7=Sunday) to day names.
var dayNames = map[int]string{
	1: "Monday",
	2: "Tuesday",
	3: "Wednesday",
	4: "Thursday",
	5: "Friday",
	6: "Saturday",
	7: "Sunday",
}

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

	// Determine timezone (use default if not provided)
	timezone := DefaultTimezone
	if input.Timezone != nil && *input.Timezone != "" {
		timezone = *input.Timezone
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
		Timezone:                  timezone,
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
	if input.Timezone != nil {
		commute.Timezone = *input.Timezone
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
	errs = append(errs, s.validateDaysOfWeek(input.DaysOfWeek, true)...)

	// Validate preferred arrival time
	if input.PreferredArrivalTimeLocal == "" {
		errs = append(errs, models.FieldError{Field: "preferredArrivalTimeLocal", Message: "is required"})
	} else if !timeHHMMRegex.MatchString(input.PreferredArrivalTimeLocal) {
		errs = append(errs, models.FieldError{Field: "preferredArrivalTimeLocal", Message: "must be in HH:mm format"})
	}

	// Validate timezone (optional)
	if input.Timezone != nil && *input.Timezone != "" {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			errs = append(errs, models.FieldError{Field: "timezone", Message: "must be a valid IANA timezone identifier"})
		}
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

	// Validate timezone (optional)
	if input.Timezone != nil && *input.Timezone != "" {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			errs = append(errs, models.FieldError{Field: "timezone", Message: "must be a valid IANA timezone identifier"})
		}
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
func (s *Service) validateOptionalArrivalTime(t string) []models.FieldError {
	if t == "" {
		return []models.FieldError{{Field: "preferredArrivalTimeLocal", Message: "cannot be empty"}}
	}
	if !timeHHMMRegex.MatchString(t) {
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
	schedule := s.buildSchedule(c)

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
		Schedule:  schedule,
		Notes:     c.Notes,
		CreatedAt: models.Timestamp(c.CreatedAt),
		UpdatedAt: models.Timestamp(c.UpdatedAt),
	}
}

// buildSchedule builds a normalized CommuteSchedule from domain data.
func (s *Service) buildSchedule(c *Commute) models.CommuteSchedule {
	// Build day names from day numbers
	names := make([]string, 0, len(c.DaysOfWeek))
	for _, day := range c.DaysOfWeek {
		if name, ok := dayNames[day]; ok {
			names = append(names, name)
		}
	}

	schedule := models.CommuteSchedule{
		DaysOfWeek:  c.DaysOfWeek,
		DayNames:    names,
		ArrivalTime: c.PreferredArrivalTimeLocal,
		Timezone:    c.Timezone,
	}

	// Load timezone for calculations
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	// Calculate IsActiveToday and NextOccurrence
	now := time.Now().In(loc)
	todayWeekday := isoWeekday(now.Weekday())
	schedule.IsActiveToday = containsDay(c.DaysOfWeek, todayWeekday)

	// Find next occurrence within 7 days
	if next := s.findNextOccurrence(c, loc, now); next != nil {
		formatted := next.Format(time.RFC3339)
		schedule.NextOccurrence = &formatted
	}

	return schedule
}

// findNextOccurrence finds the next scheduled commute time within 7 days.
func (s *Service) findNextOccurrence(c *Commute, loc *time.Location, now time.Time) *time.Time {
	if len(c.DaysOfWeek) == 0 {
		return nil
	}

	// Parse arrival time
	parts := parseTimeHHMM(c.PreferredArrivalTimeLocal)
	if parts == nil {
		return nil
	}
	hour, minute := parts[0], parts[1]

	// Check each day for the next 7 days
	for i := 0; i < 7; i++ {
		checkDate := now.AddDate(0, 0, i)
		checkWeekday := isoWeekday(checkDate.Weekday())

		if containsDay(c.DaysOfWeek, checkWeekday) {
			// Create the candidate time on this day
			candidate := time.Date(
				checkDate.Year(), checkDate.Month(), checkDate.Day(),
				hour, minute, 0, 0, loc,
			)

			// If it's today but the time has passed, skip to next occurrence
			if i == 0 && candidate.Before(now) {
				continue
			}

			return &candidate
		}
	}

	return nil
}

// isoWeekday converts Go's time.Weekday (0=Sunday) to ISO weekday (1=Monday, 7=Sunday).
func isoWeekday(w time.Weekday) int {
	if w == time.Sunday {
		return 7
	}
	return int(w)
}

// containsDay checks if a day number is in the list.
func containsDay(days []int, day int) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

// parseTimeHHMM parses a time string in HH:mm format and returns [hour, minute].
func parseTimeHHMM(t string) []int {
	if !timeHHMMRegex.MatchString(t) {
		return nil
	}

	// Parse using time.Parse with a reference format
	parsed, err := time.Parse("15:04", t)
	if err != nil {
		return nil
	}

	return []int{parsed.Hour(), parsed.Minute()}
}

// ValidationError represents validation errors.
type ValidationError struct {
	Errors []models.FieldError
}

func (e *ValidationError) Error() string {
	return "validation failed"
}
