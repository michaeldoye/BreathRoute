// Package handler provides HTTP handlers for the BreatheRoute API.
package handler

import (
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

// OpsHandler handles operational endpoints.
type OpsHandler struct {
	version          string
	buildTime        string
	providerRegistry *resilience.Registry
}

// NewOpsHandler creates a new OpsHandler.
func NewOpsHandler(version, buildTime string) *OpsHandler {
	return &OpsHandler{
		version:   version,
		buildTime: buildTime,
	}
}

// WithProviderRegistry sets the provider registry for health reporting.
func (h *OpsHandler) WithProviderRegistry(registry *resilience.Registry) *OpsHandler {
	h.providerRegistry = registry
	return h
}

// HealthCheck handles GET /v1/ops/health - liveness check.
func (h *OpsHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
func (h *OpsHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual dependency checks (database, cache, etc.)
	health := models.Health{
		Status: models.HealthStatusOK,
		Time:   models.Timestamp(time.Now()),
	}
	response.JSON(w, http.StatusOK, health)
}

// SystemStatus handles GET /v1/ops/status - provider and subsystem status.
func (h *OpsHandler) SystemStatus(w http.ResponseWriter, r *http.Request) {
	now := models.Timestamp(time.Now())

	// Get provider status from registry
	providers := h.getProviderStatuses()

	// Determine overall status based on provider health
	overallStatus := models.HealthStatusOK
	for _, p := range providers {
		if p.Status == models.HealthStatusDegraded {
			overallStatus = models.HealthStatusDegraded
		} else if p.Status == models.HealthStatusFail {
			overallStatus = models.HealthStatusFail
			break
		}
	}

	status := models.SystemStatus{
		Status: overallStatus,
		Time:   now,
		Subsystems: []models.SubsystemStatus{
			{Name: "cloud-sql", Status: models.HealthStatusOK},
			{Name: "redis", Status: models.HealthStatusOK},
		},
		Providers: providers,
	}
	response.JSON(w, http.StatusOK, status)
}

// getProviderStatuses returns the status of all registered providers.
func (h *OpsHandler) getProviderStatuses() []models.ProviderStatus {
	if h.providerRegistry == nil {
		return []models.ProviderStatus{}
	}

	healthList := h.providerRegistry.GetAllHealth()
	statuses := make([]models.ProviderStatus, 0, len(healthList))

	for _, health := range healthList {
		ps := models.ProviderStatus{
			Provider: health.Name,
			Status:   h.mapCircuitStateToHealth(health.CircuitState),
		}

		if health.LastSuccessAt != nil {
			ts := models.Timestamp(*health.LastSuccessAt)
			ps.LastSuccessAt = &ts
		}

		if health.LastFailureAt != nil {
			ts := models.Timestamp(*health.LastFailureAt)
			ps.LastFailureAt = &ts
		}

		if health.LastError != "" {
			ps.Message = &health.LastError
		}

		statuses = append(statuses, ps)
	}

	return statuses
}

// mapCircuitStateToHealth maps circuit breaker state to health status.
func (h *OpsHandler) mapCircuitStateToHealth(state gobreaker.State) models.HealthStatus {
	switch state {
	case gobreaker.StateClosed:
		return models.HealthStatusOK
	case gobreaker.StateHalfOpen:
		return models.HealthStatusDegraded
	case gobreaker.StateOpen:
		return models.HealthStatusFail
	default:
		return models.HealthStatusOK
	}
}
