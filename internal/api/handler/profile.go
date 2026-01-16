package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// ProfileHandler handles user profile endpoints.
type ProfileHandler struct{}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler() *ProfileHandler {
	return &ProfileHandler{}
}

// GetProfile handles GET /v1/me/profile - get user's sensitivity profile.
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Get actual profile from database
	profile := models.Profile{
		Weights: models.ExposureWeights{
			NO2:    0.6,
			PM25:   0.2,
			O3:     0.1,
			Pollen: 0.1,
		},
		Constraints: models.RouteConstraints{
			AvoidMajorRoads: true,
		},
		CreatedAt: models.Timestamp(time.Now().AddDate(0, -1, 0)),
		UpdatedAt: models.Timestamp(time.Now().AddDate(0, 0, -7)),
	}
	response.JSON(w, http.StatusOK, profile)
}

// UpsertProfile handles PUT /v1/me/profile - create or update profile.
func (h *ProfileHandler) UpsertProfile(w http.ResponseWriter, r *http.Request) {
	var input models.ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Validate input using validator
	// TODO: Save profile to database

	profile := models.Profile{
		Weights:     input.Weights,
		Constraints: input.Constraints,
		CreatedAt:   models.Timestamp(time.Now()),
		UpdatedAt:   models.Timestamp(time.Now()),
	}
	response.JSON(w, http.StatusOK, profile)
}
