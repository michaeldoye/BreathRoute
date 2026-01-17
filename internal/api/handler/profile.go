package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/user"
)

// ProfileHandler handles user profile endpoints.
type ProfileHandler struct {
	userService *user.Service
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(userService *user.Service) *ProfileHandler {
	return &ProfileHandler{userService: userService}
}

// GetProfile handles GET /v1/me/profile - get user's sensitivity profile.
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	profile, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

// UpsertProfile handles PUT /v1/me/profile - create or update profile.
func (h *ProfileHandler) UpsertProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	var input models.ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	fieldErrors := validateProfileInput(&input)
	if len(fieldErrors) > 0 {
		response.BadRequest(w, r, "validation failed", fieldErrors)
		return
	}

	profile, err := h.userService.UpsertProfile(r.Context(), userID, &input)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

// validateProfileInput validates profile input and returns any field errors.
func validateProfileInput(input *models.ProfileInput) []models.FieldError {
	var fieldErrors []models.FieldError

	// Validate exposure weights (must be in range [0, 1])
	fieldErrors = validateWeight(fieldErrors, input.Weights.NO2, "weights.no2")
	fieldErrors = validateWeight(fieldErrors, input.Weights.PM25, "weights.pm25")
	fieldErrors = validateWeight(fieldErrors, input.Weights.O3, "weights.o3")
	fieldErrors = validateWeight(fieldErrors, input.Weights.Pollen, "weights.pollen")

	// Validate route constraints
	fieldErrors = validateConstraints(fieldErrors, input.Constraints)

	return fieldErrors
}

// validateWeight validates a weight field is in range [0, 1].
func validateWeight(errs []models.FieldError, value float64, field string) []models.FieldError {
	if value < 0 || value > 1 {
		errs = append(errs, models.FieldError{
			Field:   field,
			Message: "must be between 0 and 1",
		})
	}
	return errs
}

// validateConstraints validates route constraint fields.
func validateConstraints(errs []models.FieldError, constraints models.RouteConstraints) []models.FieldError {
	if constraints.MaxExtraMinutesVsFastest != nil {
		if *constraints.MaxExtraMinutesVsFastest < 0 || *constraints.MaxExtraMinutesVsFastest > 120 {
			errs = append(errs, models.FieldError{
				Field:   "constraints.maxExtraMinutesVsFastest",
				Message: "must be between 0 and 120",
			})
		}
	}
	if constraints.MaxTransfers != nil {
		if *constraints.MaxTransfers < 0 || *constraints.MaxTransfers > 10 {
			errs = append(errs, models.FieldError{
				Field:   "constraints.maxTransfers",
				Message: "must be between 0 and 10",
			})
		}
	}
	return errs
}
