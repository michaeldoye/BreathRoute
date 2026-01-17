package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// DeviceHandler handles device endpoints.
type DeviceHandler struct{}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler() *DeviceHandler {
	return &DeviceHandler{}
}

// ListDevices handles GET /v1/me/devices - list registered devices.
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, _ *http.Request) {
	// TODO: Get actual devices from database
	now := models.Timestamp(time.Now())
	devices := models.PagedDevices{
		Items: []models.Device{
			{
				ID:          "dev_01HY4ABCDEF0123456789",
				Platform:    models.PushPlatformAPNS,
				TokenLast4:  strPtr("A1b2"),
				DeviceModel: strPtr("iPhone15,3"),
				OSVersion:   strPtr("iOS 18.2"),
				AppVersion:  strPtr("1.3.0"),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, http.StatusOK, devices)
}

// RegisterDevice handles POST /v1/me/devices - register or update device.
func (h *DeviceHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var input models.DeviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Validate input and save/update device in database
	now := models.Timestamp(time.Now())

	// Get last 4 characters of token
	var tokenLast4 *string
	if len(input.Token) >= 4 {
		last4 := input.Token[len(input.Token)-4:]
		tokenLast4 = &last4
	}

	device := models.Device{
		ID:          input.DeviceID,
		Platform:    input.Platform,
		TokenLast4:  tokenLast4,
		DeviceModel: input.DeviceModel,
		OSVersion:   input.OSVersion,
		AppVersion:  input.AppVersion,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// TODO: Check if device already exists to return 200 vs 201
	location := fmt.Sprintf("/v1/me/devices/%s", input.DeviceID)
	response.Created(w, location, device)
}

// UnregisterDevice handles DELETE /v1/me/devices/{deviceId} - unregister device.
func (h *DeviceHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		response.BadRequest(w, r, "deviceId is required", nil)
		return
	}

	// TODO: Delete device from database
	response.NoContent(w)
}
