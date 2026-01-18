package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/device"
)

// DeviceHandler handles device endpoints.
type DeviceHandler struct {
	service *device.Service
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(service *device.Service) *DeviceHandler {
	return &DeviceHandler{service: service}
}

// ListDevices handles GET /v1/me/devices - list registered devices.
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "authentication required")
		return
	}

	devices, err := h.service.List(r.Context(), userID, 50)
	if err != nil {
		response.InternalError(w, r, "failed to list devices")
		return
	}

	response.JSON(w, http.StatusOK, devices)
}

// RegisterDevice handles POST /v1/me/devices - register or update device.
func (h *DeviceHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "authentication required")
		return
	}

	var input models.DeviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate input
	if fieldErrors := h.validateRegisterInput(&input); len(fieldErrors) > 0 {
		response.BadRequest(w, r, "validation failed", fieldErrors)
		return
	}

	result, created, err := h.service.Register(r.Context(), userID, &input)
	if err != nil {
		response.InternalError(w, r, "failed to register device")
		return
	}

	location := fmt.Sprintf("/v1/me/devices/%s", input.DeviceID)
	if created {
		response.Created(w, location, result)
	} else {
		w.Header().Set("Location", location)
		response.JSON(w, http.StatusOK, result)
	}
}

// UnregisterDevice handles DELETE /v1/me/devices/{deviceId} - unregister device.
func (h *DeviceHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "authentication required")
		return
	}

	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		response.BadRequest(w, r, "deviceId is required", nil)
		return
	}

	err := h.service.Unregister(r.Context(), userID, deviceID)
	if err != nil {
		if errors.Is(err, device.ErrDeviceNotFound) {
			response.NotFound(w, r, "device not found")
			return
		}
		response.InternalError(w, r, "failed to unregister device")
		return
	}

	response.NoContent(w)
}

// validateRegisterInput validates the device registration input.
func (h *DeviceHandler) validateRegisterInput(input *models.DeviceRegisterRequest) []models.FieldError {
	var errs []models.FieldError

	if input.DeviceID == "" {
		errs = append(errs, models.FieldError{Field: "deviceId", Message: "is required"})
	}

	if input.Platform == "" {
		errs = append(errs, models.FieldError{Field: "platform", Message: "is required"})
	} else if input.Platform != models.PushPlatformFCM && input.Platform != models.PushPlatformAPNS {
		errs = append(errs, models.FieldError{Field: "platform", Message: "must be FCM or APNS"})
	}

	if input.Token == "" {
		errs = append(errs, models.FieldError{Field: "token", Message: "is required"})
	} else if len(input.Token) < 16 {
		errs = append(errs, models.FieldError{Field: "token", Message: "must be at least 16 characters"})
	}

	return errs
}
