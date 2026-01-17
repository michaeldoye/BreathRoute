package handler

import (
	"encoding/json"
	"net/http"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/featureflags"
)

// FeatureFlagsHandler handles feature flag admin endpoints.
type FeatureFlagsHandler struct {
	service *featureflags.Service
}

// NewFeatureFlagsHandler creates a new FeatureFlagsHandler.
func NewFeatureFlagsHandler(service *featureflags.Service) *FeatureFlagsHandler {
	return &FeatureFlagsHandler{service: service}
}

// ListFeatureFlags handles GET /v1/admin/feature-flags - list all feature flags.
func (h *FeatureFlagsHandler) ListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	flags := h.service.GetAllFlags(r.Context())

	// Convert to response format
	items := make([]featureflags.Flag, 0, len(flags))
	for _, flag := range flags {
		items = append(items, *flag)
	}

	resp := featureflags.FlagList{Items: items}
	response.JSON(w, r, http.StatusOK, resp)
}

// UpsertFeatureFlags handles PUT /v1/admin/feature-flags - update feature flags.
func (h *FeatureFlagsHandler) UpsertFeatureFlags(w http.ResponseWriter, r *http.Request) {
	var input featureflags.FlagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate input
	if len(input.Updates) == 0 {
		response.BadRequest(w, r, "at least one update is required", []models.FieldError{
			{Field: "updates", Message: "cannot be empty"},
		})
		return
	}

	if len(input.Reason) < 3 {
		response.BadRequest(w, r, "reason is required", []models.FieldError{
			{Field: "reason", Message: "must be at least 3 characters"},
		})
		return
	}

	// Convert updates to flags
	flagsToUpdate := make([]*featureflags.Flag, 0, len(input.Updates))
	for _, update := range input.Updates {
		if update.Key == "" {
			response.BadRequest(w, r, "flag key is required", []models.FieldError{
				{Field: "updates[].key", Message: "cannot be empty"},
			})
			return
		}
		flagsToUpdate = append(flagsToUpdate, &featureflags.Flag{
			Key:   update.Key,
			Value: update.Value,
		})
	}

	// Update flags
	if err := h.service.SetFlags(r.Context(), flagsToUpdate); err != nil {
		response.InternalError(w, r, "failed to update feature flags")
		return
	}

	// TODO: Audit log the change with input.Reason

	// Return updated flags
	flags := h.service.GetAllFlags(r.Context())
	items := make([]featureflags.Flag, 0, len(flags))
	for _, flag := range flags {
		items = append(items, *flag)
	}

	resp := featureflags.FlagList{Items: items}
	response.JSON(w, r, http.StatusOK, resp)
}

// InvalidateCache handles POST /v1/admin/feature-flags/invalidate - clear the cache.
func (h *FeatureFlagsHandler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	h.service.InvalidateCache()
	response.NoContent(w, r)
}
