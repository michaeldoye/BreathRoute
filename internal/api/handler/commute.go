package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/commute"
)

// CommuteHandler handles commute endpoints.
type CommuteHandler struct {
	service *commute.Service
}

// NewCommuteHandler creates a new CommuteHandler.
func NewCommuteHandler(service *commute.Service) *CommuteHandler {
	return &CommuteHandler{service: service}
}

// ListCommutes handles GET /v1/me/commutes - list saved commutes.
func (h *CommuteHandler) ListCommutes(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	// Parse limit from query params
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	commutes, err := h.service.List(r.Context(), userID, limit)
	if err != nil {
		response.InternalError(w, r, "failed to list commutes")
		return
	}

	response.JSON(w, r, http.StatusOK, commutes)
}

// CreateCommute handles POST /v1/me/commutes - create a saved commute.
func (h *CommuteHandler) CreateCommute(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	var input models.CommuteCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	result, err := h.service.Create(r.Context(), userID, &input)
	if err != nil {
		var validationErr *commute.ValidationError
		if errors.As(err, &validationErr) {
			response.BadRequest(w, r, "validation failed", validationErr.Errors)
			return
		}
		response.InternalError(w, r, "failed to create commute")
		return
	}

	location := fmt.Sprintf("/v1/me/commutes/%s", result.ID)
	response.Created(w, r, location, result)
}

// GetCommute handles GET /v1/me/commutes/{commuteId} - get a saved commute.
func (h *CommuteHandler) GetCommute(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	result, err := h.service.Get(r.Context(), userID, commuteID)
	if err != nil {
		if errors.Is(err, commute.ErrCommuteNotFound) {
			response.NotFound(w, r, "commute")
			return
		}
		response.InternalError(w, r, "failed to get commute")
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

// UpdateCommute handles PUT /v1/me/commutes/{commuteId} - update a saved commute.
func (h *CommuteHandler) UpdateCommute(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	var input models.CommuteUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	result, err := h.service.Update(r.Context(), userID, commuteID, &input)
	if err != nil {
		if errors.Is(err, commute.ErrCommuteNotFound) {
			response.NotFound(w, r, "commute")
			return
		}
		var validationErr *commute.ValidationError
		if errors.As(err, &validationErr) {
			response.BadRequest(w, r, "validation failed", validationErr.Errors)
			return
		}
		response.InternalError(w, r, "failed to update commute")
		return
	}

	response.JSON(w, r, http.StatusOK, result)
}

// DeleteCommute handles DELETE /v1/me/commutes/{commuteId} - delete a saved commute.
func (h *CommuteHandler) DeleteCommute(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "user not authenticated")
		return
	}

	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	err := h.service.Delete(r.Context(), userID, commuteID)
	if err != nil {
		if errors.Is(err, commute.ErrCommuteNotFound) {
			response.NotFound(w, r, "commute")
			return
		}
		response.InternalError(w, r, "failed to delete commute")
		return
	}

	response.NoContent(w, r)
}
