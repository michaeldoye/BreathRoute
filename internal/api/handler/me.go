package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// MeHandler handles user account endpoints.
type MeHandler struct{}

// NewMeHandler creates a new MeHandler.
func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

// GetMe handles GET /v1/me - get current user account summary.
func (h *MeHandler) GetMe(w http.ResponseWriter, _ *http.Request) {
	// TODO: Get actual user from context (after auth middleware)
	me := models.Me{
		UserID:    "usr_01HY1A2B3C4D5E6F7G8H9J",
		Locale:    "nl-NL",
		CreatedAt: models.Timestamp(time.Now().AddDate(0, -1, 0)),
	}
	response.JSON(w, http.StatusOK, me)
}

// GetConsents handles GET /v1/me/consents - get consent states.
func (h *MeHandler) GetConsents(w http.ResponseWriter, _ *http.Request) {
	// TODO: Get actual consents from database
	consents := models.Consents{
		Analytics:         true,
		Marketing:         false,
		PushNotifications: true,
		UpdatedAt:         models.Timestamp(time.Now().AddDate(0, 0, -7)),
	}
	response.JSON(w, http.StatusOK, consents)
}

// UpdateConsents handles PUT /v1/me/consents - update consent states.
func (h *MeHandler) UpdateConsents(w http.ResponseWriter, r *http.Request) {
	var input models.ConsentsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Update consents in database
	consents := models.Consents{
		Analytics:         true,
		Marketing:         false,
		PushNotifications: true,
		UpdatedAt:         models.Timestamp(time.Now()),
	}

	if input.Analytics != nil {
		consents.Analytics = *input.Analytics
	}
	if input.Marketing != nil {
		consents.Marketing = *input.Marketing
	}
	if input.PushNotifications != nil {
		consents.PushNotifications = *input.PushNotifications
	}

	response.JSON(w, http.StatusOK, consents)
}
