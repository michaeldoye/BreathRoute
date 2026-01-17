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

// MeHandler handles user account endpoints.
type MeHandler struct {
	userService *user.Service
}

// NewMeHandler creates a new MeHandler.
func NewMeHandler(userService *user.Service) *MeHandler {
	return &MeHandler{userService: userService}
}

// GetMe handles GET /v1/me - get current user account summary.
func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	me, err := h.userService.GetMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, me)
}

// UpdateMe handles PUT /v1/me - update current user settings.
func (h *MeHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	var input models.MeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate units if provided
	if input.Units != nil {
		if *input.Units != models.UnitsMetric && *input.Units != models.UnitsImperial {
			response.BadRequest(w, r, "invalid units value", []models.FieldError{
				{Field: "units", Message: "must be METRIC or IMPERIAL"},
			})
			return
		}
	}

	me, err := h.userService.UpdateMe(r.Context(), userID, &input)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, me)
}

// GetConsents handles GET /v1/me/consents - get consent states.
func (h *MeHandler) GetConsents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	consents, err := h.userService.GetConsents(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, consents)
}

// UpdateConsents handles PUT /v1/me/consents - update consent states.
func (h *MeHandler) UpdateConsents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	var input models.ConsentsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	consents, err := h.userService.UpdateConsents(r.Context(), userID, &input)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			response.NotFound(w, r, "user")
			return
		}
		response.InternalError(w, r, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, consents)
}
