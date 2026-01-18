package handler

import (
	"net/http"

	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/featureflags"
)

// FeatureFlagsHandler handles feature flag endpoints.
type FeatureFlagsHandler struct {
	service *featureflags.Service
}

// NewFeatureFlagsHandler creates a new FeatureFlagsHandler.
func NewFeatureFlagsHandler(service *featureflags.Service) *FeatureFlagsHandler {
	return &FeatureFlagsHandler{service: service}
}

// ListFeatureFlags handles GET /v1/admin/flags - list all feature flags.
func (h *FeatureFlagsHandler) ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement listing feature flags
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"flags": []interface{}{},
	})
}

// UpsertFeatureFlags handles PUT /v1/admin/flags - update feature flags.
func (h *FeatureFlagsHandler) UpsertFeatureFlags(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement upserting feature flags
	response.NoContent(w)
}

// InvalidateCache handles POST /v1/admin/flags/invalidate - invalidate flag cache.
func (h *FeatureFlagsHandler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement cache invalidation
	response.NoContent(w)
}
