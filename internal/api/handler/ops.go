// Package handler provides HTTP handlers for the BreatheRoute API.
package handler

import (
	"net/http"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// OpsHandler handles operational endpoints.
type OpsHandler struct {
	version   string
	buildTime string
}

// NewOpsHandler creates a new OpsHandler.
func NewOpsHandler(version, buildTime string) *OpsHandler {
	return &OpsHandler{
		version:   version,
		buildTime: buildTime,
	}
}

// HealthCheck handles GET /v1/ops/health - liveness check.
func (h *OpsHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	health := models.Health{
		Status: models.HealthStatusOK,
		Time:   models.Timestamp(time.Now()),
		Details: map[string]interface{}{
			"version":   h.version,
			"buildTime": h.buildTime,
		},
	}
	response.JSON(w, http.StatusOK, health)
}

// ReadinessCheck handles GET /v1/ops/ready - readiness check.
func (h *OpsHandler) ReadinessCheck(w http.ResponseWriter, _ *http.Request) {
	// TODO: Add actual dependency checks (database, cache, etc.)
	health := models.Health{
		Status: models.HealthStatusOK,
		Time:   models.Timestamp(time.Now()),
	}
	response.JSON(w, http.StatusOK, health)
}

// SystemStatus handles GET /v1/ops/status - provider and subsystem status.
func (h *OpsHandler) SystemStatus(w http.ResponseWriter, _ *http.Request) {
	// TODO: Add actual subsystem and provider status checks
	now := models.Timestamp(time.Now())
	status := models.SystemStatus{
		Status: models.HealthStatusOK,
		Time:   now,
		Subsystems: []models.SubsystemStatus{
			{Name: "cloud-sql", Status: models.HealthStatusOK},
			{Name: "redis", Status: models.HealthStatusOK},
		},
		Providers: []models.ProviderStatus{
			{Provider: "luchtmeetnet", Status: models.HealthStatusOK, LastSuccessAt: &now},
			{Provider: "ns", Status: models.HealthStatusOK, LastSuccessAt: &now},
		},
	}
	response.JSON(w, http.StatusOK, status)
}
